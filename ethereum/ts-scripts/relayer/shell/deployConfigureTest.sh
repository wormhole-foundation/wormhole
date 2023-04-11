ts-node ./ts-scripts/relayer/config/checkNetworks.ts \
  && ts-node ./ts-scripts/relayer/relayProvider/deployRelayProvider.ts \
  && ts-node ./ts-scripts/relayer/coreRelayer/deployCoreRelayer.ts \
  && ts-node ./ts-scripts/relayer/coreRelayer/registerChainsCoreRelayerSelfSign.ts \
  && ts-node ./ts-scripts/relayer/relayProvider/configureRelayProvider.ts \
  && ts-node ./ts-scripts/relayer/mockIntegration/deployMockIntegration.ts \
  && ts-node ./ts-scripts/relayer/mockIntegration/messageTest.ts \
  && ts-node ./ts-scripts/relayer/config/syncContractsJson.ts