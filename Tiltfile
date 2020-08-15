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

# solana smart contract components

docker_build(
    ref = "solana-agent",
    context = ".",
    only = ["./proto", "./solana"],
    dockerfile = "Dockerfile.agent",
)

# solana local devnet

docker_build(
    ref = "solana-devnet",
    context = "third_party/solana",
    dockerfile = "third_party/solana/Dockerfile",
)

k8s_yaml("devnet/solana-devnet.yaml")

k8s_resource("solana-devnet")

# eth devnet

# TODO: Slow - takes ~30s to rebuild on a no-op, even with caching, because npm sucks.
# We might wamt to add excludes here and use file sync for smart contract developmeent.
docker_build(
    ref = "eth-node",
    context = "ethereum",
    dockerfile = "ethereum/Dockerfile",
)

k8s_yaml("devnet/eth-devnet.yaml")

k8s_resource("eth-devnet")
