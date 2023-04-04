# Load testing the OCSP signing components.

Here are instructions on how to realistically load test the OCSP signing
components of Boulder, exercising the pkcs11key, boulder-ca, and
ocsp-updater components.

Set up a SoftHSM instance running pkcs11-daemon on some remote host with more
CPUs than your local machine. Easiest way to do this is to clone the Boulder
repo, and on the remote machine run:

    remote-machine$ docker compose run -p 5657:5657 bhsm

Check that the port is open:

    local-machine$ nc -zv remote-machine 5657
    Connection to remote-machine 5657 port [tcp/*] succeeded!

Edit docker-compose.yml to change these in the "boulder" section's "env":

    PKCS11_PROXY_SOCKET: tcp://remote-machine:5657
    FAKE_DNS: 172.17.0.1

Run the pkcs11key benchmark to check raw signing speed at various settings for SESSIONS:

    local-machine$ docker compose run -e SESSIONS=4 -e MODULE=/usr/local/lib/softhsm/libsofthsm2.so --entrypoint /go/src/github.com/letsencrypt/pkcs11key/test.sh boulder

Initialize the tokens for use by Boulder:

    local-machine$ docker compose run --entrypoint "softhsm --module /usr/local/lib/softhsm/libsofthsm2.so --init-token --pin 5678 --so-pin 1234 --slot 0 --label intermediate" boulder
    local-machine$ docker compose run --entrypoint "softhsm --module /usr/local/lib/softhsm/libsofthsm2.so --init-token --pin 5678 --so-pin 1234 --slot 1 --label root" boulder

Configure Boulder to always consider all OCSP responses instantly stale, so it
will sign new ones as fast as it can. Edit "ocspMinTimeToExpiry" in
test/config/ocsp-updater.json (or test/config-next/ocsp-updater.json):

    "ocspMinTimeToExpiry": "0h",

Run a local Boulder instance:

    local-machine$ docker compose up

Issue a bunch of certificates with chisel.py, ideally a few thousand
(corresponding to the default batch size of 5000 in ocsp-updater.json, to make
sure each batch is maxed out):

    local-machine$ while true; do python test/chisel.py $(openssl rand -hex 4).com ; done

Use the local Prometheus instance to graph the number of complete gRPC calls:

http://localhost:9090/graph?g0.range_input=5m&g0.expr=irate(grpc_client_handled_total%7Bgrpc_method%3D%22GenerateOCSP%22%7D%5B1m%5D)&g0.tab=0

If you vary the NumSessions config value in test/config/ca.json, you should see
the signing speed vary linearly, up to the number of cores in the remote
machine. Note that hyperthreaded cores look like 2 cores but may only perform
as 1 (needs testing).

Keep in mind that round-trip time between your local machine and your HSM
machine greatly impact signing speed.
