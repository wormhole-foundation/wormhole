# Developing the bridge

## Local Devnet

The following dependencies are required for local development:

- Go >= 1.14
- Docker >= 19.03
- Tilt >= 0.17.0
- Any of the local Kubernetes clusters supported by Tilt (we recommend minikube or kind).

See the [Tilt docs](https://docs.tilt.dev/install.html) docs on how to set up your local cluster -
it won't take more than a few minutes to set up!

This works natively on both Linux and MacOS.

Then, just run `tilt up`. Whenever you modify a file, the devnet is automatically rebuilt.

Once you're done, press Ctrl-C. Run `tilt down` to tear down the devnet.
