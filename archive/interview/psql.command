#!/bin/sh
set -eu

profile=catalog
case "${1:-}" in
    catalog|orders) profile="$1"; shift ;;
    ""|-*) ;;
    *) printf 'Usage: %s [catalog|orders] [psql options]\n' "$0" >&2; exit 2 ;;
esac

cd -- "$(dirname -- "$0")"
exec psql "postgres://student@127.0.0.1:15432/go_interview_${profile}?sslmode=disable" "$@"
