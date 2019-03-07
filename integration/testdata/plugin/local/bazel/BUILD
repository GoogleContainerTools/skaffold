load("@io_bazel_rules_docker//go:image.bzl", "go_image")

go_image(
    name = "skaffold_example",
    srcs = ["main.go"],
    goos = "linux",
    goarch = "amd64",
    static = "on",
)
