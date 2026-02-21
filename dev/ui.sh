#!/usr/bin/env bash
set -euo pipefail

SESSION="podji"
DELAY="${PODJI_DELAY:-0.3}"
DIR="$(cd "$(dirname "$0")/.." && pwd)"

die() { echo "ERROR: $*" >&2; exit 1; }

has_session() { tmux has-session -t "$SESSION" 2>/dev/null; }

capture() {
    local flag="${1:-}"
    if [[ "$flag" == "-e" ]]; then
        tmux capture-pane -t "$SESSION" -p -e
    else
        tmux capture-pane -t "$SESSION" -p
    fi
}

auto_capture() {
    sleep "$DELAY"
    capture
}

cmd_start() {
    cd "$DIR"
    go build ./cmd/podji || die "build failed"
    has_session && tmux kill-session -t "$SESSION" 2>/dev/null
    tmux new-session -d -s "$SESSION" -x 120 -y 40 "$DIR/podji"
    sleep 1
    capture
}

cmd_key() {
    [[ $# -ge 1 ]] || die "usage: $0 key <k> [k2 ...]"
    has_session || die "no session running"
    for k in "$@"; do
        tmux send-keys -t "$SESSION" "$k"
    done
    auto_capture
}

cmd_type() {
    [[ $# -ge 1 ]] || die "usage: $0 type <text>"
    has_session || die "no session running"
    tmux send-keys -t "$SESSION" -l "$*"
    auto_capture
}

cmd_cap() {
    has_session || die "no session running"
    capture "$@"
}

cmd_vhs() {
    local tape="${1:-}"
    if [[ -z "$tape" ]]; then
        tape="$(mktemp /tmp/podji_vhs_XXXXXX.tape)"
        cat > "$tape" << 'TAPE'
Set Width 1200
Set Height 600
Set FontSize 14
Hide
Type "./podji"
Enter
Sleep 2s
Show
Sleep 1s
TAPE
    fi
    local gif="/tmp/podji_vhs_$(date +%s).gif"
    (cd "$DIR" && vhs -o "$gif" "$tape")
    echo "$gif"
}

cmd_quit() {
    if has_session; then
        tmux send-keys -t "$SESSION" q
        sleep 0.5
        tmux kill-session -t "$SESSION" 2>/dev/null || true
    fi
    echo "session ended"
}

case "${1:-}" in
    start) shift; cmd_start "$@" ;;
    key)   shift; cmd_key "$@" ;;
    type)  shift; cmd_type "$@" ;;
    cap)   shift; cmd_cap "$@" ;;
    vhs)   shift; cmd_vhs "$@" ;;
    quit)  shift; cmd_quit "$@" ;;
    *)     die "usage: $0 {start|key|type|cap|vhs|quit}" ;;
esac
