client_addr = "10.55.55.10"
bind_addr   = "10.55.55.10"
log_level   = "INFO"
ui_config {
  enabled = true
}
ports {
  dns = 53
}

services {
  id      = "akamai-purger-a"
  name    = "akamai-purger"
  address = "10.77.77.77"
  port    = 9099
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "akamai-purger-b"
  name    = "akamai-purger"
  address = "10.88.88.88"
  port    = 9099
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "boulder-a"
  name    = "boulder"
  address = "10.77.77.77"
}

services {
  id      = "boulder-a"
  name    = "boulder"
  address = "10.88.88.88"
}

services {
  id      = "ca-a"
  name    = "ca"
  address = "10.77.77.77"
  port    = 9093
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "ca-b"
  name    = "ca"
  address = "10.88.88.88"
  port    = 9093
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "ca1"
  name    = "ca1"
  address = "10.77.77.77"
  port    = 9093
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "ca2"
  name    = "ca2"
  address = "10.88.88.88"
  port    = 9093
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "ca-ocsp-a"
  name    = "ca-ocsp"
  address = "10.77.77.77"
  port    = 9096
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "ca-ocsp-b"
  name    = "ca-ocsp"
  address = "10.88.88.88"
  port    = 9096
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "ca-crl-a"
  name    = "ca-crl"
  address = "10.77.77.77"
  port    = 9106
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "ca-crl-b"
  name    = "ca-crl"
  address = "10.88.88.88"
  port    = 9106
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "crl-storer-a"
  name    = "crl-storer"
  address = "10.77.77.77"
  port    = 9109
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "crl-storer-b"
  name    = "crl-storer"
  address = "10.88.88.88"
  port    = 9109
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "dns-a"
  name    = "dns"
  address = "10.77.77.77"
  port    = 8053
  tags    = ["udp"] // Required for SRV RR support in VA RVA.
}

services {
  id      = "dns-b"
  name    = "dns"
  address = "10.88.88.88"
  port    = 8054
  tags    = ["udp"] // Required for SRV RR support in VA RVA.
}

services {
  id      = "nonce-a"
  name    = "nonce"
  address = "10.77.77.77"
  port    = 9101
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "nonce-b"
  name    = "nonce"
  address = "10.88.88.88"
  port    = 9101
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "nonce1"
  name    = "nonce1"
  address = "10.77.77.77"
  port    = 9101
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "nonce2"
  name    = "nonce2"
  address = "10.88.88.88"
  port    = 9101
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "publisher-a"
  name    = "publisher"
  address = "10.77.77.77"
  port    = 9091
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "publisher-b"
  name    = "publisher"
  address = "10.88.88.88"
  port    = 9091
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "publisher1"
  name    = "publisher1"
  address = "10.77.77.77"
  port    = 9091
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "publisher2"
  name    = "publisher2"
  address = "10.88.88.88"
  port    = 9091
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "ra-a"
  name    = "ra"
  address = "10.77.77.77"
  port    = 9094
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "ra-b"
  name    = "ra"
  address = "10.88.88.88"
  port    = 9094
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "ra1"
  name    = "ra1"
  address = "10.77.77.77"
  port    = 9094
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "ra2"
  name    = "ra2"
  address = "10.88.88.88"
  port    = 9094
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "rva1-a"
  name    = "rva1"
  address = "10.77.77.77"
  port    = 9097
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "rva1-b"
  name    = "rva1"
  address = "10.77.77.77"
  port    = 9098
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "sa-a"
  name    = "sa"
  address = "10.77.77.77"
  port    = 9095
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "sa-b"
  name    = "sa"
  address = "10.88.88.88"
  port    = 9095
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "sa1"
  name    = "sa1"
  address = "10.77.77.77"
  port    = 9095
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "sa2"
  name    = "sa2"
  address = "10.88.88.88"
  port    = 9095
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "va-a"
  name    = "va"
  address = "10.77.77.77"
  port    = 9092
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "va-b"
  name    = "va"
  address = "10.88.88.88"
  port    = 9092
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "va1"
  name    = "va1"
  address = "10.77.77.77"
  port    = 9092
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}

services {
  id      = "va2"
  name    = "va2"
  address = "10.88.88.88"
  port    = 9092
  tags    = ["tcp"] // Required for SRV RR support in gRPC DNS resolution.
}
