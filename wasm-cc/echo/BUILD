load("@io_bazel_rules_go//proto:def.bzl", "go_proto_library")

proto_library(
    name = "echo_proto",
    srcs = ["echo.proto"],
    visibility = ["//visibility:public"],
)

cc_proto_library(
    name = "echo_cc_proto",
    deps = [
        ":echo_proto",
    ],
    visibility = ["//visibility:public"],
)
