// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "../../contracts/interfaces/relayer/IWormholeReceiver.sol";
import "../../contracts/libraries/relayer/BytesParsing.sol";

uint256 constant uint256Length = 32;

/**
 * This contract is meant to test different kinds of extreme scenarios when an integration returns data
 * after its `receiveWormholeMessages` interface is called.
 * 
 * Only meant for testing purposes.
 */
contract BigRevertBufferIntegration is IWormholeReceiver {
    using BytesParsing for bytes;
    // This is the function which receives all messages from the remote contracts.
    function receiveWormholeMessages(
        bytes memory payload,
        bytes[] memory /*additionalVaas*/,
        bytes32 /*sourceAddress*/,
        uint16 /*sourceChain*/,
        bytes32 /*deliveryHash*/
    ) public payable override {
        (uint256 revertLength,) = payload.asUint256(0);
        bytes memory revertBuffer = new bytes(revertLength);
        for (uint256 i = 0; i < revertBuffer.length; ++i) {
            revertBuffer[i] = bytes1(uint8(i));
        }

        // We avoid reverting with the standard `Error(string)` here because it may mess up terminals with these garbage bytes
        // It's easier to predict what to test with this anyway.
        assembly ("memory-safe") {
            let buf := add(revertBuffer, uint256Length)
            revert(buf, revertLength)
        }
    }
}
