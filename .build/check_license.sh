#!/bin/bash
set -eu
set -o pipefail

# Try to use npm/uber-licence if node is available
LICENCE_BIN="uber-licence"
run_uber_licence() {
  local bin="$LICENCE_BIN"
  # Ok, somebody hasn't run `npm install -g uber-licence`, that's cool
  if ! which "$bin" >/dev/null ; then
    npm install "$bin"
    bin="./node_modules/uber-licence/bin/licence"
  fi

  readonly local output=$("$bin" --file "*.go" | sed "s/^fix //")
  if [ -z "$output" ]; then
    exit 0
  fi

  echo "The following files were missing licence headers."
  echo "Please amend your commit."
  echo
  echo "$output"
  exit 1
}

set +u
# Don't even try in CI, node is flaky
if [ -z "$CI" ] ; then
  if which uber-licence >/dev/null || which npm >/dev/null; then
    run_uber_licence
  fi
fi
set -u

text=$(head -1 LICENSE.txt)

ERROR_COUNT=0
while read -r file
do
    head -1 "${file}" | grep -q "${text}"
    if [ $? -ne 0 ]; then
        echo "$file is missing license header."
        (( ERROR_COUNT++ ))
    fi
done < <(git ls-files "*\.go")

exit $ERROR_COUNT
