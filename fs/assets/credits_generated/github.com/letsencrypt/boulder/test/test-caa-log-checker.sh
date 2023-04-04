#!/usr/bin/env bash
#
# Run the CAA log checker over logs from an integration tests run.
#
# We verify two things:
#  - It should succeed when given the full output as both RA logs and VA logs.
#  - It should fail when given RA logs (containing issuances) but empty VA logs
#    (containing CAA checks).
#

set -x

LOGFILE=/tmp/boulder.log

# We rely on the integration tests previously having been run with a container
# name of "boulder_tests". See ../t.sh.
docker logs boulder_tests > ${LOGFILE}

# Expect success
./bin/boulder caa-log-checker -ra-log ${LOGFILE} -va-logs ${LOGFILE}

# Expect error
./bin/boulder caa-log-checker -ra-log ${LOGFILE} -va-logs /dev/null >/tmp/output 2>&1 &&
  (echo "caa-log-checker succeeded when it should have failed. Output:";
   cat /tmp/output;
   exit 9)

# Explicitly exit zero so the status code from the intentionally-erroring last
# command doesn't wind up as the overall status code for this script.
exit 0
