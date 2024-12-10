load("@rules_oci//oci:defs.bzl", "oci_load")

oci_load(
    name = "hello",
    image = "@hello//:hello",
    repo_tags = ["hello:latest"],
)

filegroup(
    name = "hello.tar",
    srcs = [":hello"],
    output_group = "tarball",
)
