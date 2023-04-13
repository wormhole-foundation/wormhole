npx ts-node ./ts-scripts/relayer/config/checkNetworks.ts --last-run=true \
  && npx ts-node ./ts-scripts/relayer/relayProvider/deployRelayProvider.ts \
  && npx ts-node ./ts-scripts/relayer/coreRelayer/deployCoreRelayer.ts \
  && npx ts-node ./ts-scripts/relayer/coreRelayer/registerChainsCoreRelayerSelfSign.ts \
  && npx ts-node ./ts-scripts/relayer/relayProvider/configureRelayProvider.ts \
  && npx ts-node ./ts-scripts/relayer/mockIntegration/deployMockIntegration.ts \
  && npx ts-node ./ts-scripts/relayer/mockIntegration/messageTest.ts \
  && npx ts-node ./ts-scripts/relayer/config/syncContractsJson.ts