 echo "deploying generic relayer contracts" \ 
  npx ts-node ./ts-scripts/relayer/relayProvider/deployRelayProvider.ts \
  && npx ts-node ./ts-scripts/relayer/coreRelayer/deployCoreRelayer.ts \
  && npx ts-node ./ts-scripts/relayer/relayProvider/registerChainsRelayProvider.ts \
  && npx ts-node ./ts-scripts/relayer/coreRelayer/registerChainsCoreRelayerSelfSign.ts \
  && npx ts-node ./ts-scripts/relayer/relayProvider/configureRelayProvider.ts \
  && npx ts-node ./ts-scripts/relayer/mockIntegration/deployMockIntegration.ts \