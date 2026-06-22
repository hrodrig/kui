package cli

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hrodrig/kui/internal/config"
	"github.com/hrodrig/kui/internal/server"
	"github.com/hrodrig/kui/internal/store"
	"github.com/hrodrig/kui/internal/version"
	"github.com/spf13/cobra"
)

func Execute() int {
	root := newRootCmd()
	if err := root.Execute(); err != nil {
		return 1
	}
	return 0
}

func newRootCmd() *cobra.Command {
	var cfgFile string

	root := &cobra.Command{
		Use:     "kui",
		Short:   "Analytics UI for kiko",
		Version: version.Version,
	}
	root.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default ./kui.yml)")

	serve := &cobra.Command{
		Use:   "serve",
		Short: "Start the web UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			logLevel, _ := cmd.Flags().GetString("log-level")
			return serveCmd(cfgFile, logLevel)
		},
	}
	serve.Flags().String("listen", "", "HTTP listen address (overrides config)")
	serve.Flags().String("log-level", "", "Log level (trace, debug, info, warn, error, fatal, off)")
	root.AddCommand(serve)

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version info",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Println(version.BuildInfo())
		},
	}
	root.AddCommand(versionCmd)

	return root
}

func serveCmd(cfgFile, logLevel string) error {
	cfg, err := config.Load(cfgFile, logLevel)
	if err != nil {
		return err
	}

	st, err := store.Open(cfg.Database.Path)
	if err != nil {
		return err
	}
	defer st.Close()

	ctx := context.Background()
	if err := server.BootstrapAdmin(ctx, st, cfg); err != nil {
		return err
	}

	srv, err := server.New(cfg, st)
	if err != nil {
		return err
	}

	httpSrv := &http.Server{
		Addr:    cfg.Listen,
		Handler: srv.Handler(),
	}

	runCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-runCtx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutdownCtx)
	}()

	cfg.Log.Info("kui v%s starting on %s (db=%s, kiko=%s)",
		version.Version, cfg.Listen, cfg.Database.Path, cfg.Kiko.URL)

	if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	cfg.Log.Info("kui stopped")
	return nil
}
