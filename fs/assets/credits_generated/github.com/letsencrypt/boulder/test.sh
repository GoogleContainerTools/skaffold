#!/usr/bin/env bash

# -e Stops execution in the instance of a command or pipeline error
# -u Treat unset variables as an error and exit immediately
set -eu

if type realpath >/dev/null 2>&1 ; then
  cd "$(realpath -- $(dirname -- "$0"))"
fi

#
# Defaults
#
export RACE="false"
STAGE="starting"
STATUS="FAILURE"
RUN=()
UNIT_PACKAGES=()
FILTER=()

#
# Print Functions
#
function print_outcome() {
  if [ "$STATUS" == SUCCESS ]
  then
    echo -e "\e[32m"$STATUS"\e[0m"
  else
    echo -e "\e[31m"$STATUS"\e[0m while running \e[31m"$STAGE"\e[0m"
  fi
}

function print_list_of_integration_tests() {
  go test -tags integration -list=. ./test/integration/... | grep '^Test'
  exit 0
}

function exit_msg() {
  # complain to STDERR and exit with error
  echo "$*" >&2
  exit 2
}

function check_arg() {
  if [ -z "$OPTARG" ]
  then
    exit_msg "No arg for --$OPT option, use: -h for help">&2
  fi
}

function print_usage_exit() {
  echo "$USAGE"
  exit 0
}

function print_heading {
  echo
  echo -e "\e[34m\e[1m"$1"\e[0m"
}

function run_and_expect_silence() {
  echo "$@"
  result_file=$(mktemp -t bouldertestXXXX)
  "$@" 2>&1 | tee "${result_file}"

  # Fail if result_file is nonempty.
  if [ -s "${result_file}" ]; then
    rm "${result_file}"
    exit 1
  fi
  rm "${result_file}"
}

#
# Testing Helpers
#
function run_unit_tests() {
  if [ "${RACE}" == true ]; then
    # Run the full suite of tests once with the -race flag.
    go test -race "${UNIT_PACKAGES[@]}" "${FILTER[@]}"
  else
    # When running locally, we skip the -race flag for speedier test runs. We
    # also pass -p 1 to require the tests to run serially instead of in
    # parallel. This is because our unittests depend on mutating a database and
    # then cleaning up after themselves. If they run in parallel, they can fail
    # spuriously because one test is modifying a table (especially
    # registrations) while another test is reading it.
    # https://github.com/letsencrypt/boulder/issues/1499
    go test "${UNIT_PACKAGES[@]}" "${FILTER[@]}"
  fi
}

#
# Main CLI Parser
#
USAGE="$(cat -- <<-EOM

Usage:
Boulder test suite CLI, intended to be run inside of a Docker container:

  docker compose run --use-aliases boulder ./$(basename "${0}") [OPTION]...

With no options passed, runs standard battery of tests (lint, unit, and integration)

    -l, --lints                           Adds lint to the list of tests to run
    -u, --unit                            Adds unit to the list of tests to run
    -p <DIR>, --unit-test-package=<DIR>   Run unit tests for specific go package(s)
    -e, --enable-race-detection           Enables -race flag for all unit and integration tests
    -n, --config-next                     Changes BOULDER_CONFIG_DIR from test/config to test/config-next
    -i, --integration                     Adds integration to the list of tests to run
    -s, --start-py                        Adds start to the list of tests to run
    -v, --gomod-vendor                    Adds gomod-vendor to the list of tests to run
    -g, --generate                        Adds generate to the list of tests to run
    -m, --make-artifacts                  Adds make-artifacts to the list of tests to run
    -o, --list-integration-tests          Outputs a list of the available integration tests
    -f <REGEX>, --filter=<REGEX>          Run only those tests matching the regular expression

                                          Note:
                                           This option disables the '"back in time"' integration test setup

                                           For tests, the regular expression is split by unbracketed slash (/)
                                           characters into a sequence of regular expressions

                                          Example:
                                           TestAkamaiPurgerDrainQueueFails/TestWFECORS
    -h, --help                            Shows this help message

EOM
)"

while getopts lueciosvgmnhp:f:-: OPT; do
  if [ "$OPT" = - ]; then     # long option: reformulate OPT and OPTARG
    OPT="${OPTARG%%=*}"       # extract long option name
    OPTARG="${OPTARG#$OPT}"   # extract long option argument (may be empty)
    OPTARG="${OPTARG#=}"      # if long option argument, remove assigning `=`
  fi
  case "$OPT" in
    l | lints )                      RUN+=("lints") ;;
    u | unit )                       RUN+=("unit") ;;
    p | unit-test-package )          check_arg; UNIT_PACKAGES+=("${OPTARG}") ;;
    e | enable-race-detection )      RACE="true" ;;
    i | integration )                RUN+=("integration") ;;
    o | list-integration-tests )     print_list_of_integration_tests ;;
    f | filter )                     check_arg; FILTER+=("${OPTARG}") ;;
    s | start-py )                   RUN+=("start") ;;
    v | gomod-vendor )               RUN+=("gomod-vendor") ;;
    g | generate )                   RUN+=("generate") ;;
    m | make-artifacts )             RUN+=("make-artifacts") ;;
    n | config-next )                BOULDER_CONFIG_DIR="test/config-next" ;;
    h | help )                       print_usage_exit ;;
    ??* )                            exit_msg "Illegal option --$OPT" ;;  # bad long option
    ? )                              exit 2 ;;  # bad short option (error reported via getopts)
  esac
done
shift $((OPTIND-1)) # remove parsed options and args from $@ list

# The list of segments to run. Order doesn't matter. Note: gomod-vendor
# is specifically left out of the defaults, because we don't want to run
# it locally (it could delete local state).
if [ -z "${RUN[@]+x}" ]
then
  RUN+=("lints" "unit" "integration")
fi

# Filter is used by unit and integration but should not be used for both at the same time
if [[ "${RUN[@]}" =~ unit ]] && [[ "${RUN[@]}" =~ integration ]] && [[ -n "${FILTER[@]+x}" ]]
then
  exit_msg "Illegal option: (-f, --filter) when specifying both (-u, --unit) and (-i, --integration)"
fi

# If unit + filter: set correct flags for go test
if [[ "${RUN[@]}" =~ unit ]] && [[ -n "${FILTER[@]+x}" ]]
then
  FILTER=(--test.run "${FILTER[@]}")
fi

# If integration + filter: set correct flags for test/integration-test.py
if [[ "${RUN[@]}" =~ integration ]] && [[ -n "${FILTER[@]+x}" ]]
then
  FILTER=(--filter "${FILTER[@]}")
fi

# If unit test packages are not specified: set flags to run unit tests
# for all boulder packages
if [ -z "${UNIT_PACKAGES[@]+x}" ]
then
  UNIT_PACKAGES+=("-p" "1" "./...")
fi

print_heading "Boulder Test Suite CLI"
print_heading "Settings:"

# On EXIT, trap and print outcome
trap "print_outcome" EXIT

settings="$(cat -- <<-EOM
    RUN:                ${RUN[@]}
    BOULDER_CONFIG_DIR: $BOULDER_CONFIG_DIR
    UNIT_PACKAGES:      ${UNIT_PACKAGES[@]}
    RACE:               $RACE
    FILTER:             ${FILTER[@]}

EOM
)"

echo "$settings"
print_heading "Starting..."

#
# Run various linters.
#
STAGE="lints"
if [[ "${RUN[@]}" =~ "$STAGE" ]] ; then
  print_heading "Running Lints"
  golangci-lint run --timeout 9m ./...
  python3 test/grafana/lint.py
  # Check for common spelling errors using codespell.
  # Update .codespell.ignore.txt if you find false positives (NOTE: ignored
  # words should be all lowercase).
  run_and_expect_silence codespell \
    --ignore-words=.codespell.ignore.txt \
    --skip=.git,.gocache,go.sum,go.mod,vendor,bin,*.pyc,*.pem,*.der,*.resp,*.req,*.csr,.codespell.ignore.txt,.*.swp
fi

#
# Unit Tests.
#
STAGE="unit"
if [[ "${RUN[@]}" =~ "$STAGE" ]] ; then
  print_heading "Running Unit Tests"
  run_unit_tests
fi

#
# Integration tests
#
STAGE="integration"
if [[ "${RUN[@]}" =~ "$STAGE" ]] ; then
  print_heading "Running Integration Tests"
  python3 test/integration-test.py --chisel --gotest "${FILTER[@]}"
fi

# Test that just ./start.py works, which is a proxy for testing that
# `docker compose up` works, since that just runs start.py (via entrypoint.sh).
STAGE="start"
if [[ "${RUN[@]}" =~ "$STAGE" ]] ; then
  print_heading "Running Start Test"
  python3 start.py &
  for I in $(seq 1 100); do
    sleep 1
    curl -s http://localhost:4001/directory && break
  done
  if [[ "$I" = 100 ]]; then
    echo "Boulder did not come up after ./start.py."
    exit 1
  fi
fi

# Run go mod vendor (happens only in CI) to check that the versions in
# vendor/ really exist in the remote repo and match what we have.
STAGE="gomod-vendor"
if [[ "${RUN[@]}" =~ "$STAGE" ]] ; then
  print_heading "Running Go Mod Vendor"
  go mod vendor
  run_and_expect_silence git diff --exit-code .
fi

# Run generate to make sure all our generated code can be re-generated with
# current tools.
# Note: Some of the tools we use seemingly don't understand ./vendor yet, and
# so will fail if imports are not available in $GOPATH.
STAGE="generate"
if [[ "${RUN[@]}" =~ "$STAGE" ]] ; then
  print_heading "Running Generate"
  # Additionally, we need to run go install before go generate because the stringer command
  # (using in ./grpc/) checks imports, and depends on the presence of a built .a
  # file to determine an import really exists. See
  # https://golang.org/src/go/internal/gcimporter/gcimporter.go#L30
  # Without this, we get error messages like:
  #   stringer: checking package: grpc/bcodes.go:6:2: could not import
  #     github.com/letsencrypt/boulder/probs (can't find import:
  #     github.com/letsencrypt/boulder/probs)
  go install ./probs
  go install ./vendor/google.golang.org/grpc/codes
  run_and_expect_silence go generate ./...
  run_and_expect_silence git diff --exit-code .
fi

STAGE="make-artifacts"
if [[ "${RUN[@]}" =~ "$STAGE" ]]; then
  print_heading "Running Make Artifacts"
  make deb rpm tar
fi

# Because set -e stops execution in the instance of a command or pipeline
# error; if we got here we assume success
STATUS="SUCCESS"
