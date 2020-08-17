# protos

local_resource(
    name = "proto-gen",
    deps = "./proto",
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

    # Ignore target folders from local (non-container) development.
    ignore = ["./solana/target", "./solana/agent/target", "./solana/cli/target"],
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

docker_build(
    ref = "eth-node",
    context = "./ethereum",
    dockerfile = "./ethereum/Dockerfile",

    # ignore local node_modules (in case they're present)
    ignore = ["./ethereum/node_modules"],

    # sync external scripts for incremental development
    # (everything else needs to be restarted from scratch for determinism)
    #
    # This relies on --update-mode=exec to work properly with a non-root user.
    # https://github.com/tilt-dev/tilt/issues/3708
    live_update = [
        sync("./ethereum/src", "/home/node/app/src"),
    ],

)

k8s_yaml("devnet/eth-devnet.yaml")

k8s_resource("eth-devnet")
