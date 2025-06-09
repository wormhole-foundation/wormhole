# This Tiltfile contains the deployment and build config for the Wormhole devnet.
#
#  We use Buildkit cache mounts and careful layering to avoid unnecessary rebuilds - almost
#  all source code changes result in small, incremental rebuilds. Dockerfiles are written such
#  that, for example, changing the contract source code won't cause Solana itself to be rebuilt.
#

load("ext://namespace", "namespace_create", "namespace_inject")
load("ext://secret", "secret_yaml_generic")

# set the replica value of a StatefulSet
def set_replicas_in_statefulset(config_yaml, statefulset_name,  num_replicas):
    for obj in config_yaml:
        if obj["kind"] == "StatefulSet" and obj["metadata"]["name"] == statefulset_name:
            obj["spec"]["replicas"] = num_replicas
    return config_yaml

# set the env value of all containers in all jobs
def set_env_in_jobs(config_yaml, name, value):
    for obj in config_yaml:
        if obj["kind"] == "Job":
            for container in obj["spec"]["template"]["spec"]["containers"]:
                if not "env" in container:
                    container["env"] = []
                container["env"].append({"name": name, "value": value})
    return config_yaml

allow_k8s_contexts("ci")

# Disable telemetry by default
analytics_settings(False)

# Moar updates (default is 3)
update_settings(max_parallel_updates = 10)

# Runtime configuration
config.define_bool("ci", False, "We are running in CI")
config.define_bool("manual", False, "Set TRIGGER_MODE_MANUAL by default")

config.define_string("num", False, "Number of guardian nodes to run")
config.define_string("maxWorkers", False, "Maximum number of workers for sdk-ci-tests. See https://jestjs.io/docs/cli#--maxworkersnumstring")

# You do not usually need to set this argument - this argument is for debugging only. If you do use a different
# namespace, note that the "wormhole" namespace is hardcoded in tests and don't forget specifying the argument
# when running "tilt down".
#
config.define_string("namespace", False, "Kubernetes namespace to use")

# When running Tilt on a server, this can be used to set the public hostname Tilt runs on
# for service links in the UI to work.
config.define_string("webHost", False, "Public hostname for port forwards")

# When running Tilt on a server, this can be used to set the public hostname Tilt runs on
# for service links in the UI to work.
config.define_string("guardiand_loglevel", False, "Log level for guardiand (debug, info, warn, error, dpanic, panic, fatal)")

# Components
config.define_bool("near", False, "Enable Near component")
config.define_bool("sui", False, "Enable Sui component")
config.define_bool("btc", False, "Enable BTC component")
config.define_bool("aptos", False, "Enable Aptos component")
config.define_bool("aztec", False, "Enable Aztec component")
config.define_bool("algorand", False, "Enable Algorand component")
config.define_bool("evm2", False, "Enable second Eth component")
config.define_bool("solana", False, "Enable Solana component")
config.define_bool("solana_watcher", False, "Enable Solana watcher on guardian")
config.define_bool("pythnet", False, "Enable PythNet component")
config.define_bool("terra_classic", False, "Enable Terra Classic component")
config.define_bool("terra2", False, "Enable Terra 2 component")
config.define_bool("ci_tests", False, "Enable tests runner component")
config.define_bool("guardiand_debug", False, "Enable dlv endpoint for guardiand")
config.define_bool("node_metrics", False, "Enable Prometheus & Grafana for Guardian metrics")
config.define_bool("guardiand_governor", False, "Enable chain governor in guardiand")
config.define_bool("wormchain", False, "Enable a wormchain node")
config.define_bool("ibc_relayer", False, "Enable IBC relayer between cosmos chains")
config.define_bool("redis", False, "Enable a redis instance")
config.define_bool("generic_relayer", False, "Enable the generic relayer off-chain component")
config.define_bool("query_server", False, "Enable cross-chain query server")

cfg = config.parse()
num_guardians = int(cfg.get("num", "1"))
max_workers = cfg.get("maxWorkers", "50%")
namespace = cfg.get("namespace", "wormhole")
webHost = cfg.get("webHost", "localhost")
ci = cfg.get("ci", False)
algorand = cfg.get("algorand", ci)
near = cfg.get("near", ci)
aptos = cfg.get("aptos", ci)
aztec = cfg.get("aztec", ci)
sui = cfg.get("sui", ci)
evm2 = cfg.get("evm2", ci)
solana = cfg.get("solana", ci)
pythnet = cfg.get("pythnet", False)
solana_watcher = cfg.get("solana_watcher", solana or pythnet)
terra_classic = cfg.get("terra_classic", ci)
terra2 = cfg.get("terra2", ci)
wormchain = cfg.get("wormchain", ci)
ci_tests = cfg.get("ci_tests", ci)
guardiand_debug = cfg.get("guardiand_debug", False)
node_metrics = cfg.get("node_metrics", False)
guardiand_governor = cfg.get("guardiand_governor", False)
ibc_relayer = cfg.get("ibc_relayer", ci)
btc = cfg.get("btc", False)
redis = cfg.get('redis', ci)
generic_relayer = cfg.get("generic_relayer", ci)
query_server = cfg.get("query_server", ci)

if ci:
    guardiand_loglevel = cfg.get("guardiand_loglevel", "warn")
else:
    guardiand_loglevel = cfg.get("guardiand_loglevel", "info")


if cfg.get("manual", False):
    trigger_mode = TRIGGER_MODE_MANUAL
else:
    trigger_mode = TRIGGER_MODE_AUTO

# namespace

if not ci:
    namespace_create(namespace)

def k8s_yaml_with_ns(objects):
    return k8s_yaml(namespace_inject(objects, namespace))

docker_build(
    ref = "cli-gen",
    context = ".",
    dockerfile = "Dockerfile.cli",
)

docker_build(
    ref = "const-gen",
    context = ".",
    dockerfile = "Dockerfile.const",
    build_args={"num_guardians": '%s' % (num_guardians)},
)

# node

docker_build(
    ref = "guardiand-image",
    context = ".",
    dockerfile = "node/Dockerfile",
    target = "build",
    ignore=["./sdk/js", "./relayer"]
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

def generate_bootstrap_peers(num_guardians, port_num):
    # Improve the chances of the guardians discovering each other in tilt by making them all bootstrap peers.
    # The devnet guardian uses deterministic P2P peer IDs based on the guardian index. The peer IDs here
    # were generated using `DeterministicP2PPrivKeyByIndex` in `node/pkg/devnet/deterministic_p2p_key.go`.
    peer_ids = [
        "12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw",
        "12D3KooWHHzSeKaY8xuZVzkLbKFfvNgPPeKhFBGrMbNzbm5akpqu",
        "12D3KooWKRyzVWW6ChFjQjK4miCty85Niy49tpPV95XdKu1BcvMA",
        "12D3KooWB1b3qZxWJanuhtseF3DmPggHCtG36KZ9ixkqHtdKH9fh",
        "12D3KooWE4qDcRrueTuRYWUdQZgcy7APZqBngVeXRt4Y6ytHizKV",
        "12D3KooWPgam4TzSVCRa4AbhxQnM9abCYR4E9hV57SN7eAjEYn1j",
        "12D3KooWM4yJB31d4hF2F9Vdwuj9WFo1qonoySyw4bVAQ9a9d21o",
        "12D3KooWCv935r3ropYhUe5yMCp9QiUoc9A6cZpYQ5x84DqEPbwb",
        "12D3KooWQfG74brcJhzpNwjPCZmcbBv8f6wxKgLSYmEDXXdPXQpH",
        "12D3KooWNEWRB7PnuZs164xaA9QWM3iZHekHyEQo5qGP5KCHHuSN",
        "12D3KooWB224kvi7vN34xJfsfW7bnv6eodxTkgo9VFA6UiaGMgRD",
        "12D3KooWCR2EoapJjoQVR4E3NLjWn818gG3XizQ92Yx6C424HL2g",
        "12D3KooWNc5rNmCJ9yvXviXaENnp7vqDQjomZwia4aA7Q3hSYkiW",
        "12D3KooWBremnqYWBDK6ctvCuhCqJAps5ZAPADu53gXhQHexrvtP",
        "12D3KooWFqdBYPrtwErMosomvD4uRtVhXQdqqZZHC3NCBZYVxr4t",
        "12D3KooW9yvKfP5HgVaLnNaxWywo3pLAEypk7wjUcpgKwLznk5gQ",
        "12D3KooWRuYVGEsecrJJhZsSoKf1UNdBVYKFCmFLNj9ucZiSQCYj",
        "12D3KooWGEcD5sW5osB6LajkHGqiGc3W8eKfYwnJVVqfujkpLWX2",
        "12D3KooWQYz2inBsgiBoqNtmEn1qeRBr9B8cdishFuBgiARcfMcY"
    ]
    bootstrap = ""
    for idx in range(num_guardians):
        if bootstrap != "":
            bootstrap += ","
        bootstrap += "/dns4/guardian-{idx}.guardian/udp/{port}/quic/p2p/{peer}".format(idx = idx, port = port_num, peer = peer_ids[idx])
    return bootstrap

bootstrapPeers = generate_bootstrap_peers(num_guardians, 8999)
ccqBootstrapPeers = generate_bootstrap_peers(num_guardians, 8996)

def build_node_yaml():
    node_yaml = read_yaml_stream("devnet/node.yaml")

    node_yaml_with_replicas = set_replicas_in_statefulset(node_yaml, "guardian", num_guardians)

    for obj in node_yaml_with_replicas:
        if obj["kind"] == "StatefulSet" and obj["metadata"]["name"] == "guardian":
            container = obj["spec"]["template"]["spec"]["containers"][0]
            if container["name"] != "guardiand":
                fail("container 0 is not guardiand")

            container["command"] += ["--logLevel="+guardiand_loglevel]

            if guardiand_debug:
                container["command"] = command_with_dlv(container["command"])
                print(container["command"])

            if num_guardians > 1:
                container["command"] += [
                    "--bootstrap",
                    bootstrapPeers,
                    "--ccqP2pBootstrap",
                    ccqBootstrapPeers,
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

            if aztec:
                container["command"] += [
                    "--aztecRPC",
                    "http://aztec-sandbox:8090",
                    "--aztecContract",
                    "0x0d6fe810321185c97a0e94200f998bcae787aaddf953a03b14ec5da3b6838bad",
                ]

            if sui:
                container["command"] += [
                    "--suiRPC",
                    "http://sui:9000",
                    "--suiMoveEventType",
                    "0x320a40bff834b5ffa12d7f5cc2220dd733dd9e8e91c425800203d06fb2b1fee8::publish_message::WormholeMessage",
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

            if solana_watcher:
                container["command"] += [
                    "--solanaRPC",
                    "http://solana-devnet:8899",
                    "--solanaContract",
                    "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o",
                    "--solanaShimContract",
                    "EtZMZM22ViKMo4r5y4Anovs3wKQ2owUmDpjygnMMcdEX",
                ]

            if pythnet:
                container["command"] += [
                    "--pythnetRPC",
#                    "http://solana-devnet:8899",
                     "http://pythnet.rpcpool.com",
                    "--pythnetWS",
#                   "ws://solana-devnet:8900",
                    "wss://pythnet.rpcpool.com",
                    "--pythnetContract",
                    "H3fxXJ86ADW2PNuDDmZJg6mzTtPxkYCpNuQUTgmJ7AjU",
                ]

            if terra_classic:
                container["command"] += [
                    "--terraWS",
                    "ws://terra-terrad:26657/websocket",
                    "--terraLCD",
                    "http://terra-terrad:1317",
                    "--terraContract",
                    "terra14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9ssrc8au",
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
                    "1004",
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

            if wormchain:
                container["command"] += [
                    "--wormchainURL",
                    "wormchain:9090",

                     "--accountantWS",
                    "http://wormchain:26657",

                    "--accountantContract",
                    "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465",
                    "--accountantKeyPath",
                    "/tmp/mounted-keys/wormchain/accountantKey",
                    "--accountantKeyPassPhrase",
                    "test0000",
                    "--accountantCheckEnabled",
                    "true",

                    "--accountantNttContract",
                    "wormhole17p9rzwnnfxcjp32un9ug7yhhzgtkhvl9jfksztgw5uh69wac2pgshdnj3k",
                    "--accountantNttKeyPath",
                    "/tmp/mounted-keys/wormchain/accountantNttKey",
                    "--accountantNttKeyPassPhrase",
                    "test0000",

                    "--ibcContract",
                    "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj",
                    "--ibcWS",
                    "ws://wormchain:26657/websocket",
                    "--ibcLCD",
                    "http://wormchain:1317",

                    "--gatewayRelayerContract",
                    "wormhole1wn625s4jcmvk0szpl85rj5azkfc6suyvf75q6vrddscjdphtve8sca0pvl",
                    "--gatewayRelayerKeyPath",
                    "/tmp/mounted-keys/wormchain/gwrelayerKey",
                    "--gatewayRelayerKeyPassPhrase",
                    "test0000",

                    "--gatewayContract",
                    "wormhole1ghd753shjuwexxywmgs4xz7x2q732vcnkm6h2pyv9s6ah3hylvrqtm7t3h",
                    "--gatewayWS",
                    "ws://wormchain:26657/websocket",
                    "--gatewayLCD",
                    "http://wormchain:1317"
                ]

    return encode_yaml_stream(node_yaml_with_replicas)

k8s_yaml_with_ns(build_node_yaml())

guardian_resource_deps = ["eth-devnet"]
if evm2:
    guardian_resource_deps = guardian_resource_deps + ["eth-devnet2"]
if solana_watcher:
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
if aztec:
    guardian_resource_deps = guardian_resource_deps + ["aztec-sandbox"]
if wormchain:
    guardian_resource_deps = guardian_resource_deps + ["wormchain", "wormchain-deploy"]
if sui:
    guardian_resource_deps = guardian_resource_deps + ["sui"]

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
    resource_deps = ["guardian"],
    port_forwards = [
        port_forward(6061, container_port = 6060, name = "Debug/Status Server [:6061]", host = webHost),
        port_forward(7072, name = "Spy gRPC [:7072]", host = webHost),
    ],
    labels = ["guardian"],
    trigger_mode = trigger_mode,
)

if solana or pythnet:
    # solana client cli (used for devnet setup)

    docker_build(
        ref = "bridge-client",
        context = ".",
        only = ["./proto", "./solana", "./clients"],
        dockerfile = "solana/Dockerfile.client",
        # Ignore target folders from local (non-container) development.
        ignore = ["./solana/*/target", "./solana/tests"],
    )

    # solana smart contract

    docker_build(
        ref = "solana-contract",
        context = "solana",
        dockerfile = "solana/Dockerfile",
        target = "builder",
        ignore = ["./solana/*/target", "./solana/tests"],
        build_args = {"BRIDGE_ADDRESS": "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"}
    )

    # solana local devnet

    docker_build(
        ref = "solana-test-validator",
        context = "solana",
        dockerfile = "solana/Dockerfile.test-validator",
    )

    k8s_yaml_with_ns("devnet/solana-devnet.yaml")

    k8s_resource(
        "solana-devnet",
        port_forwards = [
            port_forward(8899, name = "Solana RPC [:8899]", host = webHost),
            port_forward(8900, name = "Solana WS [:8900]", host = webHost),
        ],
        labels = ["solana"],
        trigger_mode = trigger_mode,
    )

# eth devnet

docker_build(
    ref = "eth-node",
    context = ".",
    only = ["./ethereum", "./relayer/ethereum"],
    dockerfile = "./ethereum/Dockerfile",

    # ignore local node_modules (in case they're present)
    ignore = ["./ethereum/node_modules","./relayer/ethereum/node_modules"],
    build_args = {"num_guardians": str(num_guardians), "dev": str(not ci)},

    # sync external scripts for incremental development
    # (everything else needs to be restarted from scratch for determinism)
    #
    # This relies on --update-mode=exec to work properly with a non-root user.
    # https://github.com/tilt-dev/tilt/issues/3708
    live_update = [
        sync("./ethereum/src", "/home/node/app/src"),
    ],
)

if redis or generic_relayer:
    docker_build(
        ref = "redis",
        context = ".",
        only = ["./third_party"],
        dockerfile = "third_party/redis/Dockerfile",
    )

if redis:
    k8s_resource(
        "redis",
        port_forwards = [
            port_forward(6379, name = "Redis Default [:6379]", host = webHost),
        ],
        labels = ["redis"],
        trigger_mode = trigger_mode,
    )

    k8s_yaml_with_ns("devnet/redis.yaml")

if generic_relayer:
    k8s_resource(
        "redis-relayer",
        port_forwards = [
            port_forward(6378, name = "Generic Relayer Redis [:6378]", host = webHost),
        ],
        labels = ["redis-relayer"],
        trigger_mode = trigger_mode,
    )

    k8s_yaml_with_ns("devnet/redis-relayer.yaml")



if generic_relayer:
    k8s_resource(
        "relayer-engine",
        resource_deps = ["guardian", "redis-relayer", "spy"],
        port_forwards = [
            port_forward(3003, container_port=3000, name = "Bullmq UI [:3003]", host = webHost),
        ],
        labels = ["relayer-engine"],
        trigger_mode = trigger_mode,
    )
    docker_build(
        ref = "relayer-engine",
        context = ".",
        only = ["./relayer/generic_relayer", "./relayer/ethereum/ts-scripts/relayer/config"],
        dockerfile = "relayer/generic_relayer/relayer-engine-v2/Dockerfile",
        build_args = {"dev": str(not ci)}
    )
    k8s_yaml_with_ns("devnet/relayer-engine.yaml")

k8s_yaml_with_ns("devnet/eth-devnet.yaml")

k8s_resource(
    "eth-devnet",
    port_forwards = [
        port_forward(8545, name = "Anvil RPC [:8545]", host = webHost),
    ],
    labels = ["evm"],
    trigger_mode = trigger_mode,
)

if evm2:
    k8s_yaml_with_ns("devnet/eth-devnet2.yaml")

    k8s_resource(
        "eth-devnet2",
        port_forwards = [
            port_forward(8546, 8545, name = "Anvil RPC [:8546]", host = webHost),
        ],
        labels = ["evm"],
        trigger_mode = trigger_mode,
    )


# Note that ci_tests requires other resources in order to build properly:
# - eth-devnet  -- required by: accountant_tests, ntt_accountant_tests, tx-verifier
# - eth-devnet2 -- required by: accountant_tests, ntt_accountant_tests
# - wormchain   -- required by: accountant_tests, ntt_accountant_tests
# - solana      -- required by: spydk-ci-tests
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
    docker_build(
        ref = "query-sdk-test-image",
        context = ".",
        dockerfile = "testing/Dockerfile.querysdk.test",
        only = [],
        live_update = [
            sync("./sdk/js/src", "/app/sdk/js-query/src"),
            sync("./testing", "/app/testing"),
        ],
    )
    docker_build(
        ref = "tx-verifier-evm",
        context = "./devnet/tx-verifier/",
        dockerfile = "./devnet/tx-verifier/Dockerfile.tx-verifier-evm"
    )
    k8s_yaml_with_ns("devnet/tx-verifier-evm.yaml")

    k8s_yaml_with_ns(
        encode_yaml_stream(
            set_env_in_jobs(
                set_env_in_jobs(
                    set_env_in_jobs(read_yaml_stream("devnet/tests.yaml"), "NUM_GUARDIANS", str(num_guardians)),
                    "BOOTSTRAP_PEERS", str(ccqBootstrapPeers)),
                    "MAX_WORKERS", max_workers))
    )

    # separate resources to parallelize docker builds
    k8s_resource(
        "sdk-ci-tests",
        labels = ["ci"],
        trigger_mode = trigger_mode,
        resource_deps = [], # testing/sdk.sh handles waiting for spy, not having deps gets the build earlier
    )
    k8s_resource(
        "spydk-ci-tests",
        labels = ["ci"],
        trigger_mode = trigger_mode,
        resource_deps = [], # testing/spydk.sh handles waiting for spy, not having deps gets the build earlier
    )
    k8s_resource(
        "accountant-ci-tests",
        labels = ["ci"],
        trigger_mode = trigger_mode,
        resource_deps = [], # uses devnet-consts.json, but wormchain/contracts/tools/test_accountant.sh handles waiting for guardian, not having deps gets the build earlier
    )
    k8s_resource(
        "ntt-accountant-ci-tests",
        labels = ["ci"],
        trigger_mode = trigger_mode,
        resource_deps = [], # uses devnet-consts.json, but wormchain/contracts/tools/test_ntt_accountant.sh handles waiting for guardian, not having deps gets the build earlier
    )
    k8s_resource(
        "query-sdk-ci-tests",
        labels = ["ci"],
        trigger_mode = trigger_mode,
        resource_deps = [], # testing/querysdk.sh handles waiting for query-server, not having deps gets the build earlier
    )
    # launches Transfer Verifier binary and sets up monitoring script
    k8s_resource(
        "tx-verifier-evm",
        labels = ["tx-verifier-evm"],
        trigger_mode = trigger_mode,
        resource_deps = ["eth-devnet"],
    )

if terra_classic:
    docker_build(
        ref = "terra-image",
        context = "./terra/devnet",
        dockerfile = "terra/devnet/Dockerfile",
        platform = "linux/amd64",
    )

    docker_build(
        ref = "terra-contracts",
        context = "./terra",
        dockerfile = "./terra/Dockerfile",
        platform = "linux/amd64",
    )

    k8s_yaml_with_ns("devnet/terra-devnet.yaml")

    k8s_resource(
        "terra-terrad",
        port_forwards = [
            port_forward(26657, name = "Terra RPC [:26657]", host = webHost),
            port_forward(1317, name = "Terra LCD [:1317]", host = webHost),
        ],
        labels = ["terra"],
        trigger_mode = trigger_mode,
    )

if terra2 or wormchain:
    docker_build(
        ref = "cosmwasm_artifacts",
        context = ".",
        dockerfile = "./cosmwasm/Dockerfile",
        target = "artifacts",
        platform = "linux/amd64",
    )

if terra2:
    docker_build(
        ref = "terra2-image",
        context = "./cosmwasm/deployment/terra2/devnet",
        dockerfile = "./cosmwasm/deployment/terra2/devnet/Dockerfile",
        platform = "linux/amd64",
    )

    docker_build(
        ref = "terra2-deploy",
        context = "./cosmwasm/deployment/terra2",
        dockerfile = "./cosmwasm/Dockerfile.deploy",
    )

    k8s_yaml_with_ns("devnet/terra2-devnet.yaml")

    k8s_resource(
        "terra2-terrad",
        port_forwards = [
            port_forward(26658, container_port = 26657, name = "Terra 2 RPC [:26658]", host = webHost),
            port_forward(1318, container_port = 1317, name = "Terra 2 LCD [:1318]", host = webHost),
        ],
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
        labels = ["algorand"],
        trigger_mode = trigger_mode,
    )

if sui:
    k8s_yaml_with_ns("devnet/sui-devnet.yaml")

    docker_build(
        ref = "sui-node",
        target = "sui",
        context = ".",
        dockerfile = "sui/Dockerfile",
        ignore = ["./sui/sui.log*", "sui/sui.log*", "sui.log.*"],
        only = ["./sui"],
    )

    k8s_resource(
        "sui",
        port_forwards = [
            port_forward(9000, 9000, name = "RPC [:9000]", host = webHost),
            port_forward(9184, name = "Prometheus [:9184]", host = webHost),
        ],
        labels = ["sui"],
        trigger_mode = trigger_mode,
    )

if near:
    k8s_yaml_with_ns("devnet/near-devnet.yaml")

    docker_build(
        ref = "near-node",
        context = "near",
        dockerfile = "near/Dockerfile",
        only = ["Dockerfile", "node_builder.sh", "start_node.sh", "README.md"],
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
        labels = ["near"],
        trigger_mode = trigger_mode,
    )

if wormchain:
    docker_build(
        ref = "wormchaind-image",
        context = ".",
        dockerfile = "./wormchain/Dockerfile",
        platform = "linux/amd64",
        build_args = {"num_guardians": str(num_guardians)},
        only = [],
        ignore = ["./wormchain/testing", "./wormchain/ts-sdk", "./wormchain/design", "./wormchain/vue", "./wormchain/build/wormchaind"],
    )

    # docker_build(
    #     ref = "vue-export",
    #     context = ".",
    #     dockerfile = "./wormchain/Dockerfile.proto",
    #     target = "vue-export",
    # )

    docker_build(
        ref = "wormchain-deploy",
        context = "./wormchain",
        dockerfile = "./wormchain/Dockerfile.deploy",
    )

    def build_wormchain_yaml(yaml_path, num_instances):
        wormchain_yaml = read_yaml_stream(yaml_path)

        # set the number of replicas in the StatefulSet to be num_guardians
        wormchain_set = set_replicas_in_statefulset(wormchain_yaml, "wormchain", num_instances)

        # add a Service for each wormchain instance
        services = []
        for obj in wormchain_set:
            if obj["kind"] == "Service" and obj["metadata"]["name"] == "wormchain-0":

                # make a Service for each replica so we can resolve it by name from other pods.
                # copy wormchain-0's Service then set the name and selector for the instance.
                for instance_num in list(range(1, num_instances)):
                    instance_name = 'wormchain-%s' % (instance_num)

                    # Copy the Service's properties to a new dict, by value, three levels deep.
                    # tl;dr - if the value is a dict, use a comprehension to copy it immutably.
                    service = { k: ({ k2: ({ k3:v3
                        for (k3,v3) in v2.items()} if type(v2) == "dict" else v2)
                        for (k2,v2) in v.items()} if type(v) == "dict" else v)
                        for (k,v) in obj.items()}

                    # add the name we want to be able to resolve via k8s DNS
                    service["metadata"]["name"] = instance_name
                    # add the name of the pod the service should connect to
                    service["spec"]["selector"] = { "statefulset.kubernetes.io/pod-name": instance_name }

                    services.append(service)

        return encode_yaml_stream(wormchain_set + services)

    wormchain_path = "devnet/wormchain.yaml"
    if num_guardians >= 2:
        # update wormchain's k8s config to spin up multiple instances
        k8s_yaml_with_ns(build_wormchain_yaml(wormchain_path, num_guardians))
    else:
        k8s_yaml_with_ns(wormchain_path)

    k8s_resource(
        "wormchain",
        port_forwards = [
            port_forward(1319, container_port = 1317, name = "REST [:1319]", host = webHost),
            port_forward(9090, container_port = 9090, name = "GRPC", host = webHost),
            port_forward(26659, container_port = 26657, name = "TENDERMINT [:26659]", host = webHost)
        ],
        labels = ["wormchain"],
        trigger_mode = trigger_mode,
    )

    k8s_resource(
        "wormchain-deploy",
        resource_deps = ["wormchain"],
        labels = ["wormchain"],
        trigger_mode = trigger_mode,
    )

if ibc_relayer:
    docker_build(
        ref = "ibc-relayer-image",
        context = ".",
        dockerfile = "./wormchain/ibc-relayer/Dockerfile",
        only = []
    )

    k8s_yaml_with_ns("devnet/ibc-relayer.yaml")

    k8s_resource(
        "ibc-relayer",
        port_forwards = [
            port_forward(7597, name = "HTTPDEBUG [:7597]", host = webHost),
        ],
        resource_deps = ["wormchain-deploy", "terra2-terrad"],
        labels = ["ibc-relayer"],
        trigger_mode = trigger_mode,
    )

if btc:
    k8s_yaml_with_ns("devnet/btc-localnet.yaml")

    docker_build(
        ref = "btc-node",
        context = "bitcoin",
        dockerfile = "bitcoin/Dockerfile",
        target = "bitcoin-build",
    )

    k8s_resource(
        "btc",
        port_forwards = [
            port_forward(18556, name = "RPC [:18556]", host = webHost),
        ],
        labels = ["btc"],
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
        labels = ["aptos"],
        trigger_mode = trigger_mode,
    )

if aztec:
    k8s_yaml_with_ns("devnet/aztec-devnet.yaml")
    k8s_resource(
        "aztec-sandbox",
        port_forwards = [
            port_forward(8090, name = "RPC [:8090]", host = webHost)
        ],
        labels = ["aztec-sandbox"],
        trigger_mode = trigger_mode,
    )

def build_query_server_yaml():
    qs_yaml = read_yaml_stream("devnet/query-server.yaml")

    for obj in qs_yaml:
        if obj["kind"] == "StatefulSet" and obj["metadata"]["name"] == "query-server":
            container = obj["spec"]["template"]["spec"]["containers"][0]
            container["command"] += ["--bootstrap="+ccqBootstrapPeers]

    return encode_yaml_stream(qs_yaml)

if query_server:
    k8s_yaml_with_ns(build_query_server_yaml())

    k8s_resource(
        "query-server",
        resource_deps = ["guardian"],
        port_forwards = [
            port_forward(6069, name = "REST [:6069]", host = webHost),
            port_forward(6068, name = "Status [:6068]", host = webHost)
        ],
        labels = ["query-server"],
        trigger_mode = trigger_mode
    )
