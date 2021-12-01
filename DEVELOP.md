# Developing the bridge

## Local Devnet

The following dependencies are required for local development:

- [Go](https://golang.org/dl/) >= 1.17.0
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

Launch the devnet while specifying the number of guardians nodes to run (default is five):

    tilt up -- --num=1

If you want to work on non-consensus parts of the code, running with a single guardian is easiest since
you won't have to wait for k8s to restart all pods.

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

Tear down cluster:

    tilt down --delete-namespaces

Once you're done, press Ctrl-C. Run `tilt down` to tear down the devnet.

## Getting started on a development VM

This tutorial assumes a clean Debian >=10 VM. We recommend at least **16 vCPU, 64G of RAM and 500G of disk**.
Rust eats CPU for breakfast, so the more CPUs, the nicer your Solana compilation experience will be.

Install Git first:

    sudo apt-get install -y git

First, create an SSH key in your shell:

    ssh-keygen -t ed25519
    cat .ssh/id_ed25519.pub

You can then [add your public key on Gerrit](https://forge.certus.one/settings/#SSHKeys) and [clone the repository](https://forge.certus.one/admin/repos/wormhole) in your shell. The clone command should look like this:

```shell
# don't copy this - copy it from Gerrit instead
git clone "ssh://yourusername@forge.certus.one:30322/wormhole" && scp -p -P 30322 yourusername@forge.certus.one:hooks/commit-msg "wormhole/.git/hooks/"
```

Configure your Git identity to match your Gerrit name and email:

    git config --global user.name "Your Name"
    git config --global user.email "yourname@company.com"

*If you are a Jump Crypto employee, make sure to log into Gerrit using Azure SSO using your
jumptrading.com email address and request Gerrit review permissions in #wormhole-dev on Slack.*

You can then use our regular Git and Gerrit workflow as detailed in [Submit change for review](./CONTRIBUTING.md#Submit change for review).

### Set up devnet on the VM

After cloning the repo, run the setup script:

    scripts/devnet-setup.sh

You then need to close and re-open your session to apply the new environment.
If you use persistent SSH sessions, make sure to kill the session before reconnecting.

You can then run tilt normally (see above).

The easiest way to get access to the Tilt UI is to simply run Tilt on a public port, and use a firewall
of your choice to control access:

    tilt up --host=0.0.0.0 --port=8080

## Tips and tricks

### Post messages

To Solana:

    kubectl exec solana-devnet-0 -c setup -- client post-message Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o 1 confirmed ffff

To Solana as CPI instruction:

    kubectl exec solana-devnet-0 -c setup -- client post-message --proxy CP1co2QMMoDPbsmV7PGcUTLFwyhgCgTXt25gLQ5LewE1 Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o 1 confirmed ffff


### IntelliJ Protobuf Autocompletion

Locally compile protos to populate the buf cache:

    make generate

Set the include path:

![](https://i.imgur.com/bDij6Cu.png)


### BigTable event persistence

Guardian events can be persisted to a cloud BigTable instance by passing a GCP project and service account key to Tilt.
Launch the devnet with flags supplying your database info to forward events to your cloud BigTable, rather than the local devnet BigTable emulator:

    tilt up -- --num=1  --gcpProject=your-project-id --bigTableKeyPath=./your-service-account-key.json

### bridge UI

Run the bridge UI in devnet by supplying the `--bridge_ui` flag:

    tilt up -- --bridge_ui

