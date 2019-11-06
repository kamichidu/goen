#!/bin/bash

set -e -u

golangci_lint_version=v1.21.0
gobin="$(go env GOBIN)"
if [[ -z "${gobin}" ]]; then
    gobin="$(go env GOPATH)/bin"
fi

if [[ -z "$(which golangci-lint 2>/dev/null)" ]]; then
    wget -O - -q https://install.goreleaser.com/github.com/golangci/golangci-lint.sh \
        | sh -s -- -b "${gobin}" "${golangci_lint_version}"
fi

exec "golangci-lint" "run" "$@"
