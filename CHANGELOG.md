# Changelog

All notable changes to **kui** are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.3.2] - 2026-06-21

### Added

- **Log format** — each line now includes `- kui -` before level for multi-service log
  federation (`2026-06-22T01:40:25Z  - kui - INFO  ...`).
- **Config search path** — `./kui.yml` → `~/.kui/kui.yml` → `/etc/kui/kui.yml`;
  startup logs `Using config file: <path>` or `Using config file: none, default settings`.
- **Log level flag** — `--log-level` CLI flag and `KUI_LOG_LEVEL` env var
  (trace, debug, info, warn, error, fatal, off).
- **Log level banner** — `Debug level set to: DEBUG` on startup.
- **Database path logged** — shows `database path: ./data/kui.db` on startup.

### Changed

- Config search path: removed `./configs/`, now follows same pattern as kiko.

## [0.3.1] - 2026-06-20

### Added

- **Health endpoints** — `GET /api/v1/healthz` (liveness) and
  `GET /api/v1/readyz` (readiness with DB ping) for Kubernetes probes.

## [0.3.0] - 2026-06-20

### Added

- **Test coverage** — auth 82%, store 87%, i18n 91%, kikoclient 86%, config 96%, version 100%.
  `make cover-check` gates at ≥ 80 % with `COVERAGE_MIN=80`.
- **Security workflows** — standalone `security.yml` (govulncheck + grype on Docker image),
  `codeql.yml` (CodeQL static analysis), and `codecov.yml` (informational uploads).
- **Dependency pins** — explicit `golang.org/x/net@v0.55.0`, `golang.org/x/crypto@v0.52.0`,
  `go-viper/mapstructure/v2@v2.4.0`, enforced by Makefile guards and GoReleaser hooks.
- **`gocyclo` rule** — cyclomatic complexity ≤ 14 (excludes `*_templ.go`).
- **`Dockerfile.release`** — distroless runtime image for GoReleaser.
- **`.dockerignore`** — allowlist build context (go.mod, go.sum, cmd/, internal/ only).
- **`.cursor/rules/`** — dedicated rules for gocyclo, coverage, security, release-tests.
- **`CODE_OF_CONDUCT.md`** — Contributor Covenant v2.1.
- **`codecov.yml`** and `SECURITY.md` — project health documentation.

### Changed

- **Makefile** — split `COVER_PKGS` (gate) from `TEST_PKGS` (all). New targets:
  `cover-check`, `gocyclo`, `govulncheck`, `grype`, `security`, `tools`, `docker-*`.
  `release-check` gates all quality checks.
- **README badges** — point CI, Security, CodeQL, Codecov to live workflows.
- **Refactored** `getDashboard` → `fillDashboardKiko` helper to reduce cyclomatic complexity.
- **`.gitignore`** — allowlist for project assets; deny `*.db`, `data/*.db`, `coverage.out`.

## [0.2.0] - 2026-06-20

### Added

- First tagged release — all features from 0.1.0.

## [0.1.0] - 2026-06-20

### Added

- **Analytics dashboard** — KPIs, daily timeline, traffic channels, top paths and referrers (7 / 30 / 90 day ranges).
- **kiko stats client** — server-side Query API integration (`KIKO_URL`, `KIKO_API_KEY`).
- **Authentication** — bcrypt passwords, cookie sessions, optional “remember me”.
- **User management** — admin CRUD, roles (`admin` / `user`), per-user host allowlists.
- **Optional 2FA** — TOTP setup/login flow; admin can reset another user’s 2FA.
- **i18n** — English, Spanish, French, German, Portuguese (BR); query param, cookie, and `Accept-Language`.
- **Light / dark theme** — toggle with `localStorage` persistence; charts recolor on theme change.
- **Version badges** — live **kiko** and **kui** build info in the header (`GET /api/v1/version`).
- **UI stack** — templ + HTMX, custom `kui.css`, vendored Bootstrap 5.3 and Chart.js 4.4.
- **Docker** — multi-stage distroless image.
- **Dev tooling** — `make seed-kiko` (API hits) and `make seed-kiko-history` (backdated SQLite demo data).
