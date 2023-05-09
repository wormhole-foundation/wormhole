# These files can be listed through:
# `find contracts/ -name *.sol`

# These are unnecessary. They are imported by other contracts.
# ./contracts/libraries/external/BytesLib.sol
# These are part of the generic relayer project.
# They are compiled with forge.
#./contracts/relayer/relayProvider/RelayProvider.sol
#./contracts/relayer/relayProvider/RelayProviderMessages.sol
#./contracts/relayer/relayProvider/RelayProviderGetters.sol
#./contracts/relayer/relayProvider/RelayProviderProxy.sol
#./contracts/relayer/relayProvider/RelayProviderImplementation.sol
#./contracts/relayer/relayProvider/RelayProviderState.sol
#./contracts/relayer/relayProvider/RelayProviderGovernance.sol
#./contracts/relayer/relayProvider/RelayProviderSetters.sol
#./contracts/relayer/relayProvider/RelayProviderSetup.sol
#./contracts/relayer/relayProvider/RelayProviderStructs.sol
#./contracts/relayer/create2Factory/Create2Factory.sol
#./contracts/relayer/coreRelayer/CoreRelayerStorage.sol
#./contracts/relayer/coreRelayer/CoreRelayerBase.sol
#./contracts/relayer/coreRelayer/CoreRelayer.sol
#./contracts/relayer/coreRelayer/Utils.sol
#./contracts/relayer/coreRelayer/CoreRelayerSend.sol
#./contracts/relayer/coreRelayer/CoreRelayerSerde.sol
#./contracts/relayer/coreRelayer/BytesParsing.sol
#./contracts/relayer/coreRelayer/CoreRelayerGovernance.sol
#./contracts/relayer/coreRelayer/CoreRelayerDelivery.sol

export CORE_CONTRACT_FILES='./contracts/Implementation.sol ./contracts/mock/MockRelayerIntegration.sol ./contracts/mock/MockBatchedVAASender.sol ./contracts/mock/MockImplementation.sol ./contracts/Messages.sol ./contracts/Wormhole.sol ./contracts/interfaces/IWormhole.sol ./contracts/interfaces/relayer/IRelayProvider.sol ./contracts/interfaces/relayer/IWormholeRelayer.sol ./contracts/interfaces/relayer/IWormholeReceiver.sol ./contracts/Setup.sol ./contracts/GovernanceStructs.sol ./contracts/Getters.sol ./contracts/Structs.sol ./contracts/Governance.sol ./contracts/Migrations.sol ./contracts/Setters.sol ./contracts/Shutdown.sol ./contracts/State.sol'

export TOKEN_BRIDGE_CONTRACT_FILES='./contracts/bridge/mock/MockBridgeImplementation.sol ./contracts/bridge/mock/MockFeeToken.sol ./contracts/bridge/mock/MockTokenBridgeIntegration.sol ./contracts/bridge/mock/MockWETH9.sol ./contracts/bridge/mock/MockTokenImplementation.sol ./contracts/bridge/token/Token.sol ./contracts/bridge/token/TokenImplementation.sol ./contracts/bridge/token/TokenState.sol ./contracts/bridge/BridgeImplementation.sol ./contracts/bridge/BridgeGetters.sol ./contracts/bridge/interfaces/ITokenBridge.sol ./contracts/bridge/interfaces/IWETH.sol ./contracts/bridge/utils/Migrator.sol ./contracts/bridge/BridgeSetters.sol ./contracts/bridge/BridgeState.sol ./contracts/bridge/BridgeGovernance.sol ./contracts/bridge/Bridge.sol ./contracts/bridge/BridgeStructs.sol ./contracts/bridge/TokenBridge.sol ./contracts/bridge/BridgeShutdown.sol ./contracts/bridge/BridgeSetup.sol'

export NFT_BRIDGE_CONTRACT_FILES='./contracts/nft/mock/MockNFTImplementation.sol ./contracts/nft/mock/MockNFTBridgeImplementation.sol ./contracts/nft/NFTBridgeSetters.sol ./contracts/nft/token/NFTState.sol ./contracts/nft/token/NFTImplementation.sol ./contracts/nft/token/NFT.sol ./contracts/nft/NFTBridgeEntrypoint.sol ./contracts/nft/NFTBridgeImplementation.sol ./contracts/nft/NFTBridgeSetup.sol ./contracts/nft/NFTBridgeShutdown.sol ./contracts/nft/NFTBridgeStructs.sol ./contracts/nft/NFTBridgeState.sol ./contracts/nft/interfaces/INFTBridge.sol ./contracts/nft/NFTBridge.sol ./contracts/nft/NFTBridgeGovernance.sol ./contracts/nft/NFTBridgeGetters.sol'