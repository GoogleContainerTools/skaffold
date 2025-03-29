#!/usr/bin/env bash
set -e

: ${GITHUB_TOKEN?"Need to set GITHUB_TOKEN env var."}

usage() {
  echo "Usage: "
  echo "  $0 <workflow> <job> [event]"
  echo "    <workflow>  the workflow file to use"
  echo "    <job>  job name to execute"
  echo "    [event]  event file"
  exit 1; 
}

WORKFLOW_FILE="${1}"
if [[ -z "${WORKFLOW_FILE}" ]]; then
  echo "Must specify a workflow file"
  echo
  usage
  exit 1
fi

JOB_NAME="${2}"
if [[ -z "${JOB_NAME}" ]]; then
  echo "Must specify a job"
  echo
  usage
  exit 1
fi

EVENT_FILE="${3}"

ACT_EXEC=$(command -v act)
if [[ -z "${ACT_EXEC}" ]]; then
  echo "Need act to be available: https://github.com/nektos/act"
  exit 1
fi

if [[ -n "${EVENT_FILE}" ]]; then
  ${ACT_EXEC} \
    -v \
    -e "${EVENT_FILE}" \
    -P ubuntu-latest=nektos/act-environments-ubuntu:18.04 \
    -s GITHUB_TOKEN \
    -W "${WORKFLOW_FILE}" \
    -j "${JOB_NAME}"
else
  ${ACT_EXEC} \
    -v \
    -P ubuntu-latest=nektos/act-environments-ubuntu:18.04 \
    -s GITHUB_TOKEN \
    -W "${WORKFLOW_FILE}" \
    -j "${JOB_NAME}"
fi