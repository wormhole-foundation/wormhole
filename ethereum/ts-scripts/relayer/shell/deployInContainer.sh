 echo "deploying generic relayer contracts" \ 
  npx tsx ./ts-scripts/relayer/create2Factory/deployCreate2Factory.ts \
  && npx tsx ./ts-scripts/relayer/relayProvider/deployRelayProvider.ts \
  && npx tsx ./ts-scripts/relayer/coreRelayer/deployWormholeRelayer.ts \
  && npx tsx ./ts-scripts/relayer/mockIntegration/deployMockIntegration.ts \
  && npx tsx ./ts-scripts/relayer/coreRelayer/registerChainsWormholeRelayerSelfSign.ts \
  && npx tsx ./ts-scripts/relayer/relayProvider/configureRelayProvider.ts \