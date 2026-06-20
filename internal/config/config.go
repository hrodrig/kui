package config

import (
	"fmt"
	"strings"

	"github.com/hrodrig/kui/internal/log"
	"github.com/spf13/viper"
)

type Config struct {
	Listen         string      `mapstructure:"listen"`
	LogLevel       string      `mapstructure:"log_level"`
	Database       DatabaseCfg `mapstructure:"database"`
	Kiko           KikoCfg     `mapstructure:"kiko"`
	Session        SessionCfg  `mapstructure:"session"`
	Admin          AdminCfg    `mapstructure:"admin"`
	DefaultLocale  string      `mapstructure:"default_locale"`
	EnabledLocales []string    `mapstructure:"enabled_locales"`
	Log            *log.Logger
}

type DatabaseCfg struct {
	Path string `mapstructure:"path"`
}

type KikoCfg struct {
	URL    string `mapstructure:"url"`
	APIKey string `mapstructure:"api_key"`
}

type SessionCfg struct {
	CookieName    string `mapstructure:"cookie_name"`
	TTLHours      int    `mapstructure:"ttl_hours"`
	ShortTTLHours int    `mapstructure:"short_ttl_hours"`
	Secure        bool   `mapstructure:"secure"`
}

type AdminCfg struct {
	Email    string `mapstructure:"email"`
	Password string `mapstructure:"password"`
}

func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetDefault("listen", ":3000")
	v.SetDefault("log_level", "info")
	v.SetDefault("database.path", "./data/kui.db")
	v.SetDefault("kiko.url", "http://127.0.0.1:8080")
	v.SetDefault("session.cookie_name", "kui_session")
	v.SetDefault("session.ttl_hours", 168)
	v.SetDefault("session.short_ttl_hours", 8)
	v.SetDefault("session.secure", false)
	v.SetDefault("admin.email", "admin@localhost")
	v.SetDefault("default_locale", "en")

	bindEnv(v)
	if path != "" {
		v.SetConfigFile(path)
		if err := v.ReadInConfig(); err != nil {
			return nil, fmt.Errorf("read config: %w", err)
		}
	} else {
		v.SetConfigName("kui")
		v.AddConfigPath(".")
		v.AddConfigPath("./configs")
		_ = v.ReadInConfig()
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if cfg.EnabledLocales == nil {
		cfg.EnabledLocales = []string{}
	}

	level, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		return nil, err
	}
	cfg.Log = log.New(nil, level)
	return &cfg, nil
}

func bindEnv(v *viper.Viper) {
	v.SetEnvPrefix("kui")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.BindEnv("listen", "KUI_LISTEN")
	v.BindEnv("log_level", "KUI_LOG_LEVEL")
	v.BindEnv("database.path", "KUI_DATABASE_PATH")
	v.BindEnv("kiko.url", "KIKO_URL")
	v.BindEnv("kiko.api_key", "KIKO_API_KEY")
	v.BindEnv("session.cookie_name", "KUI_SESSION_COOKIE")
	v.BindEnv("session.ttl_hours", "KUI_SESSION_TTL_HOURS")
	v.BindEnv("session.short_ttl_hours", "KUI_SESSION_SHORT_TTL_HOURS")
	v.BindEnv("session.secure", "KUI_SESSION_SECURE")
	v.BindEnv("admin.email", "KUI_ADMIN_EMAIL")
	v.BindEnv("admin.password", "KUI_ADMIN_PASSWORD")
	v.BindEnv("default_locale", "KUI_DEFAULT_LOCALE")
	v.BindEnv("enabled_locales", "KUI_ENABLED_LOCALES")
}
