# Developing the bridge

## Local Devnet

The following dependencies are required for local development:

- [Go](https://golang.org/dl/) >= 1.21.9 (latest minor release is recommended)
- [Tilt](http://tilt.dev/) >= 0.20.8
- Any of the local Kubernetes clusters supported by Tilt.
  We strongly recommend [minikube](https://kubernetes.io/docs/setup/learning-environment/minikube/) >=
  v1.21.0 with the kvm2 driver.
  - Tilt will use Minikube's embedded Docker server. If Minikube is not used, a local instance of
    [Docker](https://docs.docker.com/engine/install/) / moby-engine >= 19.03 is required.

See the [Tilt docs](https://docs.tilt.dev/install.html) docs on how to set up your local cluster -
it won't take more than a few minutes to set up! Example minikube invocation, adjust limits as needed:

    minikube start --cpus=8 --memory=8G --disk-size=50G --driver=kvm2

npm wants to set up an insane number of inotify watches in the web container which may exceed kernel limits.
The minikube default is too low, adjust it like this:

    minikube ssh 'echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf && sudo sysctl -p'

This should work on Linux, MacOS and Windows.

By default, the devnet is deployed to the `wormhole` namespace rather than `default`. This makes it easy to clean up the
entire deployment by simply removing the namespace, which isn't possible with `default`. Change your default namespace
to avoid having to specify `-n wormhole` for all commands:

    kubectl config set-context --current --namespace=wormhole

After installing all dependencies, just run `tilt up`.
Whenever you modify a file, the devnet is automatically rebuilt and a rolling update is done.

Launch the devnet:

    tilt up

By default this runs a network consisting of one guardian, two anvil (Eth) chains, a Solana test validator, an Algorand sandbox, and LocalTerra for both Terra Classic and Terra 2. If you want to work on non-consensus parts of the code, running with a single guardian is easiest since you won't have to wait for k8s to restart all pods. See the usage guide below for arguments to customize the tilt network.

## Usage

Watch pod status in your cluster:

    kubectl get pod -A -w

Get logs for single guardian node:

    kubectl logs guardian-0

Restart a specific pod:

    kubectl delete pod guardian-0

Adjust number of nodes in running cluster: (this is only useful if you want to test scenarios where the number
of nodes diverges from the guardian set - otherwise, `tilt down --delete-namespaces` and restart the cluster)

    tilt args -- --num=2

Run without all optional networks:

    tilt up -- --algorand=false --evm2=false --solana=false --terra_classic=false --terra2=false

Tear down cluster:

    tilt down --delete-namespaces

Once you're done, press Ctrl-C. Run `tilt down` to tear down the devnet.

## Getting started on a development VM

This tutorial assumes a clean Debian >=10 VM. We recommend at least **16 vCPU, 64G of RAM and 500G of disk**.
Rust eats CPU for breakfast, so the more CPUs, the nicer your Solana compilation experience will be.

Install Git first:

    sudo apt-get install -y git

First, create an SSH key on the VM:

    ssh-keygen -t ed25519
    cat .ssh/id_ed25519.pub

You can then [add your public key on GitHub](https://github.com/settings/keys) and clone the repository:

    git clone git@github.com:wormhole-foundation/wormhole.git

Configure your Git identity:

    git config --global user.name "Your Name"
    git config --global user.email "yourname@company.com"

Your email address should be linked to your personal or company GitHub account.

### Set up devnet on the VM

After cloning the repo, run the setup script. It expects to run as a regular user account with sudo permissions.
It installs Go, Minikube, Tilt and any other dependencies required for Wormhole development:

    cd wormhole
    scripts/dev-setup.sh

You then need to close and re-open your session to apply the new environment.
If you use ControlMaster SSH sessions, make sure to kill the session before reconnecting (`ssh -O exit hostname`).

Start a minikube session with recommended parameters:

    start-recommended-minikube

You can then run tilt normally (see above).

The easiest way to get access to the Tilt UI is to simply run Tilt on a public port, and use a firewall
of your choice to control access. For GCP, we ship a script that automatically runs `tilt up` on the right IP:

    scripts/tilt-gcp-up.sh

If something breaks, just run `minikube delete` and start from scratch by running `start-recommended-minikube`.

### VSCode remote development

VSCode's SSH remote development plugin is known to work well with the workflow described above.

### IntelliJ remote development

IntelliJ's [remote development backend](https://www.jetbrains.com/remote-development/gateway/) is reported to work as well. Just install Jetbrains Gateway on your local machine, connect it to your remote VM via SSH, and pick the latest IntelliJ release. Your local license, keymap and theme - if any - will be used automatically.

[Projector](https://lp.jetbrains.com/projector/) should also work for clients that can't run the native UI locally
(if you want to code on your VR headset, smart toaster or Chromebook - this is the way!).

## Tips and tricks

### Generate protos (Go / TypeScript)

As of [#1352](https://github.com/wormhole-foundation/wormhole/pull/1352), the tsproto generated ts files are provided in two npm packages for [node](./sdk/js-proto-node/) and [web](./sdk/js-proto-web/)

As of [#1824](https://github.com/wormhole-foundation/wormhole/pull/1824), changes to the proto files must match the generated go files.

To re-generate these files run `rm -rf node/pkg/proto && docker build --target go-export -f Dockerfile.proto -o type=local,dest=node .` from the root of the repo.

### Call gRPC services

<!-- cspell:disable -->

    tools/bin/grpcurl -protoset <(tools/bin/buf build -o -) -plaintext localhost:7072 spy.v1.SpyRPCService/SubscribeSignedVAA

<!-- cspell:enable -->

With parameters (using proto json encoding):

<!-- cspell:disable -->

    tools/bin/grpcurl -protoset <(tools/bin/buf build -o -) \
        -d '{"filters": [{"emitter_filter": {"emitter_address": "574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d", "chain_id": "CHAIN_ID_SOLANA"}}]}' \
        -plaintext localhost:7072 spy.v1.SpyRPCService/SubscribeSignedVAA

<!-- cspell:enable -->

### Post messages

To Solana:

<!-- cspell:disable -->

    kubectl exec solana-devnet-0 -c setup -- client post-message Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o 1 confirmed ffff

<!-- cspell:enable -->

To Solana as CPI instruction:

<!-- cspell:disable -->

    kubectl exec solana-devnet-0 -c setup -- client post-message --proxy CP1co2QMMoDPbsmV7PGcUTLFwyhgCgTXt25gLQ5LewE1 Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o 1 confirmed ffff

<!-- cspell:enable -->

### Observation Requests

    kubectl exec -it guardian-0 -- /guardiand admin send-observation-request --socket /tmp/admin.sock 1 4636d8f7593c78a5092bed13dec765cc705752653db5eb1498168c92345cd389

### IntelliJ Protobuf Autocompletion

Locally compile protos to populate the buf cache:

    make generate

Set the include path:

![](https://i.imgur.com/bDij6Cu.png)

### Algorand

Node logs:

    kubectl exec -c algod algorand-0 -- tail -f /network/Node/node.log
    kubectl exec -c algod algorand-0 -- tail -f /network/Primary/node.log

Account list:

    kubectl exec -c goal-kmd algorand-0 -- ./goal account list

Get yourself a working shell:

    kubectl exec -c goal-kmd algorand-0 -it shell-demo -- /bin/bash

### guardiand debugging

Use the `--guardiand_debug` Tilt argument to run guardiand within a dlv session. The session will be exposed just like
any other Tilt services. You can then connect any IDE that supports Go debugging, like IntelliJ (add a "Go Remote"
target and specify the host and port your Tilt instance runs on).
