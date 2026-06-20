# Changelog

All notable changes to **kui** are documented here.

Format based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

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
