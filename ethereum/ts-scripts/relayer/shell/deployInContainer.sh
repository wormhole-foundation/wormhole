 echo "deploying generic relayer contracts" \ 
  npx tsx@4.5.1 ./ts-scripts/relayer/create2Factory/deployCreate2Factory.ts \
  && npx tsx@4.5.1 ./ts-scripts/relayer/deliveryProvider/deployDeliveryProvider.ts \
  && npx tsx@4.5.1 ./ts-scripts/relayer/wormholeRelayer/deployWormholeRelayer.ts \
  && npx tsx@4.5.1 ./ts-scripts/relayer/mockIntegration/deployMockIntegration.ts \
  && npx tsx@4.5.1 ./ts-scripts/relayer/wormholeRelayer/registerChainsWormholeRelayerSelfSign.ts \
  && npx tsx@4.5.1 ./ts-scripts/relayer/deliveryProvider/configureDeliveryProvider.ts \
