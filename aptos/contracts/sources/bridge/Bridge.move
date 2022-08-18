
// module Wormhole::Bridge {
//     use 0x1::type_info::{Self, TypeInfo};
//     use Wormhole::VAA::{Self, VAA, parseAndVerifyVAA};
//     use Wormhole::BridgeState::{setOutstandingBridged, outstandingBridged, bridgeContracts};
//     //use Wormhole::BridgeStructs::{AssetMeta, Transfer, TransferWithPayload};

//     public entry fun attestToken<CoinType>(deployer: address){
        
//     }

//     public entry fun createWrapped(encodedVM: vector<u8>){        
//         let (vaa, result, reason) = parseAndVerifyVAA(encodedVM);
//         VAA::destroy(vaa);

//     //     //require(valid, reason);
//     //     //require(verifyBridgeVM(vm), "invalid emitter");

//     //     BridgeStructs.AssetMeta memory meta = parseAssetMeta(vm.payload);
//     //     //return _createWrapped(meta, vm.sequence);
//     //     let sequence = vaa.sequence;

//     //     assert!(meta.tokenChain != chainId(), 0);
//     //     assert!(wrappedAsset(meta.tokenChain, meta.tokenAddress) == address(0), 0);

//     // }
//     //     // initialize the TokenImplementation
//     //     bytes memory initialisationArgs = abi.encodeWithSelector(
//     //         TokenImplementation.initialize.selector,
//     //         bytes32ToString(meta.name),
//     //         bytes32ToString(meta.symbol),
//     //         meta.decimals,
//     //         sequence,

//     //         address(this),

//     //         meta.tokenChain,
//     //         meta.tokenAddress
//     //     );

//     //     // initialize the BeaconProxy
//     //     bytes memory constructorArgs = abi.encode(address(this), initialisationArgs);

//     //     // deployment code
//     //     bytes memory bytecode = abi.encodePacked(type(BridgeToken).creationCode, constructorArgs);

//     //     bytes32 salt = keccak256(abi.encodePacked(meta.tokenChain, meta.tokenAddress));

//     //     assembly {
//     //         token := create2(0, add(bytecode, 0x20), mload(bytecode), salt)

//     //         if iszero(extcodesize(token)) {
//     //             revert(0, 0)
//     //         }
//     //     }
//     //     setWrappedAsset(meta.tokenChain, meta.tokenAddress, token);
//     }

//     public fun transferTokensWithPayload(
//         address token,
//         uint256 amount,
//         uint16 recipientChain,
//         bytes32 recipient,
//         uint32 nonce,
//         bytes memory payload
//     ): u64 {
//         BridgeStructs.TransferResult memory transferResult = _transferTokens(
//             token,
//             amount,
//             0
//         );
//         sequence = logTransferWithPayload(
//             transferResult.tokenChain,
//             transferResult.tokenAddress,
//             transferResult.normalizedAmount,
//             recipientChain,
//             recipient,
//             transferResult.wormholeFee,
//             nonce,
//             payload
//         );
//     }

//     /*
//      *  @notice Initiate a transfer
//      */
//     fun _transferTokens(token: TypeInfo, amount: u128, arbiterFee: u128) internal returns (BridgeStructs.TransferResult memory transferResult) {
//         // determine token parameters
//         uint16 tokenChain;
//         bytes32 tokenAddress;
//         if (isWrappedAsset(token)) {
//             tokenChain = TokenImplementation(token).chainId();
//             tokenAddress = TokenImplementation(token).nativeContract();
//         } else {
//             tokenChain = chainId();
//             tokenAddress = bytes32(uint256(uint160(token)));
//         }

//         // query tokens decimals
//         (,bytes memory queriedDecimals) = token.staticcall(abi.encodeWithSignature("decimals()"));
//         uint8 decimals = abi.decode(queriedDecimals, (uint8));

//         // don't deposit dust that can not be bridged due to the decimal shift
//         amount = deNormalizeAmount(normalizeAmount(amount, decimals), decimals);

//         if (tokenChain == chainId()) {
//             // query own token balance before transfer
//             (,bytes memory queriedBalanceBefore) = token.staticcall(abi.encodeWithSelector(IERC20.balanceOf.selector, address(this)));
//             uint256 balanceBefore = abi.decode(queriedBalanceBefore, (uint256));

//             // transfer tokens
//             SafeERC20.safeTransferFrom(IERC20(token), msg.sender, address(this), amount);

//             // query own token balance after transfer
//             (,bytes memory queriedBalanceAfter) = token.staticcall(abi.encodeWithSelector(IERC20.balanceOf.selector, address(this)));
//             uint256 balanceAfter = abi.decode(queriedBalanceAfter, (uint256));

//             // correct amount for potential transfer fees
//             amount = balanceAfter - balanceBefore;
//         } else {
//             SafeERC20.safeTransferFrom(IERC20(token), msg.sender, address(this), amount);

//             TokenImplementation(token).burn(address(this), amount);
//         }

//         // normalize amounts decimals
//         uint256 normalizedAmount = normalizeAmount(amount, decimals);
//         uint256 normalizedArbiterFee = normalizeAmount(arbiterFee, decimals);

//         // track and check outstanding token amounts
//         if (tokenChain == chainId()) {
//             bridgeOut(token, normalizedAmount);
//         }

//         transferResult = BridgeStructs.TransferResult({
//             tokenChain : tokenChain,
//             tokenAddress : tokenAddress,
//             normalizedAmount : normalizedAmount,
//             normalizedArbiterFee : normalizedArbiterFee,
//             wormholeFee : msg.value
//         });
//     }


//     fun bridgeOut(token: TypeInfo, normalizedAmount: u128) {
//         let outstanding = outstandingBridged(token);
//         assert!(outstanding + normalizedAmount <= 2<<128 - 1, 0);
//         setOutstandingBridged(token, outstanding + normalizedAmount);
//     }

//     fun bridgedIn(token: TypeInfo, normalizedAmount: u128) {
//         setOutstandingBridged(token, outstandingBridged(token) - normalizedAmount);
//     }

//     fun verifyBridgeVM(vm: &VAA): bool{
//         if (bridgeContracts(VAA::get_emitter_chain(vm)) == VAA::get_emitter_address(vm)) {
//             return true
//         };
//         return false
//     }

// } 





