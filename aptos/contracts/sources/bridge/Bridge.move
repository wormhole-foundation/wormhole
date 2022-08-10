
module Wormhole::Bridge {
    use 0x1::type_info::{Self, TypeInfo};
    use Wormhole::VAA::{Self, parseAndVerifyVAA};

    public entry fun attestToken<CoinType>(deployer: address){
        
    }

    public entry fun createWrapped(encodedVM: vector<u8>){        
        let (vaa, result, reason) = parseAndVerifyVAA(encodedVM);
        VAA::destroy(vaa);

    //     //require(valid, reason);
    //     //require(verifyBridgeVM(vm), "invalid emitter");

    //     BridgeStructs.AssetMeta memory meta = parseAssetMeta(vm.payload);
    //     //return _createWrapped(meta, vm.sequence);
    //     let sequence = vaa.sequence;

    //     assert!(meta.tokenChain != chainId(), 0);
    //     assert!(wrappedAsset(meta.tokenChain, meta.tokenAddress) == address(0), 0);


    // }
    //     // initialize the TokenImplementation
    //     bytes memory initialisationArgs = abi.encodeWithSelector(
    //         TokenImplementation.initialize.selector,
    //         bytes32ToString(meta.name),
    //         bytes32ToString(meta.symbol),
    //         meta.decimals,
    //         sequence,

    //         address(this),

    //         meta.tokenChain,
    //         meta.tokenAddress
    //     );

    //     // initialize the BeaconProxy
    //     bytes memory constructorArgs = abi.encode(address(this), initialisationArgs);

    //     // deployment code
    //     bytes memory bytecode = abi.encodePacked(type(BridgeToken).creationCode, constructorArgs);

    //     bytes32 salt = keccak256(abi.encodePacked(meta.tokenChain, meta.tokenAddress));

    //     assembly {
    //         token := create2(0, add(bytecode, 0x20), mload(bytecode), salt)

    //         if iszero(extcodesize(token)) {
    //             revert(0, 0)
    //         }
    //     }

    //     setWrappedAsset(meta.tokenChain, meta.tokenAddress, token);
    }

    //  function bridgeOut(address token, uint normalizedAmount) internal {
    //     uint outstanding = outstandingBridged(token);
    //     require(outstanding + normalizedAmount <= type(uint64).max, "transfer exceeds max outstanding bridged token amount");
    //     setOutstandingBridged(token, outstanding + normalizedAmount);
    // }

    // function bridgedIn(address token, uint normalizedAmount) internal {
    //     setOutstandingBridged(token, outstandingBridged(token) - normalizedAmount);
    // }

} 




