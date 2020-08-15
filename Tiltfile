# protos

local_resource(
    name = "proto-gen",
    deps = "proto/*",
    cmd = "./generate.sh",
)

# bridge

docker_build(
    ref = "guardiand-image",
    context = "bridge",
    dockerfile = "bridge/Dockerfile",
)

k8s_yaml("devnet/bridge.yaml")

k8s_resource("guardian")

# solana-side components

docker_build(
    ref = "solana-agent",
    context = ".",
    only = ["./proto", "./solana"],
    dockerfile = "Dockerfile.agent",
)
