package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["token.go","adapterConfig.go","issuer.go","jwk.go","parser.go","localJWKTestServer.go"],
    visibility = ["//visibility:public"],
    deps = [
        "//adapter/token/config:go_default_library",
        "//pkg/adapter:go_default_library",
        "//pkg/status:go_default_library",
        "@com_github_googleapis_googleapis//:google/rpc",
        "@com_github_golang_glog//:go_default_library",
        "@com_github_asaskevich_govalidator//:go_default_library",
        "@com_github_dgrijalva_jwt-go//:go_default_library"
    ],
)

go_test(
    name = "go_default_test",
    size = "small",
    srcs = ["token_test.go"],
    library = ":go_default_library",
    deps = [
        "//pkg/adapter:go_default_library",
        "//pkg/adapterManager:go_default_library",
        "//pkg/config:go_default_library",
        "//pkg/adapter/test:go_default_library",
    ],
)