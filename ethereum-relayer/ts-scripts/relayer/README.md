# How to use these scripts

## Configuration

Private keys should be placed in a .env file corresponding to the Environment you intend to work in. For example, tilt private keys should be kept in ./.env.tilt

If you do not set an environment, the 'default' environment will be used, and .env will be read.

All other configuration is done through files in the ./config/\${env} directory.

./config/\${env}/chains.json is the file which controls how many chains will be executed against, as well as their RPC and basic info.

./config/\${env}/contracts.json is the file which allows you to target specific contracts on each chain.

./config/\${env}/scriptConfigs contains custom configurations for individual scripts. Not all scripts have custom arguments.

## Running the scripts

All files in the coreRelayer, deliveryProvider, and MockIntegration directories are runnable. These are intended to run from the /ethereum directory.

The target environment must be passed in as an environment variable. So, for example, you can run the DeliveryProvider deployment script by running:

```
ENV=tilt ts-node ./ts-scripts/relayer/deliveryProvider/deployDeliveryProvider.ts
```

## Chaining multiple scripts

Scripts are meant to be run individually or successively. Scripts which deploy contracts will write the deployed addresses into the ./output folder.

If "useLastRun" is set to true in the contracts.json configuration file, the lastrun files from the deployment scripts will be used, rather than the deployed addresses of the contracts.json file. This allows you to easily run things like

```
ENV=tilt ts-node ./ts-scripts/relayer/deliveryProvider/upgradeDeliveryProvider.ts && ts-node ./ts-scripts/relayer/mockIntegration/messageTest.ts
```

The ./shell directory contains shell scripts which combine commonly chained actions together.

For example, ./shell/deployConfigureTest.sh will deploy the DeliveryProvider, WormholeRelayer, and MockIntegration contracts. Configure them all to point at each other, and then run messageTest to test that everything worked. Note: useLastRun in contracts.json needs to be set to "true" in order for this script to work.
