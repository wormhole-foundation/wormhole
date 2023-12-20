// SPDX-License-Identifier: Apache 2
pragma solidity >=0.6.12 <0.9.0;

import "wormhole-solidity-sdk/Utils.sol";

import "./Endpoint.sol";
import "../interfaces/IWormhole.sol";
import "./interfaces/IEndpointManager.sol";

contract WormholeEndpoint is Endpoint {
    // TODO -- fix this after some testing
    uint256 constant GAS_LIMIT = 500000;

    address immutable wormholeCoreBridge;
    address immutable wormholeRelayerAddr;
    uint256 immutable evmChainId;

    // Mapping of consumed VAAs
    mapping(bytes32 => bool) consumedVAAs;

    event ReceivedMessage(
        bytes32 digest,
        uint16 emitterChainId,
        bytes32 emitterAddress,
        uint64 sequence
    );

    error InvalidVaa(string reason);
    error InvalidSibling(uint16 chainId, bytes32 siblingAddress);
    error TransferAlreadyCompleted(bytes32 vaaHash);
    error InvalidFork(uint256 evmChainId, uint256 blockChainId);

    constructor(
        address _manager,
        address _wormholeCoreBridge,
        address _wormholeRelayerAddr,
        uint256 _evmChainId
    ) Endpoint(_manager) {
        wormholeCoreBridge = _wormholeCoreBridge;
        wormholeRelayerAddr = _wormholeRelayerAddr;
        evmChainId = _evmChainId;
    }

    function quoteDeliveryPrice(
        uint16 targetChain
    ) external view override returns (uint256 nativePriceQuote) {
        // no delivery fee for solana (standard relaying is not yet live)
        if (targetChain == 1) {
            return 0;
        }

        (uint256 cost, ) = wormholeRelayer().quoteEVMDeliveryPrice(
            targetChain,
            0,
            GAS_LIMIT
        );

        return cost;
    }

    function _sendMessage(
        uint16 recipientChain,
        bytes memory payload
    ) internal override {
        // do not use standard relaying for solana deliveries
        if (recipientChain == 1) {
            wormhole().publishMessage(0, payload, 1);
        } else {
            wormholeRelayer().sendPayloadToEvm{value: msg.value}(
                recipientChain,
                fromWormholeFormat(getSibling(recipientChain)),
                payload,
                0,
                GAS_LIMIT
            );
        }
    }

    function receiveMessage(bytes memory encodedMessage) external override {
        // verify VAA against Wormhole Core Bridge contract
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole()
            .parseAndVerifyVM(encodedMessage);

        // ensure that the VAA is valid
        if (!valid) {
            revert InvalidVaa(reason);
        }

        // ensure that the message came from a registered sibling contract
        if (!verifyBridgeVM(vm)) {
            revert InvalidSibling(vm.emitterChainId, vm.emitterAddress);
        }

        // save the VAA hash in storage to protect against replay attacks.
        if (isVAAConsumed(vm.hash)) {
            revert TransferAlreadyCompleted(vm.hash);
        }
        setVAAConsumed(vm.hash);

        // emit `ReceivedMessage` event
        emit ReceivedMessage(
            vm.hash,
            vm.emitterChainId,
            vm.emitterAddress,
            vm.sequence
        );

        // forward the VAA payload to the endpoint manager contract
        IEndpointManager(manager).attestationReceived(vm.payload);
    }

    function wormhole() public view returns (IWormhole) {
        return IWormhole(wormholeCoreBridge);
    }

    function wormholeRelayer() public view returns (IWormholeRelayer) {
        return IWormholeRelayer(wormholeRelayerAddr);
    }

    function verifyBridgeVM(
        IWormhole.VM memory vm
    ) internal view returns (bool) {
        if (isFork()) {
            revert InvalidFork(evmChainId, block.chainid);
        }
        return super.getSibling(vm.emitterChainId) == vm.emitterAddress;
    }

    function isFork() public view returns (bool) {
        return evmChainId != block.chainid;
    }

    function isVAAConsumed(bytes32 hash) public view returns (bool) {
        return consumedVAAs[hash];
    }

    function setVAAConsumed(bytes32 hash) internal {
        consumedVAAs[hash] = true;
    }
}
