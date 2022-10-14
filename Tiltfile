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
config.define_bool("near", False, "Enable Near component")
config.define_bool("aptos", False, "Enable Aptos component")
config.define_bool("algorand", False, "Enable Algorand component")
config.define_bool("evm2", False, "Enable second Eth component")
config.define_bool("solana", False, "Enable Solana component")
config.define_bool("terra_classic", False, "Enable Terra Classic component")
config.define_bool("terra2", False, "Enable Terra 2 component")
config.define_bool("explorer", False, "Enable explorer component")
config.define_bool("spy_relayer", False, "Enable spy relayer")
config.define_bool("e2e", False, "Enable E2E testing stack")
config.define_bool("ci_tests", False, "Enable tests runner component")
config.define_bool("guardiand_debug", False, "Enable dlv endpoint for guardiand")
config.define_bool("node_metrics", False, "Enable Prometheus & Grafana for Guardian metrics")
config.define_bool("guardiand_governor", False, "Enable chain governor in guardiand")
config.define_bool("secondWormchain", False, "Enable a second wormchain node with different validator keys")

cfg = config.parse()
num_guardians = int(cfg.get("num", "1"))
namespace = cfg.get("namespace", "wormhole")
gcpProject = cfg.get("gcpProject", "local-dev")
bigTableKeyPath = cfg.get("bigTableKeyPath", "./event_database/devnet_key.json")
webHost = cfg.get("webHost", "localhost")
algorand = cfg.get("algorand", True)
near = cfg.get("near", True)
aptos = cfg.get("aptos", True)
evm2 = cfg.get("evm2", True)
solana = cfg.get("solana", True)
terra_classic = cfg.get("terra_classic", True)
terra2 = cfg.get("terra2", True)
ci = cfg.get("ci", False)
explorer = cfg.get("explorer", ci)
spy_relayer = cfg.get("spy_relayer", ci)
e2e = cfg.get("e2e", ci)
ci_tests = cfg.get("ci_tests", ci)
guardiand_debug = cfg.get("guardiand_debug", False)
node_metrics = cfg.get("node_metrics", False)
guardiand_governor = cfg.get("guardiand_governor", False)
secondWormchain = cfg.get("secondWormchain", False)

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
    name = "const-gen",
    deps = ["scripts", "clients", "ethereum/.env.test"],
    cmd = 'tilt docker build -- --target const-export -f Dockerfile.const -o type=local,dest=. --build-arg num_guardians=%s .' % (num_guardians),
    env = {"DOCKER_BUILDKIT": "1"},
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
    context = ".",
    dockerfile = "Dockerfile.node",
    target = "build",
    ignore=["./sdk/js"]
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

            if aptos:
                container["command"] += [
                    "--aptosRPC",
                    "http://aptos:8080",
                    "--aptosAccount",
                    "de0036a9600559e295d5f6802ef6f3f802f510366e0c23912b0655d972166017",
                    "--aptosHandle",
                    "0xde0036a9600559e295d5f6802ef6f3f802f510366e0c23912b0655d972166017::state::WormholeMessageHandle",
                ]

            if evm2:
                container["command"] += [
                    "--bscRPC",
                    "ws://eth-devnet2:8545",
                ]
            else:
                container["command"] += [
                    "--bscRPC",
                    "ws://eth-devnet:8545",
                ]

            if solana:
                container["command"] += [
                    "--solanaRPC",
                    "http://solana-devnet:8899",
                ]

            if terra_classic:
                container["command"] += [
                    "--terraWS",
                    "ws://terra-terrad:26657/websocket",
                    "--terraLCD",
                    "http://terra-terrad:1317",
                    "--terraContract",
                    "terra18vd8fpwxzck93qlwghaj6arh4p7c5n896xzem5",
                ]

            if terra2:
                container["command"] += [
                    "--terra2WS",
                    "ws://terra2-terrad:26657/websocket",
                    "--terra2LCD",
                    "http://terra2-terrad:1317",
                    "--terra2Contract",
                    "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au",
                ]

            if algorand:
                container["command"] += [
                    "--algorandAppID",
                    "4",
                    "--algorandIndexerRPC",
                    "http://algorand:8980",
                    "--algorandIndexerToken",
                    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                    "--algorandAlgodRPC",
                    "http://algorand:4001",
                    "--algorandAlgodToken",
                    "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                ]

            if guardiand_governor:
                container["command"] += [
                    "--chainGovernorEnabled"
                ]

            if near:
                container["command"] += [
                    "--nearRPC",
                    "http://near:3030",
                    "--nearContract",
                    "wormhole.test.near"
                ]

    return encode_yaml_stream(node_yaml)

k8s_yaml_with_ns(build_node_yaml())

guardian_resource_deps = ["proto-gen", "eth-devnet"]
if evm2:
    guardian_resource_deps = guardian_resource_deps + ["eth-devnet2"]
if solana:
    guardian_resource_deps = guardian_resource_deps + ["solana-devnet"]
if near:
    guardian_resource_deps = guardian_resource_deps + ["near"]
if terra_classic:
    guardian_resource_deps = guardian_resource_deps + ["terra-terrad"]
if terra2:
    guardian_resource_deps = guardian_resource_deps + ["terra2-terrad"]
if algorand:
    guardian_resource_deps = guardian_resource_deps + ["algorand"]
if aptos:
    guardian_resource_deps = guardian_resource_deps + ["aptos"]

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


# grafana + prometheus for node metrics
if node_metrics:

    dashboard = read_json("dashboards/Wormhole.json")

    dashboard_yaml =  {
        "apiVersion": "v1",
        "kind": "ConfigMap",
        "metadata": {
            "name": "grafana-dashboards-json"
        },
        "data": {
            "wormhole.json": encode_json(dashboard)
        }
    }
    k8s_yaml_with_ns(encode_yaml(dashboard_yaml))

    k8s_yaml_with_ns("devnet/node-metrics.yaml")

    k8s_resource(
        "prometheus-server",
        resource_deps = ["guardian"],
        port_forwards = [
            port_forward(9099, name = "Prometheus [:9099]", host = webHost),
        ],
        labels = ["guardian"],
        trigger_mode = trigger_mode,
    )

    k8s_resource(
        "grafana",
        resource_deps = ["prometheus-server"],
        port_forwards = [
            port_forward(3033, name = "Grafana UI [:3033]", host = webHost),
        ],
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

    docker_build(
        ref = "bridge-client",
        context = ".",
        only = ["./proto", "./solana", "./clients"],
        dockerfile = "Dockerfile.client",
        # Ignore target folders from local (non-container) development.
        ignore = ["./solana/*/target"],
    )

    # solana smart contract

    docker_build(
        ref = "solana-contract",
        context = "solana",
        dockerfile = "solana/Dockerfile",
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
        resource_deps = ["proto-gen", "guardian", "redis", "spy"],
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

if evm2:
    k8s_yaml_with_ns("devnet/eth-devnet2.yaml")

    k8s_resource(
        "eth-devnet2",
        port_forwards = [
            port_forward(8546, name = "Ganache RPC [:8546]", host = webHost),
        ],
        resource_deps = ["const-gen"],
        labels = ["evm"],
        trigger_mode = trigger_mode,
    )


if ci_tests:
    docker_build(
        ref = "sdk-test-image",
        context = ".",
        dockerfile = "testing/Dockerfile.sdk.test",
        only = [],
        live_update = [
            sync("./sdk/js/src", "/app/sdk/js/src"),
            sync("./testing", "/app/testing"),
        ],
    )
    docker_build(
        ref = "spydk-test-image",
        context = ".",
        dockerfile = "testing/Dockerfile.spydk.test",
        only = [],
        live_update = [
            sync("./spydk/js/src", "/app/spydk/js/src"),
            sync("./testing", "/app/testing"),
        ],
    )

    k8s_yaml_with_ns("devnet/tests.yaml")

    # separate resources to parallelize docker builds
    k8s_resource(
        "sdk-ci-tests",
        labels = ["ci"],
        trigger_mode = trigger_mode,
        resource_deps = ["guardian"],
    )
    k8s_resource(
        "spydk-ci-tests",
        labels = ["ci"],
        trigger_mode = trigger_mode,
        resource_deps = ["guardian", "spy"],
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

if terra_classic:
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

if terra2:
    docker_build(
        ref = "terra2-image",
        context = "./cosmwasm/devnet",
        dockerfile = "cosmwasm/devnet/Dockerfile",
    )

    docker_build(
        ref = "terra2-contracts",
        context = "./cosmwasm",
        dockerfile = "./cosmwasm/Dockerfile",
    )

    k8s_yaml_with_ns("devnet/terra2-devnet.yaml")

    k8s_resource(
        "terra2-terrad",
        port_forwards = [
            port_forward(26658, container_port = 26657, name = "Terra 2 RPC [:26658]", host = webHost),
            port_forward(1318, container_port = 1317, name = "Terra 2 LCD [:1318]", host = webHost),
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
        port_forwards = [port_forward(3061, container_port = 3060, name = "Terra 2 FCD [:3061]", host = webHost)],
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


if near:
    k8s_yaml_with_ns("devnet/near-devnet.yaml")

    docker_build(
        ref = "near-node",
        context = "near",
        dockerfile = "near/Dockerfile",
        only = ["Dockerfile", "node_builder.sh", "start_node.sh", "README.md", "cert.pem"],
    )

    docker_build(
        ref = "near-deploy",
        context = "near",
        dockerfile = "near/Dockerfile.deploy",
        ignore = ["./test"]
    )

    k8s_resource(
        "near",
        port_forwards = [
            port_forward(3030, name = "Node [:3030]", host = webHost),
            port_forward(3031, name = "webserver [:3031]", host = webHost),
        ],
        resource_deps = ["const-gen"],
        labels = ["near"],
        trigger_mode = trigger_mode,
    )

docker_build(
    ref = "wormhole-chaind-image",
    context = ".",
    dockerfile = "./Dockerfile.wormchain",
    only = [],
    ignore = ["./wormhole_chain/testing", "./wormhole_chain/ts-sdk", "./wormhole_chain/design", "./wormhole_chain/vue", "./wormhole_chain/build/wormhole-chaind"],
)

k8s_yaml_with_ns("wormhole_chain/validators/kubernetes/wormchain-guardian-devnet.yaml")

k8s_resource(
    "guardian-validator",
    port_forwards = [
        port_forward(1319, container_port = 1317, name = "REST [:1319]", host = webHost),
        port_forward(26659, container_port = 26657, name = "TENDERMINT [:26659]", host = webHost)
    ],
    resource_deps = [],
    labels = ["wormchain"],
    trigger_mode = trigger_mode,
)

if secondWormchain:
    k8s_yaml_with_ns("wormhole_chain/validators/kubernetes/wormchain-validator2-devnet.yaml")

    k8s_resource(
        "second-validator",
        port_forwards = [
            port_forward(1320, container_port = 1317, name = "REST [:1320]", host = webHost),
            port_forward(26660, container_port = 26657, name = "TENDERMINT [:26660]", host = webHost)
        ],
        resource_deps = [],
        labels = ["wormchain"],
        trigger_mode = trigger_mode,
    )

if aptos:
    k8s_yaml_with_ns("devnet/aptos-localnet.yaml")

    docker_build(
        ref = "aptos-node",
        context = "aptos",
        dockerfile = "aptos/Dockerfile",
        target = "aptos",
    )

    k8s_resource(
        "aptos",
        port_forwards = [
            port_forward(8080, name = "RPC [:8080]", host = webHost),
            port_forward(6181, name = "FullNode [:6181]", host = webHost),
            port_forward(8081, name = "Faucet [:8081]", host = webHost),
        ],
        resource_deps = ["const-gen"],
        labels = ["aptos"],
        trigger_mode = trigger_mode,
    )
