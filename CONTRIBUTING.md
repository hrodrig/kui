# Contributing

PRs against `develop`. Releases from `main`.

1. `make release-check` must pass locally before tagging
2. Follow existing patterns (kiko, gghstats, pgwd)
3. Keep gocyclo ≤ 14; coverage target ≥ 80% (gate is `COVERAGE_MIN`, currently 4% until tests grow)
4. After `go mod tidy` or Dependabot bumps: `go get golang.org/x/net@v0.55.0 golang.org/x/crypto@v0.52.0`, then `make check-x-net-pin check-x-crypto-pin`
5. Sign commits
