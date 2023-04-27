 echo "deploying generic relayer contracts" \ 
  npx ts-node ./ts-scripts/relayer/create2Factory/deployCreate2Factory.ts \
  && npx ts-node ./ts-scripts/relayer/relayProvider/deployRelayProvider.ts \
  && npx ts-node ./ts-scripts/relayer/coreRelayer/deployCoreRelayer.ts \
  && npx ts-node ./ts-scripts/relayer/mockIntegration/deployMockIntegration.ts \
  && npx ts-node ./ts-scripts/relayer/coreRelayer/registerChainsCoreRelayerSelfSign.ts \
  && npx ts-node ./ts-scripts/relayer/relayProvider/configureRelayProvider.ts \