#!/bin/bash

set -e -u

function usage_exit()
{
    local code="$1"
    local prog="$(basename "$0")"
    echo "Usage: ${prog} {version}" 1>&2
    exit "$code"
}

if [[ -z "${1+xxx}" ]]; then
    usage_exit 128
fi

version="$1"

case "${version}" in
    v*.*.*)
        # ok
        ;;
    *)
        usage_exit 128
        ;;
esac

basedir="$(dirname "$(dirname "$0")")"
verfile="$basedir/cmd/goen/version.go"

sed -i -e 's/version = "[^"]*"/version = "'$version'"/' "$verfile"

echo "commit $verfile" 1>&2
git commit -i "$verfile" -m 'update version'

echo "tag $version" 1>&2
git tag "$version"
