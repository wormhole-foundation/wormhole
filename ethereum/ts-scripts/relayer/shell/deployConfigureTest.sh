npx ts-node ./ts-scripts/relayer/config/checkNetworks.ts --set-last-run \
  && npx ts-node ./ts-scripts/relayer/create2Factory/deployCreate2Factory.ts \
  && npx ts-node ./ts-scripts/relayer/relayProvider/deployRelayProvider.ts \
  && npx ts-node ./ts-scripts/relayer/coreRelayer/deployCoreRelayer.ts \
  && npx ts-node ./ts-scripts/relayer/relayProvider/configureRelayProvider.ts \
  && npx ts-node ./ts-scripts/relayer/coreRelayer/registerChainsCoreRelayerSelfSign.ts \
  && npx ts-node ./ts-scripts/relayer/mockIntegration/deployMockIntegration.ts \
  && npx ts-node ./ts-scripts/relayer/config/syncContractsJson.ts \
  && npx ts-node ./ts-scripts/relayer/mockIntegration/messageTest.ts 

 # put this as 2nd script if not deployed aleady