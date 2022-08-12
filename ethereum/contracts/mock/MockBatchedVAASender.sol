// contracts/mock/MockBatchedVAASender.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../libraries/external/BytesLib.sol";
import "../interfaces/IWormhole.sol";

contract MockBatchedVAASender {
    using BytesLib for bytes;

    address wormholeCoreAddress;

    function sendMultipleMessages(
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    )
        public
        payable
        returns (
            uint64 messageSequence0,
            uint64 messageSequence1,
            uint64 messageSequence2
        )
    {
        messageSequence0 = wormholeCore().publishMessage{value: msg.value}(
            nonce,
            payload,
            consistencyLevel
        );

        messageSequence1 = wormholeCore().publishMessage{value: msg.value}(
            nonce,
            payload,
            consistencyLevel
        );

        messageSequence2 = wormholeCore().publishMessage{value: msg.value}(
            nonce,
            payload,
            consistencyLevel
        );
    }

    function wormholeCore() private view returns (IWormhole) {
        return IWormhole(wormholeCoreAddress);
    }

    function setup(address _wormholeCore) public {
        wormholeCoreAddress = _wormholeCore;
    }
}
