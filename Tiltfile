config.define_string("num", False, "Number of guardian nodes to run")
cfg = config.parse()
num_guardians = int(cfg.get("num", "5"))

# protos

local_resource(
    name = "proto-gen",
    deps = "./proto",
    cmd = "./generate-protos.sh",
)

# bridge

docker_build(
    ref = "guardiand-image",
    context = "bridge",
    dockerfile = "bridge/Dockerfile",
)

def build_bridge_yaml():
    bridge_yaml = read_yaml_stream("devnet/bridge.yaml")

    for obj in bridge_yaml:
        if obj['kind'] == 'StatefulSet' and obj['metadata']['name'] == 'guardian':
            obj['spec']['replicas'] = num_guardians
            container = obj['spec']['template']['spec']['containers'][0]
            if container['name'] != 'guardiand':
                fail("container 0 is not guardiand")
            container['command'] += ['-devNumGuardians', str(num_guardians)]

    return encode_yaml_stream(bridge_yaml)

k8s_yaml(build_bridge_yaml())

k8s_resource("guardian", resource_deps=["proto-gen"])

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
