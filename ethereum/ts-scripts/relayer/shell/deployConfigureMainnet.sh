
# This pipeline doesn't attempt to sign governance VAAs since those need to be signed by the guardians.
npx tsx ./ts-scripts/relayer/config/checkNetworks.ts \
  && npx tsx ./ts-scripts/relayer/deliveryProvider/deployDeliveryProvider.ts \
  && npx tsx ./ts-scripts/relayer/wormholeRelayer/deployWormholeRelayer.ts \
  && npx tsx ./ts-scripts/relayer/deliveryProvider/initializeDeliveryProvider.ts \
  && npx tsx ./ts-scripts/relayer/mockIntegration/deployMockIntegration.ts \
  && npx tsx ./ts-scripts/relayer/wormholeRelayer/registerChainsWormholeRelayer.ts
