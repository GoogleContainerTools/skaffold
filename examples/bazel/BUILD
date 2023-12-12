load("@rules_oci//oci:defs.bzl", "oci_tarball")

oci_tarball(
    name = "hello.tar",
    image = "@hello//:hello",
    repo_tags = ["hello:latest"],
)
