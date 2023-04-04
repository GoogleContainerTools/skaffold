# Boulder divergences from ACME

While Boulder attempts to implement the ACME specification ([RFC 8555]) as strictly as possible there are places at which we will diverge from the letter of the specification for various reasons. This document describes the difference between RFC 8555 and Boulder's implementation of ACME, informally called ACMEv2 and available at https://acme-v02.api.letsencrypt.org/directory. Boulder's implementation of ACMEv1 differs substantially from the final RFC. Documentation for Boulder's ACMEv1 support is available in [acme-divergences-v1.md](acme-divergences-v1.md). A listing of RFC conformant design decisions that may differ from other ACME servers is listed in [implementation_details](https://github.com/letsencrypt/boulder/blob/main/docs/acme-implementation_details.md).


Presently, Boulder diverges from the RFC 8555 ACME spec in the following ways:

## [Section 6.3](https://tools.ietf.org/html/rfc8555#section-6.3)

Boulder supports POST-as-GET but does not mandate it by default for requests
that simply fetch a resource (certificate, order, authorization, or challenge).
This behavior is configurable with a flag: Let's Encrypt's Staging environment
does mandate POST-as-GET, while the Production environment does not.

## [Section 6.6](https://tools.ietf.org/html/rfc8555#section-6.6)

Boulder does not provide a `Retry-After` header when a user hits a rate-limit, nor does it provide `Link` headers to further documentation on rate-limiting.

## [Section 6.7](https://tools.ietf.org/html/rfc8555#section-6.7)

Boulder uses `invalidEmail` in place of the error `invalidContact`.

Boulder does not implement the `unsupportedContact` and `dnssec` errors.

## [Section 7.1.2](https://tools.ietf.org/html/rfc8555#section-7.1.2)

Boulder does not supply the `orders` field on account objects. We intend to
support this non-essential feature in the future. Please follow Boulder Issue
[#3335](https://github.com/letsencrypt/boulder/issues/3335).

## [Section 7.4](https://tools.ietf.org/html/rfc8555#section-7.4)

Boulder does not accept the optional `notBefore` and `notAfter` fields of a
`newOrder` request paylod.

## [Section 7.4.1](https://tools.ietf.org/html/rfc8555#section-7.4.1)

Pre-authorization is an optional feature and we have no plans to implement it.
V2 clients should use order based issuance without pre-authorization.

## [Section 7.4.2](https://tools.ietf.org/html/rfc8555#section-7.4.2)

Boulder does not process `Accept` headers for `Content-Type` negotiation when retrieving certificates.

## [Section 8.2](https://tools.ietf.org/html/rfc8555#section-8.2)

Boulder does not implement the ability to retry challenges or the `Retry-After` header.

[RFC 8555]: https://tools.ietf.org/html/rfc8555
