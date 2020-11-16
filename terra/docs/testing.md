# Wormhole + Terra local test environment

For the list of dependencies please follow [DEVELOP.md](../../DEVELOP.md)

Additional dependencies:
- [Node.js](https://nodejs.org/) >= 14.x, [ts-node](https://www.npmjs.com/package/ts-node) >= 8.x

Start Tilt from the project root:

    tilt up

Afterwards use test scripts in `terra/tools` folder:

    npm install
    npm run prepare-token
    npm run prepare-wormhole

These commands will give you two important addresses: test token address and Wormhole contract address on Terra. Now you need to change guardian configuration to monitor the right contract. Copy Wormhole contract address and replace existing address in file `devnet/bridge-terra.yaml` (line 67). Save the changes and monitor Tilt dashboard until guardian services restart.

Now use both token address and Wormhole contract address to issue tocken lock transaction:

    npm run lock-tocken -- TOKEN_CONTRACT WORMHOLE_CONTRACT 1000

Where 1000 is a sample amount to transfer. After this command is issued monitor Guardian service in Tilt dashboard to see its effects propagated to the destination blockchain (in this case it is Ethereum).