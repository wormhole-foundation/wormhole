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
# We might want to add excludes here and use file sync for smart contract developmeent.
docker_build(
    ref = "eth-node",
    context = "./ethereum",
    dockerfile = "./ethereum/Dockerfile",

    # ignore local node_modules (in case they're present)
    ignore = ["./ethereum/node_modules"],

    # sync smart contract changes to running container for incremental development
    # (rebuilding the container is way too slow, thanks npm!)
    #
    # This relies on --update-mode=exec to work properly with a non-root user.
    # https://github.com/tilt-dev/tilt/issues/3708
    live_update = [
        sync("./ethereum", "/home/node/app"),
    ],
)

k8s_yaml("devnet/eth-devnet.yaml")

k8s_resource("eth-devnet")
