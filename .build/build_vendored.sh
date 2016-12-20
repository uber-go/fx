#!/bin/bash

# build_vendored.sh builds executables from vendored code inside a repository.
# For example,
#
# 	build_vendored.sh .bin go.uber.org/thriftrw
#
# Will build the version of thriftrw at vendor/go.uber.org/thriftrw and place
# the executable at .bin/thriftrw.
#
# This is better than doing `go install ./vendor/go.uber.org/thriftrw` because
# it takes care of fetching dependencies of the installed package (if it uses
# Glide) and does not change the globally installed version.

if [[ "$VERBOSE" == "1" ]]; then
	set -x
fi

if [[ -z "$2" ]]; then
	echo "USAGE: $0 DIR IMPORTPATH"
	echo ""
	echo "The binary at IMPORTPATH will be built and saved to DIR."
	exit 1
fi

if [[ ! -d vendor ]]; then
	echo "Must be run from a directory containing vendored code."
	exit 1
fi

function die() {
	echo "$1" >&2
	exit 1
}

function abspath() {
	(cd "$1" || die "Directory $1 does not exist"; pwd)
}

# findGlideLock dir looks for glide.lock in dir or any of its parent
# directories.
#
# Returns the full path to glide.lock or an empty string.
function findGlideLock() {
	if [[ -e "$1/glide.lock" ]]; then
		echo "$1/glide.lock"
		return
	fi

	if [[ "$GOPATH/src" == "$1" ]]; then
		return
	fi

	findGlideLock "$(abspath "$1/..")"
}

outputDir="$1"
importPath="$2"

# not an absolute path
if [[ "${outputDir#/}" == "$outputDir" ]]; then
	outputDir="$(pwd)/$outputDir"
fi

GOPATH=$(mktemp -d)
export GOPATH

ln -s "$PWD/vendor" "$GOPATH/src" || die "Failed to symlink vendor"
cd "$GOPATH/src/$importPath" || die "Cannot find $importPath"

# We have dependencies
glideLock=$(findGlideLock "$GOPATH/src/$importPath")
if [[ -n "$glideLock" ]]; then
	(cd "$(dirname "$glideLock")" && glide install) || die "Could not install dependencies"
	trap 'rm -rf $(dirname $glideLock)/vendor' EXIT
fi

go build -o "$outputDir/$(basename "$importPath")" . || dir "Failed to build"
