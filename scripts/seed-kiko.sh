#!/usr/bin/env sh
# Inject sample pageview hits into a local kiko instance (for kui dashboard dev).
set -eu

KIKO_URL="${KIKO_URL:-http://127.0.0.1:8080}"
KIKO_API_KEY="${KIKO_API_KEY:-local-dev-key}"

# Comma- or space-separated host list. KIKO_SEED_HOST (singular) still works.
if [ -n "${KIKO_SEED_HOSTS:-}" ]; then
	SEED_HOSTS_RAW="$KIKO_SEED_HOSTS"
elif [ -n "${KIKO_SEED_HOST:-}" ]; then
	SEED_HOSTS_RAW="$KIKO_SEED_HOST"
else
	SEED_HOSTS_RAW="localhost,gghstats.com,kzero.dev"
fi

UA_CHROME_MAC='Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
UA_FIREFOX_LINUX='Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0'
UA_SAFARI_IPHONE='Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1'
UA_EDGE_WIN='Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0'
UA_CHROME_ANDROID='Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36'

HITS_SENT=0

hit() {
	host="$1"
	ua="$2"
	path="$3"
	referrer="$4"
	title="$5"
	width="$6"
	curl -fsS -o /dev/null -X POST "$KIKO_URL/hit" \
		-H "Content-Type: application/json" \
		-H "User-Agent: $ua" \
		-d "{\"host\":\"$host\",\"path\":\"$path\",\"referrer\":\"$referrer\",\"title\":\"$title\",\"width\":$width}"
	HITS_SENT=$((HITS_SENT + 1))
}

seed_localhost() {
	h="$1"
	hit "$h" "$UA_CHROME_MAC"     "/"              "https://www.google.com/"         "Home"           1920
	hit "$h" "$UA_FIREFOX_LINUX"  "/"              "https://news.ycombinator.com/"   "Home"           1440
	hit "$h" "$UA_SAFARI_IPHONE"  "/"              ""                                "Home"           390
	hit "$h" "$UA_EDGE_WIN"       "/"              "https://t.co/abc123"             "Home"           1366
	hit "$h" "$UA_CHROME_ANDROID" "/"              "https://www.reddit.com/r/golang" "Home"           412
	hit "$h" "$UA_CHROME_MAC"     "/blog/kiko-launch"    "https://www.google.com/"  "Kiko launch"    1920
	hit "$h" "$UA_CHROME_MAC"     "/blog/kiko-launch"    "https://www.google.com/"  "Kiko launch"    1920
	hit "$h" "$UA_FIREFOX_LINUX"  "/blog/kiko-launch"    "https://dev.to/"          "Kiko launch"    1440
	hit "$h" "$UA_SAFARI_IPHONE"  "/blog/htmx-dashboard" "https://twitter.com/"     "HTMX dashboard" 390
	hit "$h" "$UA_EDGE_WIN"       "/blog/htmx-dashboard" "https://www.bing.com/"    "HTMX dashboard" 1366
	hit "$h" "$UA_CHROME_ANDROID" "/blog/self-hosting"   ""                         "Self-hosting"   412
	hit "$h" "$UA_CHROME_MAC"     "/pricing?utm_source=newsletter&utm_medium=email&utm_campaign=spring" \
		"https://mail.example.com/" "Pricing" 1920
	hit "$h" "$UA_FIREFOX_LINUX"  "/pricing?utm_source=google&utm_medium=cpc&utm_campaign=brand" \
		"https://www.google.com/" "Pricing" 1440
	hit "$h" "$UA_EDGE_WIN"       "/docs/getting-started?utm_source=github&utm_medium=social" \
		"https://github.com/" "Getting started" 1366
	hit "$h" "$UA_CHROME_MAC"     "/docs/getting-started" "https://github.com/" "Getting started" 1920
	hit "$h" "$UA_SAFARI_IPHONE"  "/docs/api"           ""                    "API reference" 390
	hit "$h" "$UA_CHROME_ANDROID" "/about"  "https://www.linkedin.com/" "About"   412
	hit "$h" "$UA_FIREFOX_LINUX"  "/about"  ""                          "About"   1440
	hit "$h" "$UA_CHROME_MAC"     "/contact" "https://www.google.com/" "Contact" 1920
	hit "$h" "$UA_CHROME_MAC"     "/blog/kiko-launch" "/blog/htmx-dashboard" "Kiko launch" 1920
	hit "$h" "$UA_CHROME_MAC"     "/pricing"          "/blog/kiko-launch"    "Pricing"     1920
	hit "$h" "$UA_FIREFOX_LINUX"  "/"                 "/blog/self-hosting"   "Home"        1440
}

seed_gghstats() {
	h="$1"
	hit "$h" "$UA_CHROME_MAC"     "/"                    "https://www.google.com/"      "GGH Stats"        1920
	hit "$h" "$UA_FIREFOX_LINUX"  "/"                    "https://discord.com/"         "GGH Stats"        1440
	hit "$h" "$UA_SAFARI_IPHONE"  "/games"               "https://twitter.com/"         "Games"            390
	hit "$h" "$UA_EDGE_WIN"       "/games/elden-ring"    "https://www.reddit.com/"      "Elden Ring stats" 1366
	hit "$h" "$UA_CHROME_ANDROID" "/games/elden-ring"    "https://www.google.com/"      "Elden Ring stats" 412
	hit "$h" "$UA_CHROME_MAC"     "/games/hollow-knight" "https://steamcommunity.com/"  "Hollow Knight"    1920
	hit "$h" "$UA_FIREFOX_LINUX"  "/leaderboard"         ""                             "Leaderboard"      1440
	hit "$h" "$UA_SAFARI_IPHONE"  "/leaderboard"         "https://t.co/stats"           "Leaderboard"      390
	hit "$h" "$UA_EDGE_WIN"       "/players/me"          "https://www.bing.com/"        "My profile"       1366
	hit "$h" "$UA_CHROME_MAC"     "/blog/patch-1-2"      "https://news.ycombinator.com/" "Patch notes"     1920
	hit "$h" "$UA_CHROME_ANDROID" "/about"               "https://www.linkedin.com/"    "About"            412
	hit "$h" "$UA_CHROME_MAC"     "/games/elden-ring"    "/leaderboard"                 "Elden Ring stats" 1920
}

seed_kzero() {
	h="$1"
	hit "$h" "$UA_CHROME_MAC"     "/"              "https://github.com/"             "kzero"          1920
	hit "$h" "$UA_FIREFOX_LINUX"  "/"              "https://dev.to/"                 "kzero"          1440
	hit "$h" "$UA_SAFARI_IPHONE"  "/docs"          ""                                "Documentation"  390
	hit "$h" "$UA_EDGE_WIN"       "/docs/install"  "https://www.google.com/"         "Install guide"  1366
	hit "$h" "$UA_CHROME_ANDROID" "/docs/cli"      "https://www.reddit.com/r/devops" "CLI reference"  412
	hit "$h" "$UA_CHROME_MAC"     "/pricing"       "https://news.ycombinator.com/"   "Pricing"        1920
	hit "$h" "$UA_FIREFOX_LINUX"  "/changelog"     "https://github.com/"             "Changelog"      1440
	hit "$h" "$UA_SAFARI_IPHONE"  "/changelog/v2"  "https://twitter.com/"            "v2 release"     390
	hit "$h" "$UA_EDGE_WIN"       "/blog/launch"   "https://lobste.rs/"              "Launch post"    1366
	hit "$h" "$UA_CHROME_MAC"     "/docs/install"  "/docs/cli"                       "Install guide"  1920
	hit "$h" "$UA_CHROME_ANDROID" "/status"        ""                                "Status"         412
}

seed_generic() {
	h="$1"
	hit "$h" "$UA_CHROME_MAC"     "/"              "https://www.google.com/" "Home"    1920
	hit "$h" "$UA_FIREFOX_LINUX"  "/about"         ""                        "About"   1440
	hit "$h" "$UA_SAFARI_IPHONE"  "/blog"          "https://twitter.com/"    "Blog"    390
	hit "$h" "$UA_EDGE_WIN"       "/contact"       "https://www.bing.com/"   "Contact" 1366
	hit "$h" "$UA_CHROME_ANDROID" "/pricing"       "https://t.co/promo"      "Pricing" 412
	hit "$h" "$UA_CHROME_MAC"     "/docs"          "https://github.com/"     "Docs"    1920
}

seed_host() {
	host="$1"
	case "$host" in
	localhost|127.0.0.1) seed_localhost "$host" ;;
	gghstats.com|www.gghstats.com) seed_gghstats "$host" ;;
	kzero.dev|www.kzero.dev) seed_kzero "$host" ;;
	*) seed_generic "$host" ;;
	esac
	echo "  -> $host"
}

summary_for() {
	host="$1"
	if command -v jq >/dev/null 2>&1; then
		echo "--- $host"
		curl -fsS "$KIKO_URL/api/v1/stats/summary?host=$host" \
			-H "X-API-Key: $KIKO_API_KEY" | jq .
	else
		echo "--- $host"
		curl -fsS "$KIKO_URL/api/v1/stats/summary?host=$host" \
			-H "X-API-Key: $KIKO_API_KEY"
		echo
	fi
}

# Normalize host list: commas/spaces → one host per line
HOSTS=$(printf '%s' "$SEED_HOSTS_RAW" | tr ', ' '\n' | sed '/^$/d')

echo "Seeding kiko at $KIKO_URL..."
while IFS= read -r host; do
	[ -n "$host" ] || continue
	seed_host "$host"
done <<EOF
$HOSTS
EOF

echo "Sent $HITS_SENT hits. Waiting for kiko flush (~12s)..."
sleep 12

while IFS= read -r host; do
	[ -n "$host" ] || continue
	summary_for "$host"
done <<EOF
$HOSTS
EOF

echo "Done. Hosts seeded: $(printf '%s' "$SEED_HOSTS_RAW" | tr '\n' ' ')"
