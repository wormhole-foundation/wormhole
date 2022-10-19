// This is a simple contract to generate Wormhole messages.
// It allows you to populate the consistency level in the message.
// It can be used to test the guardian watcher.

pragma solidity >=0.8.0 <0.9.0;

import "./libraries/external/BytesLib.sol";

interface IWormhole {
    function publishMessage(
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) external payable returns (uint64 sequence);
}

contract PublishMsg {

    IWormhole wormhole;

    constructor(address wormholeAddress) {
        wormhole = IWormhole(wormholeAddress);
    }

    function publishMsg(uint8 consistencyLevel) public payable returns (uint64 sequence) {
        return wormhole.publishMessage{
            value : msg.value
        }(0, bytes("hello"), consistencyLevel);
    }

}
