// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../../contracts/interfaces/IWormhole.sol";
import "../../contracts/interfaces/relayer/IWormholeReceiver.sol";
import "../../contracts/interfaces/relayer/IWormholeRelayer.sol";
import "../../contracts/interfaces/relayer/IRelayProvider.sol";
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
        DeliveryData memory deliveryData,
        bytes[] memory vaas
    ) public payable override {

        bytes memory payload = deliveryData.payload;
        Action action = Action(payload.toUint8(0));


        if (action == Action.ForwardRequestFromWrongAddress) {
            // Emitter must be a wormhole relayer
            DummyContract dc = new DummyContract(address(wormholeRelayer));
            dc.forward{value: msg.value}(
                deliveryData.sourceChainId,
                fromWormholeFormat(deliveryData.sourceAddress),
                REASONABLE_GAS_LIMIT,
                0,
                bytes("")
            );
        } else if (action == Action.ProviderNotSupported) {
            wormholeRelayer.forwardToEvm{value: msg.value}(
                32,
                fromWormholeFormat(deliveryData.sourceAddress),
                bytes(""),
                Wei.wrap(0),
                Gas.wrap(REASONABLE_GAS_LIMIT),
                32,
                fromWormholeFormat(deliveryData.sourceAddress)
            );
        } else if (action == Action.ReentrantCall) {
            (uint256 deliveryPrice,) = wormholeRelayer.quoteEVMDeliveryPrice(
                deliveryData.sourceChainId, 0, REASONABLE_GAS_LIMIT
            );
            vm.recordLogs();
            wormholeRelayer.sendToEvm{value: deliveryPrice + wormhole.messageFee() + msg.value}(
                deliveryData.sourceChainId,
                fromWormholeFormat(deliveryData.sourceAddress),
                bytes(""),
                Wei.wrap(0),
                Gas.wrap(REASONABLE_GAS_LIMIT)
            );
            genericRelayer.relay(wormhole.chainId());
        } else {
            wormholeRelayer.forwardToEvm{value: msg.value}(
                deliveryData.sourceChainId,
                fromWormholeFormat(deliveryData.sourceAddress),
                bytes(""),
                Wei.wrap(0),
                Gas.wrap(REASONABLE_GAS_LIMIT),
                deliveryData.sourceChainId,
                fromWormholeFormat(deliveryData.sourceAddress)
            );
        }
    }

    function fromWormholeFormat(bytes32 whAddress) public returns (address addr) {
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
        wormholeRelayer.forwardToEvm{value: msg.value}(
            chainId,
            targetAddress,
            payload,
            Wei.wrap(receiverValue),
            Gas.wrap(gasLimit),
            chainId,
            targetAddress
        );
    }
    
}
