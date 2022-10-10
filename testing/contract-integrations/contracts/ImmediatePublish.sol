pragma solidity >=0.8.0 <0.9.0;

import "./libraries/external/BytesLib.sol";

interface IWormhole {
    function publishMessage(
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) external payable returns (uint64 sequence);
}

contract ImmediatePublish {

    IWormhole wormhole;

    constructor(address wormholeAddress) {
        wormhole = IWormhole(wormholeAddress);
    }

    function immediatePublish() public payable returns (uint64 sequence) {
        return wormhole.publishMessage{
            value : msg.value
        }(0, bytes("hello"), 200);
    }

}
