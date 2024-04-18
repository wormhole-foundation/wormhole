rm -rf flattened
mkdir flattened
for fileName in \
contracts/Wormhole.sol \
contracts/Implementation.sol \
contracts/bridge/token/Token.sol \
contracts/bridge/token/TokenImplementation.sol \
contracts/bridge/BridgeImplementation.sol \
contracts/bridge/TokenBridge.sol \
contracts/nft/token/NFTImplementation.sol \
contracts/nft/NFTBridgeImplementation.sol \
contracts/nft/NFTBridgeEntrypoint.sol
  do
	echo $fileName
	flattened=flattened/`basename $fileName`
	forge flatten --output $flattened $fileName
done
