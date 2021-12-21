# This Tiltfile contains the deployment and build config for the Wormhole devnet.
#
#  We use Buildkit cache mounts and careful layering to avoid unnecessary rebuilds - almost
#  all source code changes result in small, incremental rebuilds. Dockerfiles are written such
#  that, for example, changing the contract source code won't cause Solana itself to be rebuilt.
#

load("ext://namespace", "namespace_create", "namespace_inject")
load("ext://secret", "secret_yaml_generic")

allow_k8s_contexts("ci")

# Disable telemetry by default
analytics_settings(False)

# Runtime configuration
config.define_bool("ci", False, "We are running in CI")
config.define_bool("manual", False, "Set TRIGGER_MODE_MANUAL by default")

config.define_string("num", False, "Number of guardian nodes to run")

# You do not usually need to set this argument - this argument is for debugging only. If you do use a different
# namespace, note that the "wormhole" namespace is hardcoded in tests and don't forget specifying the argument
# when running "tilt down".
#
config.define_string("namespace", False, "Kubernetes namespace to use")

# These arguments will enable writing Guardian events to a cloud BigTable instance.
# Writing to a cloud BigTable is optional. These arguments are not required to run the devnet.
config.define_string("gcpProject", False, "GCP project ID for BigTable persistence")
config.define_string("bigTableKeyPath", False, "Path to BigTable json key file")

# When running Tilt on a server, this can be used to set the public hostname Tilt runs on
# for service links in the UI to work.
config.define_string("webHost", False, "Public hostname for port forwards")

# Components
config.define_bool("pyth", False, "Enable Pyth-to-Wormhole component")
config.define_bool("explorer", False, "Enable explorer component")
config.define_bool("bridge_ui", False, "Enable bridge UI component")
config.define_bool("e2e", False, "Enable E2E testing stack")

cfg = config.parse()
num_guardians = int(cfg.get("num", "1"))
namespace = cfg.get("namespace", "wormhole")
gcpProject = cfg.get("gcpProject", "local-dev")
bigTableKeyPath = cfg.get("bigTableKeyPath", "./event_database/devnet_key.json")
webHost = cfg.get("webHost", "localhost")
ci = cfg.get("ci", False)
pyth = cfg.get("pyth", ci)
explorer = cfg.get("explorer", ci)
bridge_ui = cfg.get("bridge_ui", ci)
e2e = cfg.get("e2e", ci)

if cfg.get("manual", False):
    trigger_mode = TRIGGER_MODE_MANUAL
else:
    trigger_mode = TRIGGER_MODE_AUTO

# namespace

if not ci:
    namespace_create(namespace)

def k8s_yaml_with_ns(objects):
    return k8s_yaml(namespace_inject(objects, namespace))

# protos

proto_deps = ["./proto", "./generate-protos.sh", "buf.yaml", "buf.gen.yaml"]

local_resource(
    name = "proto-gen",
    deps = proto_deps,
    cmd = "tilt docker build -- --target go-export -f Dockerfile.proto -o type=local,dest=node .",
    env = {"DOCKER_BUILDKIT": "1"},
    trigger_mode = trigger_mode,
)

local_resource(
    name = "proto-gen-web",
    deps = proto_deps,
    resource_deps = ["proto-gen"],
    cmd = "tilt docker build -- --target node-export -f Dockerfile.proto -o type=local,dest=. .",
    env = {"DOCKER_BUILDKIT": "1"},
    trigger_mode = trigger_mode,
)

local_resource(
    name = "teal-gen",
    deps = ["staging/algorand/teal"],
    cmd = "tilt docker build -- --target teal-export -f Dockerfile.teal -o type=local,dest=. .",
    env = {"DOCKER_BUILDKIT": "1"},
    trigger_mode = trigger_mode,
)

# wasm

local_resource(
    name = "wasm-gen",
    deps = ["solana"],
    dir = "solana",
    cmd = "tilt docker build -- -f Dockerfile.wasm -o type=local,dest=.. .",
    env = {"DOCKER_BUILDKIT": "1"},
    trigger_mode = trigger_mode,
)

# node

if explorer:
    k8s_yaml_with_ns(
        secret_yaml_generic(
            "node-bigtable-key",
            from_file = "bigtable-key.json=" + bigTableKeyPath,
        ),
    )

docker_build(
    ref = "guardiand-image",
    context = "node",
    dockerfile = "node/Dockerfile",
)

def build_node_yaml():
    node_yaml = read_yaml_stream("devnet/node.yaml")

    for obj in node_yaml:
        if obj["kind"] == "StatefulSet" and obj["metadata"]["name"] == "guardian":
            obj["spec"]["replicas"] = num_guardians
            container = obj["spec"]["template"]["spec"]["containers"][0]
            if container["name"] != "guardiand":
                fail("container 0 is not guardiand")
            container["command"] += ["--devNumGuardians", str(num_guardians)]

            if explorer:
                container["command"] += [
                    "--bigTablePersistenceEnabled",
                    "--bigTableInstanceName",
                    "wormhole",
                    "--bigTableTableName",
                    "v2Events",
                    "--bigTableTopicName",
                    "new-vaa-devnet",
                    "--bigTableKeyPath",
                    "/tmp/mounted-keys/bigtable-key.json",
                    "--bigTableGCPProject",
                    gcpProject,
                ]

    return encode_yaml_stream(node_yaml)

k8s_yaml_with_ns(build_node_yaml())

k8s_resource(
    "guardian",
    resource_deps = ["proto-gen", "solana-devnet"],
    port_forwards = [
        port_forward(6060, name = "Debug/Status Server [:6060]", host = webHost),
        port_forward(7070, name = "Public gRPC [:7070]", host = webHost),
        port_forward(7071, name = "Public REST [:7071]", host = webHost),
        port_forward(2345, name = "Debugger [:2345]", host = webHost),
    ],
    trigger_mode = trigger_mode,
)

# spy
k8s_yaml_with_ns("devnet/spy.yaml")

k8s_resource(
    "spy",
    resource_deps = ["proto-gen", "guardian"],
    port_forwards = [
        port_forward(6061, container_port = 6060, name = "Debug/Status Server [:6061]", host = webHost),
        port_forward(7072, name = "Spy gRPC [:7072]", host = webHost),
    ],
    trigger_mode = trigger_mode,
)

# solana client cli (used for devnet setup)

docker_build(
    ref = "bridge-client",
    context = ".",
    only = ["./proto", "./solana", "./ethereum", "./clients"],
    dockerfile = "Dockerfile.client",
    # Ignore target folders from local (non-container) development.
    ignore = ["./solana/*/target"],
)

# solana smart contract

docker_build(
    ref = "solana-contract",
    context = "solana",
    dockerfile = "solana/Dockerfile",
)

# solana local devnet

k8s_yaml_with_ns("devnet/solana-devnet.yaml")

k8s_resource(
    "solana-devnet",
    resource_deps = ["wasm-gen"],
    port_forwards = [
        port_forward(8899, name = "Solana RPC [:8899]", host = webHost),
        port_forward(8900, name = "Solana WS [:8900]", host = webHost),
        port_forward(9000, name = "Solana PubSub [:9000]", host = webHost),
    ],
    trigger_mode = trigger_mode,
)

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

if pyth:
    # pyth autopublisher
    docker_build(
        ref = "pyth",
        context = ".",
        dockerfile = "third_party/pyth/Dockerfile.pyth",
    )
    k8s_yaml_with_ns("./devnet/pyth.yaml")

    k8s_resource("pyth", resource_deps = ["solana-devnet"], trigger_mode = trigger_mode)

    # pyth2wormhole client autoattester
    docker_build(
        ref = "p2w-attest",
        context = ".",
        only = ["./solana", "./third_party"],
        dockerfile = "./third_party/pyth/Dockerfile.p2w-attest",
        ignore = ["./solana/*/target"],
    )

    k8s_yaml_with_ns("devnet/p2w-attest.yaml")
    k8s_resource(
        "p2w-attest",
        resource_deps = ["solana-devnet", "pyth", "guardian"],
        port_forwards = [],
        trigger_mode = trigger_mode,
    )

k8s_yaml_with_ns("devnet/eth-devnet.yaml")

k8s_resource(
    "eth-devnet",
    port_forwards = [
        port_forward(8545, name = "Ganache RPC [:8545]", host = webHost),
    ],
    trigger_mode = trigger_mode,
)

k8s_resource(
    "eth-devnet2",
    port_forwards = [
        port_forward(8546, name = "Ganache RPC [:8546]", host = webHost),
    ],
    trigger_mode = trigger_mode,
)

if bridge_ui:
    docker_build(
        ref = "bridge-ui",
        context = ".",
        only = ["./ethereum", "./sdk", "./bridge_ui"],
        dockerfile = "bridge_ui/Dockerfile",
        live_update = [
            sync("./bridge_ui/src", "/app/bridge_ui/src"),
        ],
    )

    k8s_yaml_with_ns("devnet/bridge-ui.yaml")

    k8s_resource(
        "bridge-ui",
        resource_deps = ["proto-gen-web", "wasm-gen"],
        port_forwards = [
            port_forward(3000, name = "Bridge UI [:3000]", host = webHost),
        ],
        trigger_mode = trigger_mode,
    )

# algorand
k8s_yaml_with_ns("devnet/algorand.yaml")

docker_build(
    ref = "algorand",
    context = "third_party/algorand",
    dockerfile = "third_party/algorand/Dockerfile",
)

k8s_resource(
    "algorand",
    resource_deps = ["teal-gen"],
    port_forwards = [
        port_forward(4001, name = "Algorand RPC [:4001]", host = webHost),
        port_forward(4002, name = "Algorand KMD [:4002]", host = webHost),
    ],
    trigger_mode = trigger_mode,
)

# e2e
if e2e:
    k8s_yaml_with_ns("devnet/e2e.yaml")

    docker_build(
        ref = "e2e",
        context = "e2e",
        dockerfile = "e2e/Dockerfile",
        network = "host",
    )

    k8s_resource(
        "e2e",
        port_forwards = [
            port_forward(6080, name = "VNC [:6080]", host = webHost, link_path = "/vnc_auto.html"),
        ],
        trigger_mode = trigger_mode,
    )

# bigtable

if explorer:

    k8s_yaml_with_ns("devnet/bigtable.yaml")

    k8s_resource(
        "bigtable-emulator",
        port_forwards = [port_forward(8086, name = "BigTable clients [:8086]", host = webHost)],
        labels = ["explorer"],
        trigger_mode = trigger_mode,
    )

    k8s_resource("pubsub-emulator",
        port_forwards = [port_forward(8085, name = "PubSub listeners [:8085]")],
        labels = ["explorer"],
    )

    docker_build(
        ref = "cloud-functions",
        context = "./event_database/cloud_functions",
        dockerfile = "./event_database/cloud_functions/Dockerfile",
        live_update = [
            sync("./event_database/cloud_functions", "/app"),
        ],
    )
    k8s_resource(
        "cloud-functions",
        resource_deps = ["proto-gen", "bigtable-emulator", "pubsub-emulator"],
        port_forwards = [port_forward(8090, name = "Cloud Functions [:8090]")],
        labels = ["explorer"],
        trigger_mode = trigger_mode,
    )

    # explorer web app
    docker_build(
        ref = "explorer",
        context = "./explorer",
        dockerfile = "./explorer/Dockerfile",
        ignore = ["./explorer/node_modules"],
        live_update = [
            sync("./explorer/src", "/home/node/app/src"),
            sync("./explorer/public", "/home/node/app/public"),
        ],
    )

    k8s_yaml_with_ns("devnet/explorer.yaml")

    k8s_resource(
        "explorer",
        resource_deps = ["proto-gen-web"],
        port_forwards = [
            port_forward(8001, name = "Explorer Web UI [:8001]", host = webHost),
        ],
        labels = ["explorer"],
        trigger_mode = trigger_mode,
    )

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
    "terra-terrad",
    port_forwards = [
        port_forward(26657, name = "Terra RPC [:26657]", host = webHost),
        port_forward(1317, name = "Terra LCD [:1317]", host = webHost),
    ],
    trigger_mode = trigger_mode,
)

k8s_resource(
    "terra-fcd",
    port_forwards = [port_forward(3060, name = "Terra FCD [:3060]", host = webHost)],
    trigger_mode = trigger_mode,
)
