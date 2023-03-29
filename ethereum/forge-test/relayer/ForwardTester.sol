// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.0;

import "../contracts/interfaces/IWormhole.sol";
import "../contracts/interfaces/IWormholeReceiver.sol";
import "../contracts/interfaces/IWormholeRelayer.sol";
import "../contracts/interfaces/IRelayProvider.sol";
import "../contracts/libraries/external/BytesLib.sol";
import "./MockGenericRelayer.sol";
import "forge-std/console.sol";
import "forge-std/Vm.sol";

contract ForwardTester is IWormholeReceiver {
    using BytesLib for bytes;

    IWormhole wormhole;
    IWormholeRelayer wormholeRelayer;
    MockGenericRelayer genericRelayer;

    address private constant VM_ADDRESS = address(bytes20(uint160(uint256(keccak256("hevm cheat code")))));

    Vm public constant vm = Vm(VM_ADDRESS);

    constructor(address _wormhole, address _wormholeRelayer, address _wormholeSimulator) {
        wormhole = IWormhole(_wormhole);
        wormholeRelayer = IWormholeRelayer(_wormholeRelayer);
        genericRelayer = new MockGenericRelayer(_wormhole, _wormholeSimulator, _wormholeRelayer);
        genericRelayer.setWormholeRelayerContract(wormhole.chainId(), address(wormholeRelayer));
        genericRelayer.setProviderDeliveryAddress(
            wormhole.chainId(),
            wormholeRelayer.fromWormholeFormat(
                IRelayProvider(wormholeRelayer.getDefaultRelayProvider()).getDeliveryAddress(wormhole.chainId())
            )
        );
        genericRelayer.setWormholeFee(wormhole.chainId(), wormhole.messageFee());
    }

    enum Action {
        MultipleForwardsRequested,
        ForwardRequestFromWrongAddress,
        MultichainSendEmpty,
        MaxTransactionFeeNotEnough,
        FundsTooMuch,
        ReentrantCall,
        WorksCorrectly
    }

    function receiveWormholeMessages(bytes[] memory vaas, bytes[] memory additionalData) public payable override {
        (IWormhole.VM memory vaa, bool valid, string memory reason) = wormhole.parseAndVerifyVM(vaas[0]);
        require(valid, reason);

        bytes memory payload = vaa.payload;
        Action action = Action(payload.toUint8(0));

        IWormholeRelayer.MessageInfo[] memory empty = new IWormholeRelayer.MessageInfo[](0);

        if (action == Action.MultipleForwardsRequested) {
            uint256 maxTransactionFee =
                wormholeRelayer.quoteGas(vaa.emitterChainId, 10000, wormholeRelayer.getDefaultRelayProvider());
            wormholeRelayer.forward(
                vaa.emitterChainId, vaa.emitterAddress, vaa.emitterAddress, maxTransactionFee, 0, empty
            );
            wormholeRelayer.forward(
                vaa.emitterChainId, vaa.emitterAddress, vaa.emitterAddress, maxTransactionFee, 0, empty
            );
        } else if (action == Action.ForwardRequestFromWrongAddress) {
            // Emitter must be a wormhole relayer
            uint256 maxTransactionFee =
                wormholeRelayer.quoteGas(vaa.emitterChainId, 10000, wormholeRelayer.getDefaultRelayProvider());
            DummyContract dc = new DummyContract(address(wormholeRelayer));
            dc.forward(vaa.emitterChainId, vaa.emitterAddress, vaa.emitterAddress, maxTransactionFee, 0, empty);
        } else if (action == Action.MultichainSendEmpty) {
            wormholeRelayer.multichainForward(
                IWormholeRelayer.MultichainSend(
                    wormholeRelayer.getDefaultRelayProvider(), empty, new IWormholeRelayer.Send[](0)
                )
            );
        } else if (action == Action.MaxTransactionFeeNotEnough) {
            uint256 maxTransactionFee =
                wormholeRelayer.quoteGas(vaa.emitterChainId, 1, wormholeRelayer.getDefaultRelayProvider()) - 1;
            wormholeRelayer.forward(
                vaa.emitterChainId, vaa.emitterAddress, vaa.emitterAddress, maxTransactionFee, 0, empty
            );
        } else if (action == Action.FundsTooMuch) {
            // set maximum budget to less than this
            uint256 maxTransactionFee =
                wormholeRelayer.quoteGas(vaa.emitterChainId, 10000, wormholeRelayer.getDefaultRelayProvider());
            wormholeRelayer.forward(
                vaa.emitterChainId, vaa.emitterAddress, vaa.emitterAddress, maxTransactionFee, 0, empty
            );
        } else if (action == Action.ReentrantCall) {
            uint256 maxTransactionFee =
                wormholeRelayer.quoteGas(wormhole.chainId(), 10000, wormholeRelayer.getDefaultRelayProvider());
            vm.recordLogs();
            wormholeRelayer.send{value: maxTransactionFee + wormhole.messageFee()}(
                wormhole.chainId(), vaa.emitterAddress, vaa.emitterAddress, maxTransactionFee, 0, empty
            );
            genericRelayer.relay(wormhole.chainId());
        } else {
            uint256 maxTransactionFee =
                wormholeRelayer.quoteGas(vaa.emitterChainId, 10000, wormholeRelayer.getDefaultRelayProvider());
            wormholeRelayer.forward(
                vaa.emitterChainId, vaa.emitterAddress, vaa.emitterAddress, maxTransactionFee, 0, empty
            );
        }
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
        bytes32 targetAddress,
        bytes32 refundAddress,
        uint256 maxTransactionFee,
        uint256 receiverValue,
        IWormholeRelayer.MessageInfo[] memory messages
    ) public {
        wormholeRelayer.forward(chainId, targetAddress, refundAddress, maxTransactionFee, receiverValue, messages);
    }
}
