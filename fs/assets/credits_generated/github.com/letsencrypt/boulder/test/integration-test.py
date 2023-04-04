#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
This file contains basic infrastructure for running the integration test cases.
Most test cases are in v2_integration.py. There are a few exceptions: Test cases
that don't test either the v1 or v2 API are in this file, and test cases that
have to run at a specific point in the cycle (e.g. after all other test cases)
are also in this file.
"""
import argparse
import datetime
import inspect
import json
import os
import random
import re
import requests
import subprocess
import shlex
import signal
import time

import startservers

import v2_integration
from helpers import *

from acme import challenges

# Set the environment variable RACE to anything other than 'true' to disable
# race detection. This significantly speeds up integration testing cycles
# locally.
race_detection = True
if os.environ.get('RACE', 'true') != 'true':
    race_detection = False

def run_go_tests(filterPattern=None):
    """
    run_go_tests launches the Go integration tests. The go test command must
    return zero or an exception will be raised. If the filterPattern is provided
    it is used as the value of the `--test.run` argument to the go test command.
    """
    cmdLine = ["go", "test"]
    if filterPattern is not None and filterPattern != "":
        cmdLine = cmdLine + ["--test.run", filterPattern]
    cmdLine = cmdLine + ["-tags", "integration", "-count=1", "-race", "./test/integration"]
    subprocess.check_call(cmdLine, stderr=subprocess.STDOUT)

def test_single_ocsp():
    """Run ocsp-responder with the single OCSP response generated for the intermediate
       certificate using the ceremony tool during setup and check that it successfully
       answers OCSP requests, and shut the responder back down.

       This is a non-API test.
    """
    p = subprocess.Popen(
        ["./bin/boulder", "ocsp-responder", "--config", "test/issuer-ocsp-responder.json"])
    waitport(4003, ' '.join(p.args))

    # Verify that the static OCSP responder, which answers with a
    # pre-signed, long-lived response for the CA cert, works.
    verify_ocsp("/hierarchy/intermediate-cert-rsa-a.pem", "/hierarchy/root-cert-rsa.pem", "http://localhost:4003", "good")

    p.send_signal(signal.SIGTERM)
    p.wait()

exit_status = 1

def main():
    parser = argparse.ArgumentParser(description='Run integration tests')
    parser.add_argument('--chisel', dest="run_chisel", action="store_true",
                        help="run integration tests using chisel")
    parser.add_argument('--gotest', dest="run_go", action="store_true",
                        help="run Go integration tests")
    parser.add_argument('--filter', dest="test_case_filter", action="store",
                        help="Regex filter for test cases")
    # allow any ACME client to run custom command for integration
    # testing (without having to implement its own busy-wait loop)
    parser.add_argument('--custom', metavar="CMD", help="run custom command")
    parser.set_defaults(run_chisel=False, test_case_filter="", skip_setup=False)
    args = parser.parse_args()

    if not (args.run_chisel or args.custom  or args.run_go is not None):
        raise(Exception("must run at least one of the letsencrypt or chisel tests with --chisel, --gotest, or --custom"))

    if not startservers.install(race_detection=race_detection):
        raise(Exception("failed to build"))

    # Setup issuance hierarchy
    startservers.setupHierarchy()

    if not args.test_case_filter:
        now = datetime.datetime.utcnow()

        six_months_ago = now+datetime.timedelta(days=-30*6)
        if not startservers.start(fakeclock=fakeclock(six_months_ago)):
            raise(Exception("startservers failed (mocking six months ago)"))
        setup_six_months_ago()
        startservers.stop()

        twenty_days_ago = now+datetime.timedelta(days=-20)
        if not startservers.start(fakeclock=fakeclock(twenty_days_ago)):
            raise(Exception("startservers failed (mocking twenty days ago)"))
        setup_twenty_days_ago()
        startservers.stop()

    if not startservers.start(fakeclock=None):
        raise(Exception("startservers failed"))

    if args.run_chisel:
        run_chisel(args.test_case_filter)

    if args.run_go:
        run_go_tests(args.test_case_filter)

    if args.custom:
        run(args.custom.split())

    # Skip the last-phase checks when the test case filter is one, because that
    # means we want to quickly iterate on a single test case.
    if not args.test_case_filter:
        run_cert_checker()
        check_balance()

        # Run the load-generator last. run_loadtest will stop the
        # pebble-challtestsrv before running the load-generator and will not restart
        # it.
        run_loadtest()

    if not startservers.check():
        raise(Exception("startservers.check failed"))

    # This test is flaky, so it's temporarily disabled.
    # TODO(#4583): Re-enable this test.
    #check_slow_queries()

    global exit_status
    exit_status = 0

def check_slow_queries():
    """Checks that we haven't run any slow queries during the integration test.

    This depends on flags set on mysqld in docker-compose.yml.

    We skip the boulder_sa_test database because we manually run a bunch of
    non-indexed queries in unittests. We skip actions by the setup and root
    users because they're known to be non-indexed. Similarly we skip the
    cert_checker, mailer, and janitor's work because they are known to be
    slow (though we should eventually improve these).
    The SELECT ... IN () on the authz2 table shows up in the slow query log
    a lot. Presumably when there are a lot of entries in the IN() argument
    and the table is small, it's not efficient to use the index. But we
    should dig into this more.
    """
    query = """
        SELECT * FROM mysql.slow_log
            WHERE db != 'boulder_sa_test'
            AND user_host NOT LIKE "test_setup%"
            AND user_host NOT LIKE "root%"
            AND user_host NOT LIKE "cert_checker%"
            AND user_host NOT LIKE "mailer%"
            AND user_host NOT LIKE "janitor%"
            AND sql_text NOT LIKE 'SELECT status, expires FROM authz2 WHERE id IN %'
            AND sql_text NOT LIKE '%LEFT JOIN orderToAuthz2 %'
        \G
    """
    output = subprocess.check_output(
      ["mysql", "-h", "boulder-mysql", "-e", query],
      stderr=subprocess.STDOUT).decode()
    if len(output) > 0:
        print(output)
        raise Exception("Found slow queries in the slow query log")

def run_chisel(test_case_filter):
    for key, value in inspect.getmembers(v2_integration):
      if callable(value) and key.startswith('test_') and re.search(test_case_filter, key):
        value()
    for key, value in globals().items():
      if callable(value) and key.startswith('test_') and re.search(test_case_filter, key):
        value()

def run_loadtest():
    """Run the ACME v2 load generator."""
    latency_data_file = "%s/integration-test-latency.json" % tempdir

    # Stop the global pebble-challtestsrv - it will conflict with the
    # load-generator's internal challtestsrv. We don't restart it because
    # run_loadtest() is called last and there are no remaining tests to run that
    # might benefit from the pebble-challtestsrv being restarted.
    startservers.stopChallSrv()

    run(["./bin/load-generator",
        "-config", "test/load-generator/config/integration-test-config.json",
        "-results", latency_data_file])

def check_balance():
    """Verify that gRPC load balancing across backends is working correctly.

    Fetch metrics from each backend and ensure the grpc_server_handled_total
    metric is present, which means that backend handled at least one request.
    """
    addresses = [
        "sa1.service.consul:8003",
        "sa2.service.consul:8103",
        "publisher1.service.consul:8009",
        "publisher2.service.consul:8109",
        "va1.service.consul:8004",
        "va2.service.consul:8104",
        "ca1.service.consul:8001",
        "ca2.service.consul:8104",
        "ra1.service.consul:8002",
        "ra2.service.consul:8102",
    ]
    for address in addresses:
        metrics = requests.get("http://%s/metrics" % address)
        if not "grpc_server_handled_total" in metrics.text:
            raise(Exception("no gRPC traffic processed by %s; load balancing problem?")
                % address)

def run_cert_checker():
    run(["./bin/boulder", "cert-checker", "-config", "%s/cert-checker.json" % config_dir])

if __name__ == "__main__":
    main()
