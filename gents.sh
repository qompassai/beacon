#!/bin/bash
set -euo pipefail

go run vendor/github.com/qompassai/sherpats/cmd/sherpats/main.go -bytes-to-string -slices-nullable -maps-nullable -nullable-optional -namespace api api <$1 >$2.tmp
if cmp -s $2 $2.tmp; then
    rm $2.tmp
else
    mv $2.tmp $2
fi
