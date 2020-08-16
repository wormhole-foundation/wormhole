# Developing the bridge

## Local Devnet

The following dependencies are required for local development:

- [Go](https://golang.org/dl/) >= 1.14
- [Docker](https://docs.docker.com/engine/install/) / moby-engine >= 19.03
- [Tilt](http://tilt.dev/) >= 0.17.2

- Any of the local Kubernetes clusters supported by Tilt 
  (we recommend [minikube](https://kubernetes.io/docs/setup/learning-environment/minikube/) using the VM or Docker driver).

See the [Tilt docs](https://docs.tilt.dev/install.html) docs on how to set up your local cluster -
it won't take more than a few minutes to set up!

This should work on Linux, MacOS and possibly even Windows.

After installing all dependencies, just run `tilt up --update-mode=exec`. 
Whenever you modify a file, the devnet is automatically rebuilt and a rolling update is done.

Watch pod status in your cluster: `kubectl get pod -A -w`.

Once you're done, press Ctrl-C. Run `tilt down` to tear down the devnet.
