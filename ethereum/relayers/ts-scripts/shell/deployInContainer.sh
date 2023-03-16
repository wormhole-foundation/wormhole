 echo "deploying generic relayer contracts" \ 
  npx ts-node ./ts-scripts/relayProvider/deployRelayProvider.ts \
  && npx ts-node ./ts-scripts/coreRelayer/deployCoreRelayer.ts \
  && npx ts-node ./ts-scripts/relayProvider/registerChainsRelayProvider.ts \
  && npx ts-node ./ts-scripts/coreRelayer/registerChainsCoreRelayerSelfSign.ts \
  && npx ts-node ./ts-scripts/relayProvider/configureRelayProvider.ts \
  && npx ts-node ./ts-scripts/mockIntegration/deployMockIntegration.ts \