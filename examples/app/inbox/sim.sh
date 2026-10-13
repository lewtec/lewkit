#!/usr/bin/env bash
# Build Inbox for the iOS Simulator, boot a phone, install, then ping
# the custom scheme and drop a markdown file into Cache/open.jsonl.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

BUNDLE="br.tec.lew.eletrocromo.inbox"
APP="$ROOT/dist/Inbox.app"
SCHEME="eletrocromo-inbox"

boot_sim() {
	if xcrun simctl list devices booted | grep -q iPhone; then
		return
	fi
	local udid
	udid="$(xcrun simctl list devices available | awk -F '[()]' '/iPhone/{print $2; exit}')"
	if [ -z "$udid" ]; then
		echo "no available iPhone simulator" >&2
		exit 1
	fi
	xcrun simctl boot "$udid"
	open -a Simulator
	xcrun simctl bootstatus "$udid" -b
}

cmd="${1:-sim}"
case "$cmd" in
build)
	go run ./cmd/lewkit release build \
		--goos ios \
		--goarch arm64 \
		--app \
		--sdk iphonesimulator \
		--config examples/inbox/eletrocromo.json \
		--out "$APP" \
		--workdir dist/ios-inbox
	;;
ping)
	xcrun simctl openurl booted "${SCHEME}://from-simctl"
	data="$(xcrun simctl get_app_container booted "$BUNDLE" data)"
	cache="$data/Library/Caches"
	mkdir -p "$cache/inbox"
	note="$cache/inbox/from-sim.md"
	printf '# from simctl\nhello inbox\n' >"$note"
	printf '%s\n' "{\"kind\":\"files\",\"paths\":[\"$note\"]}" >>"$cache/open.jsonl"
	echo "opened ${SCHEME}://from-simctl and dropped $note"
	;;
sim)
	"$0" build
	boot_sim
	xcrun simctl install booted "$APP"
	xcrun simctl launch booted "$BUNDLE" || true
	sleep 4
	"$0" ping
	echo "Inbox is on the simulator."
	;;
*)
	echo "usage: $0 [sim|build|ping]" >&2
	exit 2
	;;
esac
