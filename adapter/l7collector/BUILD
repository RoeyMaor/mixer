package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["l7collector.go"],
    visibility = ["//visibility:public"],
    deps = [
        "//adapter/l7collector/config:go_default_library",
        "//pkg/adapter:go_default_library",
        "@org_uber_go_zap//:go_default_library",
        "@org_uber_go_zap//zapcore:go_default_library",
        "@com_github_golang_glog//:go_default_library",
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["l7collector_test.go"],
    library = ":go_default_library",
    deps = [
        "//adapter/l7collector/config:go_default_library",
        "//pkg/adapter:go_default_library",
        "//pkg/adapter/test:go_default_library",
        "@org_uber_go_zap//:go_default_library",
        "@org_uber_go_zap//zapcore:go_default_library",
    ],
)
