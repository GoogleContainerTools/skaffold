import atexit
import collections
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import threading
import time

from helpers import waithealth, waitport, config_dir, CONFIG_NEXT

Service = collections.namedtuple('Service', ('name', 'debug_port', 'grpc_addr', 'cmd', 'deps'))

SERVICES = (
    Service('boulder-remoteva-a',
        8011, 'rva1.service.consul:9097',
        ('./bin/boulder', 'boulder-remoteva', '--config', os.path.join(config_dir, 'va-remote-a.json')),
        None),
    Service('boulder-remoteva-b',
        8012, 'rva1.service.consul:9098',
        ('./bin/boulder', 'boulder-remoteva', '--config', os.path.join(config_dir, 'va-remote-b.json')),
        None),
    Service('boulder-sa-1',
        8003, 'sa1.service.consul:9095',
        ('./bin/boulder', 'boulder-sa', '--config', os.path.join(config_dir, 'sa.json'), '--addr', 'sa1.service.consul:9095', '--debug-addr', ':8003'),
        None),
    Service('boulder-sa-2',
        8103, 'sa2.service.consul:9095',
        ('./bin/boulder', 'boulder-sa', '--config', os.path.join(config_dir, 'sa.json'), '--addr', 'sa2.service.consul:9095', '--debug-addr', ':8103'),
        None),
    Service('ct-test-srv',
        4500, None,
        ('./bin/ct-test-srv', '--config', 'test/ct-test-srv/ct-test-srv.json'), None),
    Service('boulder-publisher-1',
        8009, 'publisher1.service.consul:9091',
        ('./bin/boulder', 'boulder-publisher', '--config', os.path.join(config_dir, 'publisher.json'), '--addr', 'publisher1.service.consul:9091', '--debug-addr', ':8009'),
        None),
    Service('boulder-publisher-2',
        8109, 'publisher2.service.consul:9091',
        ('./bin/boulder', 'boulder-publisher', '--config', os.path.join(config_dir, 'publisher.json'), '--addr', 'publisher2.service.consul:9091', '--debug-addr', ':8109'),
        None),
    Service('mail-test-srv',
        9380, None,
        ('./bin/mail-test-srv', '--closeFirst', '5', '--cert', 'test/mail-test-srv/localhost/cert.pem', '--key', 'test/mail-test-srv/localhost/key.pem'),
        None),
    Service('ocsp-responder',
        8005, None,
        ('./bin/boulder', 'ocsp-responder', '--config', os.path.join(config_dir, 'ocsp-responder.json')),
        ('boulder-ra-1', 'boulder-ra-2')),
    Service('boulder-va-1',
        8004, 'va1.service.consul:9092',
        ('./bin/boulder', 'boulder-va', '--config', os.path.join(config_dir, 'va.json'), '--addr', 'va1.service.consul:9092', '--debug-addr', ':8004'),
        ('boulder-remoteva-a', 'boulder-remoteva-b')),
    Service('boulder-va-2',
        8104, 'va2.service.consul:9092',
        ('./bin/boulder', 'boulder-va', '--config', os.path.join(config_dir, 'va.json'), '--addr', 'va2.service.consul:9092', '--debug-addr', ':8104'),
        ('boulder-remoteva-a', 'boulder-remoteva-b')),
    Service('boulder-ca-a',
        8001, 'ca1.service.consul:9093',
        ('./bin/boulder', 'boulder-ca', '--config', os.path.join(config_dir, 'ca-a.json'), '--ca-addr', 'ca1.service.consul:9093', '--ocsp-addr', 'ca1.service.consul:9096', '--crl-addr', 'ca1.service.consul:9106', '--debug-addr', ':8001'),
        ('boulder-sa-1', 'boulder-sa-2')),
    Service('boulder-ca-b',
        8101, 'ca2.service.consul:9093',
        ('./bin/boulder', 'boulder-ca', '--config', os.path.join(config_dir, 'ca-b.json'), '--ca-addr', 'ca2.service.consul:9093', '--ocsp-addr', 'ca2.service.consul:9096', '--crl-addr', 'ca2.service.consul:9106', '--debug-addr', ':8101'),
        ('boulder-sa-1', 'boulder-sa-2')),
    Service('akamai-test-srv',
        6789, None,
        ('./bin/akamai-test-srv', '--listen', 'localhost:6789', '--secret', 'its-a-secret'),
        None),
    Service('akamai-purger',
        9666, None,
        ('./bin/boulder', 'akamai-purger', '--config', os.path.join(config_dir, 'akamai-purger.json')),
        ('akamai-test-srv',)),
    Service('s3-test-srv',
        7890, None,
        ('./bin/s3-test-srv', '--listen', 'localhost:7890'),
        None),
    Service('crl-storer',
        9667, None,
        ('./bin/boulder', 'crl-storer', '--config', os.path.join(config_dir, 'crl-storer.json')),
        ('s3-test-srv',)),
    Service('ocsp-updater',
        8006, None,
        ('./bin/boulder', 'ocsp-updater', '--config', os.path.join(config_dir, 'ocsp-updater.json')),
        ('boulder-ca-a', 'boulder-ca-b')),
    Service('crl-updater',
        8021, None,
        ('./bin/boulder', 'crl-updater', '--config', os.path.join(config_dir, 'crl-updater.json')),
        ('boulder-ca-a', 'boulder-ca-b', 'boulder-sa-1', 'boulder-sa-2', 'crl-storer')),
    Service('boulder-ra-1',
        8002, 'ra1.service.consul:9094',
        ('./bin/boulder', 'boulder-ra', '--config', os.path.join(config_dir, 'ra.json'), '--addr', 'ra1.service.consul:9094', '--debug-addr', ':8002'),
        ('boulder-sa-1', 'boulder-sa-2', 'boulder-ca-a', 'boulder-ca-b', 'boulder-va-1', 'boulder-va-2', 'akamai-purger', 'boulder-publisher-1', 'boulder-publisher-2')),
    Service('boulder-ra-2',
        8102, 'ra2.service.consul:9094',
        ('./bin/boulder', 'boulder-ra', '--config', os.path.join(config_dir, 'ra.json'), '--addr', 'ra2.service.consul:9094', '--debug-addr', ':8102'),
        ('boulder-sa-1', 'boulder-sa-2', 'boulder-ca-a', 'boulder-ca-b', 'boulder-va-1', 'boulder-va-2', 'akamai-purger', 'boulder-publisher-1', 'boulder-publisher-2')),
    Service('bad-key-revoker',
        8020, None,
        ('./bin/boulder', 'bad-key-revoker', '--config', os.path.join(config_dir, 'bad-key-revoker.json')),
        ('boulder-ra-1', 'boulder-ra-2', 'mail-test-srv')),
    Service('nonce-service-taro',
        8111, 'nonce1.service.consul:9101',
        ('./bin/boulder', 'nonce-service', '--config', os.path.join(config_dir, 'nonce-a.json'), '--addr', '10.77.77.77:9101', '--debug-addr', ':8111',),
        None),
    Service('nonce-service-zinc',
        8112, 'nonce2.service.consul:9101',
        ('./bin/boulder', 'nonce-service', '--config', os.path.join(config_dir, 'nonce-b.json'), '--addr', '10.88.88.88:9101', '--debug-addr', ':8112',),
        None),
    Service('boulder-wfe2',
        4001, None,
        ('./bin/boulder', 'boulder-wfe2', '--config', os.path.join(config_dir, 'wfe2.json')),
        ('boulder-ra-1', 'boulder-ra-2', 'boulder-sa-1', 'boulder-sa-2', 'nonce-service-taro', 'nonce-service-zinc')),
    Service('log-validator',
        8016, None,
        ('./bin/boulder', 'log-validator', '--config', os.path.join(config_dir, 'log-validator.json')),
        None),
)

def _service_toposort(services):
    """Yields Service objects in topologically sorted order.

    No service will be yielded until every service listed in its deps value
    has been yielded.
    """
    ready = set([s for s in services if not s.deps])
    blocked = set(services) - ready
    done = set()
    while ready:
        service = ready.pop()
        yield service
        done.add(service.name)
        new = set([s for s in blocked if all([d in done for d in s.deps])])
        ready |= new
        blocked -= new
    if blocked:
        print("WARNING: services with unsatisfied dependencies:")
        for s in blocked:
            print(s.name, ":", s.deps)
        raise(Exception("Unable to satisfy service dependencies"))

processes = []

# NOTE(@cpu): We manage the challSrvProcess separately from the other global
# processes because we want integration tests to be able to stop/start it (e.g.
# to run the load-generator).
challSrvProcess = None

def setupHierarchy():
    """Set up the issuance hierarchy. Must have called install() before this."""
    e = os.environ.copy()
    e.setdefault("GOBIN", "%s/bin" % os.getcwd())
    try:
        subprocess.check_output(["go", "run", "test/cert-ceremonies/generate.go"], env=e)
    except subprocess.CalledProcessError as e:
        print(e.output)
        raise


def install(race_detection):
    # Pass empty BUILD_TIME and BUILD_ID flags to avoid constantly invalidating the
    # build cache with new BUILD_TIMEs, or invalidating it on merges with a new
    # BUILD_ID.
    go_build_flags='-tags "integration"'
    if race_detection:
        go_build_flags += ' -race'

    return subprocess.call(["/usr/bin/make", "GO_BUILD_FLAGS=%s" % go_build_flags]) == 0

def run(cmd, fakeclock):
    e = os.environ.copy()
    e.setdefault("GORACE", "halt_on_error=1")
    if fakeclock:
        e.setdefault("FAKECLOCK", fakeclock)
    p = subprocess.Popen(cmd, env=e)
    p.cmd = cmd
    return p

def start(fakeclock):
    """Return True if everything builds and starts.

    Give up and return False if anything fails to build, or dies at
    startup. Anything that did start before this point can be cleaned
    up explicitly by calling stop(), or automatically atexit.
    """
    signal.signal(signal.SIGTERM, lambda _, __: stop())
    signal.signal(signal.SIGINT, lambda _, __: stop())

    # Start the pebble-challtestsrv first so it can be used to resolve DNS for
    # gRPC.
    startChallSrv()

    # Processes are in order of dependency: Each process should be started
    # before any services that intend to send it RPCs. On shutdown they will be
    # killed in reverse order.
    for service in _service_toposort(SERVICES):
        print("Starting service", service.name)
        try:
            global processes
            p = run(service.cmd, fakeclock)
            processes.append(p)
            if service.grpc_addr is not None:
                waithealth(' '.join(p.args), service.grpc_addr)
            else:
                if not waitport(service.debug_port, ' '.join(p.args), perTickCheck=check):
                    return False
        except Exception as e:
            print("Error starting service %s: %s" % (service.name, e))
            return False

    print("All servers running. Hit ^C to kill.")
    return True

def check():
    """Return true if all started processes are still alive.

    Log about anything that died. The pebble-challtestsrv is not considered when
    checking processes.
    """
    global processes
    busted = []
    stillok = []
    for p in processes:
        if p.poll() is None:
            stillok.append(p)
        else:
            busted.append(p)
    if busted:
        print("\n\nThese processes exited early (check above for their output):")
        for p in busted:
            print("\t'%s' with pid %d exited %d" % (p.cmd, p.pid, p.returncode))
    processes = stillok
    return not busted

def startChallSrv():
    """
    Start the pebble-challtestsrv and wait for it to become available. See also
    stopChallSrv.
    """
    global challSrvProcess
    if challSrvProcess is not None:
        raise(Exception("startChallSrv called more than once"))

    # NOTE(@cpu): We specify explicit bind addresses for -https01 and
    # --tlsalpn01 here to allow HTTPS HTTP-01 responses on 443 for on interface
    # and TLS-ALPN-01 responses on 443 for another interface. The choice of
    # which is used is controlled by mock DNS data added by the relevant
    # integration tests.
    challSrvProcess = run([
        'pebble-challtestsrv',
        '--defaultIPv4', os.environ.get("FAKE_DNS"),
        '-defaultIPv6', '',
        '--dns01', ':8053,:8054',
        '--management', ':8055',
        '--http01', '10.77.77.77:80',
        '-https01', '10.77.77.77:443',
        '--tlsalpn01', '10.88.88.88:443'],
        None)
    # Wait for the pebble-challtestsrv management port.
    if not waitport(8055, ' '.join(challSrvProcess.args)):
        return False

def stopChallSrv():
    """
    Stop the running pebble-challtestsrv (if any) and wait for it to terminate.
    See also startChallSrv.
    """
    global challSrvProcess
    if challSrvProcess is None:
        return
    if challSrvProcess.poll() is None:
        challSrvProcess.send_signal(signal.SIGTERM)
        challSrvProcess.wait()
    challSrvProcess = None

@atexit.register
def stop():
    # When we are about to exit, send SIGTERM to each subprocess and wait for
    # them to nicely die. This reflects the restart process in prod and allows
    # us to exercise the graceful shutdown code paths.
    global processes
    for p in reversed(processes):
        if p.poll() is None:
            p.send_signal(signal.SIGTERM)
            p.wait()
    processes = []

    # Also stop the challenge test server
    stopChallSrv()
