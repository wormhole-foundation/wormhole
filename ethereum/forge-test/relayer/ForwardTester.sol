// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../../contracts/interfaces/IWormhole.sol";
import "../../contracts/interfaces/relayer/IWormholeReceiver.sol";
import "../../contracts/interfaces/relayer/IWormholeRelayerTyped.sol";
import "../../contracts/interfaces/relayer/IDeliveryProviderTyped.sol";
import "../../contracts/libraries/external/BytesLib.sol";
import "./MockGenericRelayer.sol";
import "forge-std/console.sol";
import "forge-std/Vm.sol";

contract ForwardTester is IWormholeReceiver {
    using BytesLib for bytes;

    IWormhole wormhole;
    IWormholeRelayer wormholeRelayer;
    MockGenericRelayer genericRelayer;

    uint32 TOO_LOW_GAS_LIMIT = 10000;
    uint32 REASONABLE_GAS_LIMIT = 500000;

    address private constant VM_ADDRESS =
        address(bytes20(uint160(uint256(keccak256("hevm cheat code")))));

    Vm public constant vm = Vm(VM_ADDRESS);

    constructor(address _wormhole, address _wormholeRelayer, address _wormholeSimulator) {
        wormhole = IWormhole(_wormhole);
        wormholeRelayer = IWormholeRelayer(_wormholeRelayer);
        genericRelayer = new MockGenericRelayer(_wormhole, _wormholeSimulator);
        genericRelayer.setWormholeRelayerContract(wormhole.chainId(), address(wormholeRelayer));
    }

    enum Action {
        ForwardRequestFromWrongAddress,
        ProviderNotSupported,
        ReentrantCall,
        WorksCorrectly
    }

    function receiveWormholeMessages(
        bytes memory payload,
        bytes[] memory,
        bytes32 sourceAddress,
        uint16 sourceChain,
        bytes32
    ) public payable override {
        Action action = Action(payload.toUint8(0));

        if (action == Action.ForwardRequestFromWrongAddress) {
            // Emitter must be a wormhole relayer
            DummyContract dc = new DummyContract(address(wormholeRelayer));
            dc.forward{value: msg.value}(
                sourceChain, fromWormholeFormat(sourceAddress), REASONABLE_GAS_LIMIT, 0, bytes("")
            );
        } else if (action == Action.ProviderNotSupported) {
            wormholeRelayer.forwardPayloadToEvm{value: msg.value}(
                32,
                fromWormholeFormat(sourceAddress),
                bytes(""),
                TargetNative.wrap(0),
                Gas.wrap(REASONABLE_GAS_LIMIT)
            );
        } else if (action == Action.ReentrantCall) {
            (uint256 deliveryPrice,) =
                wormholeRelayer.quoteEVMDeliveryPrice(sourceChain, 0, REASONABLE_GAS_LIMIT);
            vm.recordLogs();
            wormholeRelayer.sendPayloadToEvm{
                value: deliveryPrice + wormhole.messageFee() + msg.value
            }(
                sourceChain,
                fromWormholeFormat(sourceAddress),
                bytes(""),
                TargetNative.wrap(0),
                Gas.wrap(REASONABLE_GAS_LIMIT)
            );
            genericRelayer.relay(wormhole.chainId());
        } else {
            wormholeRelayer.forwardPayloadToEvm{value: msg.value}(
                sourceChain,
                fromWormholeFormat(sourceAddress),
                bytes(""),
                TargetNative.wrap(0),
                Gas.wrap(REASONABLE_GAS_LIMIT)
            );
        }
    }

    function fromWormholeFormat(bytes32 whAddress) public pure returns (address addr) {
        return address(uint160(uint256(whAddress)));
    }

    receive() external payable {}
}

contract DummyContract {
    IWormholeRelayer wormholeRelayer;

    constructor(address _wormholeRelayer) {
        wormholeRelayer = IWormholeRelayer(_wormholeRelayer);
    }

    function forward(
        uint16 chainId,
        address targetAddress,
        uint32 gasLimit,
        uint256 receiverValue,
        bytes memory payload
    ) public payable {
        wormholeRelayer.forwardPayloadToEvm{value: msg.value}(
            chainId, targetAddress, payload, TargetNative.wrap(receiverValue), Gas.wrap(gasLimit)
        );
    }
}
