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

# Components
config.define_bool("pyth", False, "Enable Pyth-to-Wormhole component")
config.define_bool("explorer", False, "Enable explorer component")

cfg = config.parse()
num_guardians = int(cfg.get("num", "1"))
namespace = cfg.get("namespace", "wormhole")
gcpProject = cfg.get("gcpProject", "local-dev")
bigTableKeyPath = cfg.get("bigTableKeyPath", "./event_database/devnet_key.json")
ci = cfg.get("ci", False)
pyth = cfg.get("pyth", ci)
explorer = cfg.get("explorer", ci)

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
)

local_resource(
    name = "proto-gen-web",
    deps = proto_deps,
    resource_deps = ["proto-gen"],
    cmd = "tilt docker build -- --target node-export -f Dockerfile.proto -o type=local,dest=. .",
    env = {"DOCKER_BUILDKIT": "1"},
)

# wasm

local_resource(
    name = "wasm-gen",
    deps = ["solana"],
    dir = "solana",
    cmd = "tilt docker build -- -f Dockerfile.wasm -o type=local,dest=.. .",
    env = {"DOCKER_BUILDKIT": "1"},
)

# node

if explorer:
    k8s_yaml_with_ns(
        secret_yaml_generic(
            "bridge-bigtable-key",
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
                    "--bigTableKeyPath",
                    "/tmp/mounted-keys/bigtable-key.json",
                    "--bigTableGCPProject",
                    gcpProject,
                ]

    return encode_yaml_stream(node_yaml)

k8s_yaml_with_ns(build_node_yaml())

k8s_resource("guardian", resource_deps = ["proto-gen", "solana-devnet"], port_forwards = [
    port_forward(6060, name = "Debug/Status Server [:6060]"),
    port_forward(7070, name = "Public gRPC [:7070]"),
    port_forward(7071, name = "Public REST [:7071]"),
])

if pyth:
    docker_build(
        ref = "pyth",
        context = ".",
        dockerfile = "third_party/pyth/Dockerfile.pyth",
    )
    k8s_yaml_with_ns("./devnet/pyth.yaml")

    k8s_resource("pyth", resource_deps = ["solana-devnet"])

# publicRPC proxy that allows grpc over http1, for local development

k8s_yaml_with_ns("./devnet/envoy-proxy.yaml")

k8s_resource(
    "envoy-proxy",
    resource_deps = ["guardian"],
    objects = ["envoy-proxy:ConfigMap"],
    port_forwards = [
        port_forward(8080, name = "gRPC proxy for guardian's publicRPC data [:8080]"),
        port_forward(9901, name = "gRPC proxy admin [:9901]"),  # for proxy debugging
    ],
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
        port_forward(8899, name = "Solana RPC [:8899]"),
        port_forward(8900, name = "Solana WS [:8900]"),
        port_forward(9000, name = "Solana PubSub [:9000]"),
    ],
)

# pyth2wormhole client

if pyth:
    docker_build(
        ref = "p2w-client",
        context = ".",
        only = ["./solana", "./third_party"],
        dockerfile = "./third_party/pyth/Dockerfile.p2w-client",

        # Ignore target folders from local (non-container) development.
        ignore = ["./solana/*/target"],
    )

    k8s_yaml_with_ns("devnet/p2w-client.yaml")

    k8s_resource(
        "p2w-client",
        resource_deps = ["solana-devnet", "pyth"],
        port_forwards = [],
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

k8s_yaml_with_ns("devnet/eth-devnet.yaml")

k8s_resource("eth-devnet", port_forwards = [
    port_forward(8545, name = "Ganache RPC [:8545]"),
])

# bigtable

def build_cloud_function(container_name, go_func_name, path, builder):
    # Invokes Tilt's custom_build(), with a Pack command.
    # inspired by https://github.com/tilt-dev/tilt-extensions/tree/master/pack
    caching_ref = container_name + ":tilt-build-pack-caching"
    pack_build_cmd = " ".join([
        "./tools/bin/pack build",
        caching_ref,
        "--path " + path,
        "--builder " + builder,
        "--env " + "GOOGLE_FUNCTION_TARGET=%s" % go_func_name,
        "--env " + "GOOGLE_FUNCTION_SIGNATURE_TYPE=http",
    ])

    if ci:
        # inherit the DOCKER_HOST socket provided by custom_build.
        pack_build_cmd = pack_build_cmd + " --docker-host inherit"

    docker_tag_cmd = "docker tag " + caching_ref + " $EXPECTED_REF"
    custom_build(
        container_name,
        pack_build_cmd + " && " + docker_tag_cmd,
        [path],
    )

if explorer:
    build_cloud_function(
        container_name = "cloud-function-readrow",
        go_func_name = "ReadRow",
        path = "./event_database/cloud_functions",
        builder = "gcr.io/buildpacks/builder:v1",
    )

    local_resource(
        name = "pack-bin",
        cmd = "go build -mod=readonly -o bin/pack github.com/buildpacks/pack/cmd/pack",
        dir = "tools",
    )

    k8s_yaml_with_ns("devnet/bigtable.yaml")

    k8s_resource("bigtable-emulator", port_forwards = [
        port_forward(8086, name = "BigTable clients [:8086]"),
    ])
    k8s_resource(
        "bigtable-readrow",
        resource_deps = ["proto-gen"],
        port_forwards = [port_forward(8090, name = "ReadRow [:8090]")],
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
            port_forward(8001, name = "Explorer Web UI [:8001]"),
        ],
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
    "terra-lcd",
    port_forwards = [port_forward(1317, name = "Terra LCD interface [:1317]")],
)

k8s_resource(
    "terra-terrad",
    port_forwards = [port_forward(26657, name = "Terra RPC [:26657]")],
)

k8s_resource(
    "terra-fcd",
    port_forwards = [port_forward(3060, name = "Terra FCD [:3060]")],
)
