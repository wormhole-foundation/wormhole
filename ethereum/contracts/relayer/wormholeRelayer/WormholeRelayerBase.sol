// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {IWormhole} from "../../interfaces/IWormhole.sol";
import {IDeliveryProvider} from "../../interfaces/relayer/IDeliveryProviderTyped.sol";
import {toWormholeFormat, min, pay} from "../../relayer/libraries/Utils.sol";
import {
    ReentrantDelivery,
    DeliveryProviderDoesNotSupportTargetChain,
    VaaKey,
    InvalidMsgValue,
    IWormholeRelayerBase
} from "../../interfaces/relayer/IWormholeRelayerTyped.sol";
import {DeliveryInstruction} from "../../relayer/libraries/RelayerInternalStructs.sol";
import {
    DeliveryTmpState,
    getDeliveryTmpState,
    getDeliverySuccessState,
    getDeliveryFailureState,
    getRegisteredWormholeRelayersState
} from "./WormholeRelayerStorage.sol";
import "../../interfaces/relayer/TypedUnits.sol";

abstract contract WormholeRelayerBase is IWormholeRelayerBase {
    using WeiLib for Wei;
    using GasLib for Gas;
    using WeiPriceLib for WeiPrice;
    using GasPriceLib for GasPrice;
    using LocalNativeLib for LocalNative;

    //see https://book.wormhole.com/wormhole/3_coreLayerContracts.html#consistency-levels
    //  15 is valid choice for now but ultimately we want something more canonical (202?)
    //  Also, these values should definitely not be defined here but should be provided by IWormhole!
    uint8 internal constant CONSISTENCY_LEVEL_FINALIZED = 15;
    uint8 internal constant CONSISTENCY_LEVEL_INSTANT = 200;

    IWormhole private immutable wormhole_;
    uint16 private immutable chainId_;

    constructor(address _wormhole) {
        wormhole_ = IWormhole(_wormhole);
        chainId_ = uint16(wormhole_.chainId());
    }

    function getRegisteredWormholeRelayerContract(uint16 chainId) public view returns (bytes32) {
        return getRegisteredWormholeRelayersState().registeredWormholeRelayers[chainId];
    }

    function deliveryAttempted(bytes32 deliveryHash) public view returns (bool attempted) {
        return getDeliverySuccessState().deliverySuccessBlock[deliveryHash] != 0 ||
            getDeliveryFailureState().deliveryFailureBlock[deliveryHash] != 0;
    }

    function deliverySuccessBlock(bytes32 deliveryHash) public view returns (uint256 blockNumber) {
        return getDeliverySuccessState().deliverySuccessBlock[deliveryHash];
    }

    function deliveryFailureBlock(bytes32 deliveryHash) public view returns (uint256 blockNumber) {
        return getDeliveryFailureState().deliveryFailureBlock[deliveryHash];
    }

    //Our get functions require view instead of pure (despite not actually reading storage) because
    //  they can't be evaluated at compile time. (https://ethereum.stackexchange.com/a/120630/103366)

    function getWormhole() internal view returns (IWormhole) {
        return wormhole_;
    }

    function getChainId() internal view returns (uint16) {
        return chainId_;
    }

    function getWormholeMessageFee() internal view returns (LocalNative) {
        return LocalNative.wrap(getWormhole().messageFee());
    }

    function msgValue() internal view returns (LocalNative) {
        return LocalNative.wrap(msg.value);
    }

    function checkMsgValue(
        LocalNative wormholeMessageFee,
        LocalNative deliveryPrice,
        LocalNative paymentForExtraReceiverValue
    ) internal view {
        if (msgValue() != deliveryPrice + paymentForExtraReceiverValue + wormholeMessageFee) {
            revert InvalidMsgValue(
                msgValue(), deliveryPrice + paymentForExtraReceiverValue + wormholeMessageFee
            );
        }
    }

    function publishAndPay(
        LocalNative wormholeMessageFee,
        LocalNative deliveryQuote,
        LocalNative paymentForExtraReceiverValue,
        bytes memory encodedInstruction,
        uint8 consistencyLevel,
        address payable rewardAddress
    ) internal returns (uint64 sequence, bool paymentSucceeded) {
        sequence = getWormhole().publishMessage{value: wormholeMessageFee.unwrap()}(
            0, encodedInstruction, consistencyLevel
        );

        paymentSucceeded = pay(
            rewardAddress,
            deliveryQuote + paymentForExtraReceiverValue
        );

        emit SendEvent(sequence, deliveryQuote, paymentForExtraReceiverValue);
    }

     // ----------------------- delivery transaction temorary storage functions -----------------------

    function startDelivery(address targetAddress, address deliveryProvider, uint16 refundChain, bytes32 refundAddress) internal {
        DeliveryTmpState storage state = getDeliveryTmpState();
        if (state.deliveryInProgress) {
            revert ReentrantDelivery(msg.sender, state.deliveryTarget);
        }

        state.deliveryInProgress = true;
        state.deliveryTarget = targetAddress;
        state.deliveryProvider = deliveryProvider;
        state.refundChain = refundChain;
        state.refundAddress = refundAddress;
    }

    function finishDelivery() internal {
        DeliveryTmpState storage state = getDeliveryTmpState();
        state.deliveryInProgress = false;
        state.deliveryTarget = address(0);
        state.deliveryProvider = address(0);
        state.refundChain = 0;
        state.refundAddress = bytes32(0);
    }

    function getOriginalDeliveryProvider() internal view returns (address) {
        return getDeliveryTmpState().deliveryProvider;
    }

    function getCurrentRefundChain() internal view returns (uint16) {
        return getDeliveryTmpState().refundChain;
    }

    function getCurrentRefundAddress() internal view returns (bytes32) {
        return getDeliveryTmpState().refundAddress;
    }
}
