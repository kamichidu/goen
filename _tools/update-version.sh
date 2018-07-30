#!/bin/bash

set -e -u

version="$1"

basedir="$(dirname "$(dirname "$0")")"
verfile="$basedir/cmd/goen/version.go"

sed -i -e 's/version = "[^"]*"/version = "'$version'"/' "$verfile"

echo "commit $verfile" 1>&2
git commit -i "$verfile" -m 'update version'

echo "tag $version" 1>&2
git tag "$version"
