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

# Moar updates (default is 3)
update_settings(max_parallel_updates = 10)

# Runtime configuration
config.define_bool("ci", False, "We are running in CI")
config.define_bool("manual", False, "Set TRIGGER_MODE_MANUAL by default")
config.define_bool("m1", False, "Build Solana from source for m1 arm64")

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
config.define_bool("algorand", False, "Enable Algorand component")
config.define_bool("solana", False, "Enable Solana component")
config.define_bool("explorer", False, "Enable explorer component")
config.define_bool("bridge_ui", False, "Enable bridge UI component")
config.define_bool("spy_relayer", False, "Enable spy relayer")
config.define_bool("e2e", False, "Enable E2E testing stack")
config.define_bool("ci_tests", False, "Enable tests runner component")
config.define_bool("bridge_ui_hot", False, "Enable hot loading bridge_ui")
config.define_bool("guardiand_debug", False, "Enable dlv endpoint for guardiand")

cfg = config.parse()
num_guardians = int(cfg.get("num", "1"))
namespace = cfg.get("namespace", "wormhole")
gcpProject = cfg.get("gcpProject", "local-dev")
bigTableKeyPath = cfg.get("bigTableKeyPath", "./event_database/devnet_key.json")
webHost = cfg.get("webHost", "localhost")
algorand = cfg.get("algorand", True)
solana = cfg.get("solana", True)
ci = cfg.get("ci", False)
explorer = cfg.get("explorer", ci)
bridge_ui = cfg.get("bridge_ui", ci)
spy_relayer = cfg.get("spy_relayer", ci)
e2e = cfg.get("e2e", ci)
ci_tests = cfg.get("ci_tests", ci)
guardiand_debug = cfg.get("guardiand_debug", False)
m1 = cfg.get("m1", False)

bridge_ui_hot = not ci

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

proto_deps = ["./proto", "buf.yaml", "buf.gen.yaml"]

local_resource(
    name = "proto-gen",
    deps = proto_deps,
    cmd = "tilt docker build -- --target go-export -f Dockerfile.proto -o type=local,dest=node .",
    env = {"DOCKER_BUILDKIT": "1"},
    labels = ["protobuf"],
    allow_parallel = True,
    trigger_mode = trigger_mode,
)

local_resource(
    name = "proto-gen-web",
    deps = proto_deps + ["buf.gen.web.yaml"],
    resource_deps = ["proto-gen"],
    cmd = "tilt docker build -- --target node-export -f Dockerfile.proto -o type=local,dest=. .",
    env = {"DOCKER_BUILDKIT": "1"},
    labels = ["protobuf"],
    allow_parallel = True,
    trigger_mode = trigger_mode,
)

local_resource(
    name = "const-gen",
    deps = ["scripts", "clients", "ethereum/.env.test"],
    cmd = 'tilt docker build -- --target const-export -f Dockerfile.const -o type=local,dest=. --build-arg num_guardians=%s .' % (num_guardians),
    env = {"DOCKER_BUILDKIT": "1"},
    allow_parallel = True,
    trigger_mode = trigger_mode,
)

# wasm

if solana:
    cmd = "tilt docker build -- -f Dockerfile.wasm -o type=local,dest=.. ."
    if m1:
        cmd = "tilt docker build -- -f Dockerfile.wasm.m1 -o type=local,dest=.. ."
    local_resource(
        name = "wasm-gen",
        deps = ["solana"],
        dir = "solana",
        cmd = cmd,
        env = {"DOCKER_BUILDKIT": "1"},
        labels = ["solana"],
        allow_parallel = True,
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

def command_with_dlv(argv):
    return [
        "/dlv",
        "--listen=0.0.0.0:2345",
        "--accept-multiclient",
        "--headless=true",
        "--api-version=2",
        "--continue=true",
        "exec",
        argv[0],
        "--",
    ] + argv[1:]

def build_node_yaml():
    node_yaml = read_yaml_stream("devnet/node.yaml")

    for obj in node_yaml:
        if obj["kind"] == "StatefulSet" and obj["metadata"]["name"] == "guardian":
            obj["spec"]["replicas"] = num_guardians
            container = obj["spec"]["template"]["spec"]["containers"][0]
            if container["name"] != "guardiand":
                fail("container 0 is not guardiand")
            container["command"] += ["--devNumGuardians", str(num_guardians)]

            if guardiand_debug:
                container["command"] = command_with_dlv(container["command"])
                container["command"] += ["--logLevel=debug"]
                print(container["command"])

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

guardian_resource_deps = ["proto-gen", "eth-devnet", "eth-devnet2", "terra-terrad", "terra2-terrad"]
if solana:
    guardian_resource_deps = guardian_resource_deps + ["solana-devnet"]

k8s_resource(
    "guardian",
    resource_deps = guardian_resource_deps,
    port_forwards = [
        port_forward(6060, name = "Debug/Status Server [:6060]", host = webHost),
        port_forward(7070, name = "Public gRPC [:7070]", host = webHost),
        port_forward(7071, name = "Public REST [:7071]", host = webHost),
        port_forward(2345, name = "Debugger [:2345]", host = webHost),
    ],
    labels = ["guardian"],
    trigger_mode = trigger_mode,
)

# guardian set update - triggered by "tilt args" changes
if num_guardians >= 2 and ci == False:
    local_resource(
        name = "guardian-set-update",
        resource_deps = guardian_resource_deps + ["guardian"],
        deps = ["scripts/send-vaa.sh", "clients/eth"],
        cmd = './scripts/update-guardian-set.sh %s %s %s' % (num_guardians, webHost, namespace),
        labels = ["guardian"],
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
    labels = ["guardian"],
    trigger_mode = trigger_mode,
)

if solana:
    # solana client cli (used for devnet setup)
    dockerfile = "Dockerfile.client"
    if m1:
        dockerfile = "Dockerfile.client.m1"

    docker_build(
        ref = "bridge-client",
        context = ".",
        only = ["./proto", "./solana", "./clients"],
        dockerfile = dockerfile,
        # Ignore target folders from local (non-container) development.
        ignore = ["./solana/*/target"],
    )

    # solana smart contract

    dockerfile = "solana/Dockerfile"
    if m1:
        dockerfile = "solana/Dockerfile.m1"

    docker_build(
        ref = "solana-contract",
        context = "solana",
        dockerfile = dockerfile,
        target = "builder",
        build_args = {"BRIDGE_ADDRESS": "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"}
    )

    # solana local devnet

    k8s_yaml_with_ns("devnet/solana-devnet.yaml")

    k8s_resource(
        "solana-devnet",
        port_forwards = [
            port_forward(8899, name = "Solana RPC [:8899]", host = webHost),
            port_forward(8900, name = "Solana WS [:8900]", host = webHost),
            port_forward(9000, name = "Solana PubSub [:9000]", host = webHost),
        ],
        resource_deps = ["const-gen"],
        labels = ["solana"],
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

if spy_relayer:
    docker_build(
        ref = "redis",
        context = ".",
        only = ["./third_party"],
        dockerfile = "third_party/redis/Dockerfile",
    )

    k8s_yaml_with_ns("devnet/redis.yaml")

    k8s_resource(
        "redis",
        port_forwards = [
            port_forward(6379, name = "Redis Default [:6379]", host = webHost),
        ],
        labels = ["spy-relayer"],
        trigger_mode = trigger_mode,
    )

    docker_build(
        ref = "spy-relay-image",
        context = ".",
        only = ["./relayer/spy_relayer"],
        dockerfile = "relayer/spy_relayer/Dockerfile",
        live_update = []
    )

    k8s_yaml_with_ns("devnet/spy-listener.yaml")

    k8s_resource(
        "spy-listener",
        resource_deps = ["proto-gen", "guardian", "redis"],
        port_forwards = [
            port_forward(6062, container_port = 6060, name = "Debug/Status Server [:6062]", host = webHost),
            port_forward(4201, name = "REST [:4201]", host = webHost),
            port_forward(8082, name = "Prometheus [:8082]", host = webHost),
        ],
        labels = ["spy-relayer"],
        trigger_mode = trigger_mode,
    )

    k8s_yaml_with_ns("devnet/spy-relayer.yaml")

    k8s_resource(
        "spy-relayer",
        resource_deps = ["proto-gen", "guardian", "redis"],
        port_forwards = [
            port_forward(8083, name = "Prometheus [:8083]", host = webHost),
        ],
        labels = ["spy-relayer"],
        trigger_mode = trigger_mode,
    )

    k8s_yaml_with_ns("devnet/spy-wallet-monitor.yaml")

    k8s_resource(
        "spy-wallet-monitor",
        resource_deps = ["proto-gen", "guardian", "redis"],
        port_forwards = [
            port_forward(8084, name = "Prometheus [:8084]", host = webHost),
        ],
        labels = ["spy-relayer"],
        trigger_mode = trigger_mode,
    )

k8s_yaml_with_ns("devnet/eth-devnet.yaml")

k8s_resource(
    "eth-devnet",
    port_forwards = [
        port_forward(8545, name = "Ganache RPC [:8545]", host = webHost),
    ],
    resource_deps = ["const-gen"],
    labels = ["evm"],
    trigger_mode = trigger_mode,
)

k8s_resource(
    "eth-devnet2",
    port_forwards = [
        port_forward(8546, name = "Ganache RPC [:8546]", host = webHost),
    ],
    resource_deps = ["const-gen"],
    labels = ["evm"],
    trigger_mode = trigger_mode,
)

if bridge_ui:
    entrypoint = "npm run build && /app/node_modules/.bin/serve -s build -n"
    live_update = []
    if bridge_ui_hot:
        entrypoint = "npm start"
        live_update = [
            sync("./bridge_ui/public", "/app/public"),
            sync("./bridge_ui/src", "/app/src"),
        ]

    docker_build(
        ref = "bridge-ui",
        context = ".",
        only = ["./bridge_ui"],
        dockerfile = "bridge_ui/Dockerfile",
        entrypoint = entrypoint,
        live_update = live_update,
    )

    k8s_yaml_with_ns("devnet/bridge-ui.yaml")

    k8s_resource(
        "bridge-ui",
        resource_deps = [],
        port_forwards = [
            port_forward(3000, name = "Bridge UI [:3000]", host = webHost),
        ],
        labels = ["portal"],
        trigger_mode = trigger_mode,
    )

if ci_tests:
    local_resource(
        name = "solana-tests",
        deps = ["solana"],
        dir = "solana",
        cmd = "tilt docker build -- -f Dockerfile --target ci_tests --build-arg BRIDGE_ADDRESS=Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o .",
        env = {"DOCKER_BUILDKIT": "1"},
        labels = ["ci"],
        allow_parallel = True,
        trigger_mode = trigger_mode,
    )

    docker_build(
        ref = "tests-image",
        context = ".",
        dockerfile = "testing/Dockerfile.tests",
        only = [],
        live_update = [
            sync("./spydk/js/src", "/app/spydk/js/src"),
            sync("./sdk/js/src", "/app/sdk/js/src"),
            sync("./testing", "/app/testing"),
            sync("./bridge_ui/src", "/app/bridge_ui/src"),
        ],
    )

    k8s_yaml_with_ns("devnet/tests.yaml")

    k8s_resource(
        "ci-tests",
        resource_deps = ["proto-gen-web", "wasm-gen", "eth-devnet", "eth-devnet2", "terra-terrad", "terra-fcd", "terra2-terrad", "terra2-fcd", "solana-devnet", "spy", "guardian"],
        labels = ["ci"],
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
        labels = ["ci"],
        trigger_mode = trigger_mode,
    )

# bigtable

if explorer:
    k8s_yaml_with_ns("devnet/bigtable.yaml")

    k8s_resource(
        "bigtable-emulator",
        port_forwards = [port_forward(8086, name = "BigTable clients [:8086]")],
        labels = ["explorer"],
        trigger_mode = trigger_mode,
    )

    k8s_resource(
        "pubsub-emulator",
        port_forwards = [port_forward(8085, name = "PubSub listeners [:8085]")],
        labels = ["explorer"],
    )

    docker_build(
        ref = "cloud-functions",
        context = "./event_database",
        dockerfile = "./event_database/functions_server/Dockerfile",
        live_update = [
            sync("./event_database/cloud_functions", "/app/cloud_functions"),
        ],
    )
    k8s_resource(
        "cloud-functions",
        resource_deps = ["proto-gen", "bigtable-emulator", "pubsub-emulator"],
        port_forwards = [port_forward(8090, name = "Cloud Functions [:8090]", host = webHost)],
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
    resource_deps = ["const-gen"],
    labels = ["terra"],
    trigger_mode = trigger_mode,
)

k8s_resource(
    "terra-postgres",
    labels = ["terra"],
    trigger_mode = trigger_mode,
)

k8s_resource(
    "terra-fcd",
    resource_deps = ["terra-terrad", "terra-postgres"],
    port_forwards = [port_forward(3060, name = "Terra FCD [:3060]", host = webHost)],
    labels = ["terra"],
    trigger_mode = trigger_mode,
)

# terra 2 devnet

docker_build(
    ref = "terra2-image",
    context = "./terra2/devnet",
    dockerfile = "terra2/devnet/Dockerfile",
)

docker_build(
    ref = "terra2-contracts",
    context = "./terra2",
    dockerfile = "./terra2/Dockerfile",
)

k8s_yaml_with_ns("devnet/terra2-devnet.yaml")

k8s_resource(
    "terra2-terrad",
    port_forwards = [
        port_forward(26657, name = "Terra 2 RPC [:26657]", host = webHost),
        port_forward(1317, name = "Terra 2 LCD [:1317]", host = webHost),
    ],
    resource_deps = ["const-gen"],
    labels = ["terra2"],
    trigger_mode = trigger_mode,
)

k8s_resource(
    "terra2-postgres",
    labels = ["terra2"],
    trigger_mode = trigger_mode,
)

k8s_resource(
    "terra2-fcd",
    resource_deps = ["terra2-terrad", "terra2-postgres"],
    port_forwards = [port_forward(3060, name = "Terra 2 FCD [:3060]", host = webHost)],
    labels = ["terra2"],
    trigger_mode = trigger_mode,
)

if algorand:
    k8s_yaml_with_ns("devnet/algorand-devnet.yaml")
  
    docker_build(
        ref = "algorand-algod",
        context = "algorand/sandbox-algorand",
        dockerfile = "algorand/sandbox-algorand/images/algod/Dockerfile"
    )

    docker_build(
        ref = "algorand-indexer",
        context = "algorand/sandbox-algorand",
        dockerfile = "algorand/sandbox-algorand/images/indexer/Dockerfile"
    )

    docker_build(
        ref = "algorand-contracts",
        context = "algorand",
        dockerfile = "algorand/Dockerfile",
        ignore = ["algorand/test/*.*"]
    )

    k8s_resource(
        "algorand",
        port_forwards = [
            port_forward(4001, name = "Algod [:4001]", host = webHost),
            port_forward(4002, name = "KMD [:4002]", host = webHost),
            port_forward(8980, name = "Indexer [:8980]", host = webHost),
        ],
        resource_deps = ["const-gen"],
        labels = ["algorand"],
        trigger_mode = trigger_mode,
    )
    
