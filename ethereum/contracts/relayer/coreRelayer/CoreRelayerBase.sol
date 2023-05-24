// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {IWormhole} from "../../interfaces/IWormhole.sol";
import {IRelayProvider} from "../../interfaces/relayer/IRelayProvider.sol";
import {toWormholeFormat, min, pay} from "../../libraries/relayer/Utils.sol";
import {
  NoDeliveryInProgress,
  ReentrantDelivery,
  ForwardRequestFromWrongAddress,
  RelayProviderDoesNotSupportTargetChain,
  VaaKey,
  InvalidMsgValue,
  IWormholeRelayerBase
} from "../../interfaces/relayer/IWormholeRelayer.sol";
import {DeliveryInstruction} from "../../libraries/relayer/RelayerInternalStructs.sol";
import {
  ForwardInstruction,
  DeliveryTmpState,
  getDeliveryTmpState,
  getRegisteredCoreRelayersState
} from "./CoreRelayerStorage.sol";
import "../../interfaces/relayer/TypedUnits.sol";

abstract contract CoreRelayerBase is IWormholeRelayerBase {
  using WeiLib for Wei;
  using GasLib for Gas;
  using WeiPriceLib for WeiPrice;
  using GasPriceLib for GasPrice;

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

  function getRegisteredCoreRelayerContract(uint16 chainId) public view returns (bytes32) {
    return getRegisteredCoreRelayersState().registeredCoreRelayers[chainId];
  }

  //Our get functions require view instead of pure (despite not actually reading storage) because
  //  they can't be evaluated at compile time. (https://ethereum.stackexchange.com/a/120630/103366)
  
  function getWormhole() internal view returns (IWormhole) {
    return wormhole_;
  }

  function getChainId() internal view returns (uint16) {
    return chainId_;
  }

  function getWormholeMessageFee() internal view returns (Wei) {
    return Wei.wrap(getWormhole().messageFee());
  }

  function msgValue() internal view returns (Wei) {
    return Wei.wrap(msg.value);
  }

  function checkMsgValue(Wei wormholeMessageFee, Wei deliveryPrice, Wei paymentForExtraReceiverValue) internal view {
    if(msgValue() != deliveryPrice + paymentForExtraReceiverValue + wormholeMessageFee) 
      revert InvalidMsgValue(msgValue(), deliveryPrice + paymentForExtraReceiverValue + wormholeMessageFee);
  }
  
  function publishAndPay(
    Wei wormholeMessageFee,
    Wei deliveryQuote,
    Wei paymentForExtraReceiverValue,
    bytes memory encodedInstruction,
    uint8 consistencyLevel,
    IRelayProvider relayProvider
  ) internal returns (uint64 sequence) { 

    sequence =
      getWormhole().publishMessage{value: wormholeMessageFee.unwrap()}(0, encodedInstruction, consistencyLevel);

    //TODO AMO: what if pay fails? (i.e. returns false)
    pay(relayProvider.getRewardAddress(), deliveryQuote + paymentForExtraReceiverValue);

    emit SendEvent(sequence, deliveryQuote, paymentForExtraReceiverValue);
  }

  // ----------------------- delivery transaction temorary storage functions -----------------------

  function startDelivery(address targetAddress) internal {
    DeliveryTmpState storage state = getDeliveryTmpState();
    if (state.deliveryInProgress)
      revert ReentrantDelivery(msg.sender, state.deliveryTarget);

    state.deliveryInProgress = true;
    state.deliveryTarget = targetAddress;
  }

  function finishDelivery() internal {
    DeliveryTmpState storage state = getDeliveryTmpState();
    state.deliveryInProgress = false;
    state.deliveryTarget = address(0);
    delete state.forwardInstructions;
  }

  function appendForwardInstruction(ForwardInstruction memory forwardInstruction) internal {
    getDeliveryTmpState().forwardInstructions.push(forwardInstruction);
  }

  function getForwardInstructions() internal view returns (ForwardInstruction[] storage) {
    return getDeliveryTmpState().forwardInstructions;
  }

  function checkMsgSenderInDelivery() internal view {
    DeliveryTmpState storage state = getDeliveryTmpState();
    if (!state.deliveryInProgress)
      revert NoDeliveryInProgress();
    
    if (msg.sender != state.deliveryTarget)
      revert ForwardRequestFromWrongAddress(msg.sender, state.deliveryTarget);
  }

}
