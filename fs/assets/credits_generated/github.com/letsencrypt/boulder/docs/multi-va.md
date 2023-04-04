# Multi-VA implementation

Boulder supports a multi-perspective validation feature intended to increase
resilience against local network hijacks and BGP attacks. It is currently
[deployed in a production
capacity](https://letsencrypt.org/2020/02/19/multi-perspective-validation.html)
by Let's Encrypt.

If you follow the [Development Instructions](https://github.com/letsencrypt/boulder#development)
to set up a Boulder environment in Docker and then change your `docker-compose.yml`'s
`BOULDER_CONFIG_DIR` to `test/config-next` instead of `test/config` you'll have
a Boulder environment configured with two primary VA instances (validation
requests are load balanced across the two) and two remote VA instances (each
primary VA will ask both remote VAs to perform matching validations for each
primary validation). Of course this is a development environment so both the
primary and remote VAs are all running on one host.

The primary and remote VAs are both the same piece of software, the `boulder-va`
service ([cmd here](https://github.com/letsencrypt/boulder/tree/main/cmd/boulder-va),
[package here](https://github.com/letsencrypt/boulder/tree/main/va)).
The boulder-ra uses [the same RPC interface](https://github.com/letsencrypt/boulder/blob/ea231adc36746cce97f860e818c2cdf92f060543/va/proto/va.proto#L8-L10)
to ask for a primary validation as the primary VA uses to ask a remote VA for a
confirmation validation.

Primary VA instances know they are a primary based on the presence of the
`"remoteVAs"` configuration element. If present it specifies gRPC service
addresses for other VA instances to use as remotes. There's also a handful of
feature flags that control how the primary VAs handle the remote VAs.

In the development environment with `config-next` the two primary VAs are `va1.service.consul:9092` and
`va2.service.consul:9092` and use
[`test/config-next/va.json`](https://github.com/letsencrypt/boulder/blob/ea231adc36746cce97f860e818c2cdf92f060543/test/config-next/va.json)
as their configuration. This config file specifies two `"remoteVA"s`,
`rva1.service.consul:9097` and `va2.service.consul:9098` and enforces
[that a maximum of 1 of the 2 remote VAs disagree](https://github.com/letsencrypt/boulder/blob/ea231adc36746cce97f860e818c2cdf92f060543/test/config-next/va.json#L44)
with the primary VA for all validations. The remote VA instances use
[`test/config-next/va-remote-a.json`](https://github.com/letsencrypt/boulder/blob/ea231adc36746cce97f860e818c2cdf92f060543/test/config-next/va-remote-a.json)
and
[`test/config-next/va-remote-b.json`](https://github.com/letsencrypt/boulder/blob/ea231adc36746cce97f860e818c2cdf92f060543/test/config-next/va-remote-b.json)
as their config files.

There are two feature flags that control whether multi-VA takes effect:
MultiVAFullResults and EnforceMultiVA. If MultiVAFullResults is enabled
then each primary validation will also send out remote validation requests, and
wait for all the results to come in, so we can log the results for analysis. If
EnforceMultiVA is enabled, we require that almost all remote validation requests
succeed. The primary VA's "maxRemoteValidationFailures" config field specifies
how many remote VAs can fail before the primary VA considers overall validation
a failure. It should be strictly less than the number of remote VAs.

Validation is also controlled by the "multiVAPolicyFile" config field on the
primary VA. This specifies a file that can contain temporary overrides for
domains or accounts that fail under multi-va. Over time those temporary
overrides will be removed.

There are some integration tests that test this end to end. The most relevant is
probably
[`test_http_multiva_threshold_fail`](https://github.com/letsencrypt/boulder/blob/ea231adc36746cce97f860e818c2cdf92f060543/test/v2_integration.py#L876-L908).
It tests that a HTTP-01 challenge made to a webserver that only gives the
correct key authorization to the primary VA and not the remotes will fail the
multi-perspective validation.
