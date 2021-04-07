# This Tiltfile contains the deployment and build config for the Wormhole devnet.
#
#  We use Buildkit cache mounts and careful layering to avoid unnecessary rebuilds - almost
#  all source code changes result in small, incremental rebuilds. Dockerfiles are written such
#  that, for example, changing the contract source code won't cause Solana itself to be rebuilt.
#
#  Graph of dependencies between Dockerfiles, image refs and k8s StatefulSets:
#
#      Dockerfile                    Image ref                      StatefulSet
#      +------------------------------------------------------------------------------+
#      rust-1.*
#       +                                                           +-----------------+
#       +-> Dockerfile.agent    +->  solana-agent  +--------+-----> | [agent]         |
#       |                                                   |  +--> |    guardian-N   |
#       +-> solana/Dockerfile   +->  solana-contract +---+  |  |    +-- --------------+
#                                                        |  |  |
#                                                        |  |  |
#                                                        |  |  |    +-----------------+
#                                                        +--|-----> |  solana-devnet  |
#      golang:1.*                                           +-----> | [setup]         |
#       +                                                      |    +-----------------+
#       +-> bridge/Dockerfile   +->  guardiand-image +---------+
#
#
#      node:lts-alpine
#       +                                                           +-----------------+
#       +-> ethereum/Dockerfile +->  eth-node  +------------------> |    eth-devnet   |
#                                                                   +-----------------+
#

load('ext://namespace', 'namespace_inject', 'namespace_create')

# Runtime configuration

config.define_string("num", False, "Number of guardian nodes to run")

# You do not usually need to set this argument - this argument is for debugging only. If you do use a different
# namespace, note that the "wormhole" namespace is hardcoded in the e2e test and don't forget specifying the argument
# when running "tilt down".
#
config.define_string("namespace", False, "Kubernetes namespace to use")

cfg = config.parse()
num_guardians = int(cfg.get("num", "5"))
namespace = cfg.get("namespace", "wormhole")

# namespace

namespace_create(namespace)

def k8s_yaml_with_ns(objects):
    return k8s_yaml(namespace_inject(objects, namespace))

# protos

local_resource(
    name = "proto-gen",
    deps = ["./proto", "./generate-protos.sh"],
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
        if obj["kind"] == "StatefulSet" and obj["metadata"]["name"] == "guardian":
            obj["spec"]["replicas"] = num_guardians
            container = obj["spec"]["template"]["spec"]["containers"][0]
            if container["name"] != "guardiand":
                fail("container 0 is not guardiand")
            container["command"] += ["--devNumGuardians", str(num_guardians)]

    return encode_yaml_stream(bridge_yaml)


k8s_yaml_with_ns(build_bridge_yaml())

k8s_resource("guardian", resource_deps=["proto-gen", "solana-devnet"], port_forwards=[
    port_forward(6060, name="Debug/Status Server [:6060]"),
])

# solana agent and cli (runs alongside bridge)

docker_build(
    ref="solana-agent",
    context=".",
    only=["./proto", "./solana"],
    dockerfile="Dockerfile.agent",

    # Ignore target folders from local (non-container) development.
    ignore = ["./solana/target", "./solana/agent/target", "./solana/cli/target"],
)

# solana smart contract

docker_build(
    ref = "solana-contract",
    context = "solana",
    dockerfile = "solana/Dockerfile",
)

# solana local devnet

k8s_yaml_with_ns("devnet/solana-devnet.yaml")

k8s_resource("solana-devnet", port_forwards=[
    port_forward(8899, name="Solana RPC [:8899]"),
    port_forward(8900, name="Solana WS [:8900]"),
    port_forward(9000, name="Solana PubSub [:9000]"),
])

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

k8s_yaml_with_ns("devnet/eth-devnet.yaml")

k8s_resource("eth-devnet", port_forwards=[
    port_forward(8545, name="Ganache RPC [:8545]")
])

# web frontend

docker_build(
    ref = "web",
    context = "./web",
    dockerfile = "./web/Dockerfile",
    ignore = ["./web/node_modules"],
    live_update = [
        sync("./web/src", "/home/node/app/src"),
        sync("./web/public", "/home/node/app/public"),
        sync("./web/contracts", "/home/node/app/contracts"),
    ],
)

k8s_yaml_with_ns("devnet/web.yaml")

k8s_resource("web", port_forwards=[
    port_forward(3000, name="Experimental Web UI [:3000]")
])

# terra devnet

docker_build(
    ref = "terra-image",
    context = "./terra/devnet",
    dockerfile = "terra/devnet/Dockerfile",
)

docker_build(
    ref = "terra-contracts",
    context = "./terra",
    dockerfile = "./terra/Dockerfile",
)

k8s_yaml_with_ns("devnet/terra-devnet.yaml")

k8s_resource(
    "terra-lcd",
    port_forwards=[port_forward(1317, name="Terra LCD interface [:1317]")]
)

k8s_resource(
    "terra-terrad",
    port_forwards=[port_forward(26657, name="Terra RPC [:26657]")]
)
