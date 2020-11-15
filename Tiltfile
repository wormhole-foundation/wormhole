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
#      rust+1.45
#       +                                                           +-----------------+
#       +-> Dockerfile.agent    +->  solana-agent  +--------+-----> | [agent]         |
#       |                                                   |  +--> |    guardian-N   |
#       +-> solana/Dockerfile   +->  solana-contract +---+  |  |    +-- --------------+
#       |                                                |  |  |
#       +-> third_party/solana/Dockerfile <--------------+  |  |
#                              +                            |  |    +-----------------+
#                              +-->  solana-devnet  +-------|-----> |  solana-devnet  |
#      golang:1.15.3                                        +-----> | [setup]         |
#       +                                                      |    +-----------------+
#       +-> bridge/Dockerfile   +->  guardiand-image +---------+
#
#
#      node:lts-alpine
#       +                                                           +-----------------+
#       +-> ethereum/Dockerfile +->  eth+node  +------------------> |    eth|devnet   |
#                                                                   +-----------------+
#

config.define_string("num", False, "Number of guardian nodes to run")
cfg = config.parse()
num_guardians = int(cfg.get("num", "5"))

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

k8s_yaml(build_bridge_yaml())

k8s_resource("guardian", resource_deps = ["proto-gen"])

# solana agent and cli (runs alongside bridge)

docker_build(
    ref = "solana-agent",
    context = ".",
    only = ["./proto", "./solana"],
    dockerfile = "Dockerfile.agent",

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

docker_build(
    ref = "solana-devnet",
    context = "third_party/solana",
    dockerfile = "third_party/solana/Dockerfile",
)

k8s_yaml("devnet/solana-devnet.yaml")

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

k8s_yaml("devnet/eth-devnet.yaml")

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

k8s_yaml("devnet/web.yaml")

k8s_resource("web", port_forwards=[
    port_forward(3000, name="Experimental Web UI [:3000]")
])

# terra devnet

k8s_yaml("devnet/terra-devnet.yaml")

k8s_resource("terra-lcd", port_forwards=[
    port_forward(1317, name="Terra LCD interface [:1317]")
])
k8s_resource("terra-terrad")