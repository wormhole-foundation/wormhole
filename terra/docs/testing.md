# Wormhole + Terra local test environment

The following dependencies are required for local development:

- [Go](https://golang.org/dl/) >= 1.15.3
- [Docker](https://docs.docker.com/engine/install/) / moby-engine >= 19.03
- [Tilt](http://tilt.dev/) >= 0.17.2
- Any of the local Kubernetes clusters supported by Tilt. 
  We recommend [minikube](https://kubernetes.io/docs/setup/learning-environment/minikube/) with the kvm2 driver.
- [Node.js](https://nodejs.org/) >= 14.x, [ts-node](https://www.npmjs.com/package/ts-node) >= 8.x

Start Tilt from the project root using `TiltfileTerra` configuration:

    tilt up -f TiltfileTerra

Then expose Terra LCD server port to the localhost:

    kubectl port-forward terra-lcd-0 1317

Afterwards use test scripts in `terra/tools` folder:

    npm install
    npm run prepare-token
    npm run prepare-wormhole

These commands will give you two important addresses: test token address and Wormhole contract address on Terra. Now you need to change guardian configuration to monitor the right contract. Copy Wormhole contract address and replace existing address in file `devnet/bridge-terra.yaml` (line 67). Save the changes and monitor Tilt dashboard until guardian services restart.

Now use both token address and Wormhole contract address to issue tocken lock transaction:

    npm run lock-tocken -- TOKEN_CONTRACT WORMHOLE_CONTRACT 1000

Where 1000 is a sample amount to transfer. After this command is issued monitor Guardian service in Tilt dashboard to see its effects propagated to the destination blockchain (in this case it is Ethereum).