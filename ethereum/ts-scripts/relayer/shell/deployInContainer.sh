 echo "deploying generic relayer contracts" \ 
  npx tsx ./ts-scripts/relayer/create2Factory/deployCreate2Factory.ts \
  && npx tsx ./ts-scripts/relayer/deliveryProvider/deployDeliveryProvider.ts \
  && npx tsx ./ts-scripts/relayer/wormholeRelayer/deployWormholeRelayer.ts \
  && npx tsx ./ts-scripts/relayer/mockIntegration/deployMockIntegration.ts \
  && npx tsx ./ts-scripts/relayer/wormholeRelayer/registerChainsWormholeRelayerSelfSign.ts \
  && npx tsx ./ts-scripts/relayer/deliveryProvider/configureDeliveryProvider.ts \