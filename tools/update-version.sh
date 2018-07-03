#!/bin/bash

set -e -u

basedir="$(dirname "$(dirname "$0")")"
verfile="$basedir/cmd/goen/version.go"

version="$(git describe --tags --always --dirty )"

sed -i -e 's/version = "[^"]*"/version = "'$version'"/' "$verfile"

git commit -i "$verfile" -m 'Update version'
