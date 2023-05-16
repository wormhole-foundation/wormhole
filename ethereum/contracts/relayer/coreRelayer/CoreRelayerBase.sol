// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {IWormhole} from "../../interfaces/IWormhole.sol";
import {IRelayProvider} from "../../interfaces/relayer/IRelayProvider.sol";
import {toWormholeFormat, min, pay, MAX_U128} from "./Utils.sol";
import {
  NoDeliveryInProgress,
  ReentrantDelivery,
  ForwardRequestFromWrongAddress,
  RelayProviderDoesNotSupportTargetChain,
  RelayProviderQuotedBogusAssetPrice,
  VaaKey,
  RelayProviderQuotedBogusGasPrice,
  Send,
  MaxTransactionFeeGreaterThanUint128,
  ReceiverValueGreaterThanUint128,
  TargetGasDeliveryAmountGreaterThanUint32,
  DeliveryInstruction,
  ExecutionParameters,
  IWormholeRelayerBase
} from "../../interfaces/relayer/IWormholeRelayer.sol";
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

  function publishAndPay(
    Wei wormholeMessageFee,
    Wei maxTransactionFee,
    Wei receiverValue,
    bytes memory encodedInstruction,
    uint8 consistencyLevel,
    IRelayProvider relayProvider
  ) internal returns (uint64 sequence) { 
    sequence =
      getWormhole().publishMessage{value: Wei.unwrap(wormholeMessageFee)}(0, encodedInstruction, consistencyLevel);

    emit SendEvent(sequence, Wei.unwrap(maxTransactionFee), Wei.unwrap(receiverValue));

    Wei amount;
    unchecked {amount = maxTransactionFee + receiverValue;}
    //TODO AMO: what if pay fails? (i.e. returns false)
    pay(relayProvider.getRewardAddress(), amount);
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

  // ----------------------------------------- Conversion ------------------------------------------

  function constructSend(
    uint16 targetChainId,
    bytes32 targetAddress,
    uint16 refundChainId,
    bytes32 refundAddress,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory payload,
    VaaKey[] memory vaaKeys,
    uint8 consistencyLevel,
    address relayProviderAddress,
    bytes memory relayParameters
  ) internal pure returns (Send memory) {
    if (maxTransactionFee > type(uint128).max )
      revert MaxTransactionFeeGreaterThanUint128();
    if ( receiverValue > type(uint128).max)
      revert ReceiverValueGreaterThanUint128();

    return Send(
      targetChainId,
      targetAddress,
      refundChainId,
      refundAddress,
      Wei.wrap(maxTransactionFee), // todo: use intoWei
      Wei.wrap(receiverValue),
      payload,
      vaaKeys,
      consistencyLevel,
      relayProviderAddress,
      relayParameters
    );
  }

  /** 
   * Calculate how much gas the relay provider can pay for on `sendParams.targetChain` using
   *   `sendParams.newTransactionFee`, and calculate how much value the relay provider will pass
   *   into `sendParams.targetAddress`.
   */
  function convertSendToDeliveryInstruction(
    Send memory send
  ) internal view returns (DeliveryInstruction memory instruction, IRelayProvider relayProvider) {
    relayProvider = IRelayProvider(send.relayProviderAddress);

    (Wei maximumRefundTarget, Wei receiverValueTarget, Gas gasLimit) =
      calculateTargetParams(
        send.targetChainId, send.maxTransactionFee, send.receiverValue, relayProvider
      );

    instruction = DeliveryInstruction({
      targetChainId: send.targetChainId,
      targetAddress: send.targetAddress,
      refundChainId: send.refundChainId,
      refundAddress: send.refundAddress,
      maximumRefundTarget: maximumRefundTarget,
      receiverValueTarget: receiverValueTarget,
      senderAddress: toWormholeFormat(msg.sender),
      sourceRelayProvider: toWormholeFormat(send.relayProviderAddress),
      targetRelayProvider: relayProvider.getTargetChainAddress(send.targetChainId),
      vaaKeys: send.vaaKeys,
      consistencyLevel: send.consistencyLevel,
      payload: send.payload,
      executionParameters: ExecutionParameters({
        gasLimit: gasLimit
      })
    });
  }

  function calculateTargetParams(
    uint16 targetChainId,
    Wei maxTransactionFee,
    Wei receiverValue,
    IRelayProvider provider
  ) internal view returns (
    Wei maximumRefundTarget,
    Wei receiverValueTarget,
    Gas gasLimit
  ) {
    if (!provider.isChainSupported(targetChainId))
      revert RelayProviderDoesNotSupportTargetChain(address(provider), targetChainId);

    (WeiPrice sourcePrice, WeiPrice targetPrice) =
      getAssetPricesWithBuffer(getChainId(), targetChainId, provider);

    receiverValueTarget = convertAmount(receiverValue, sourcePrice, targetPrice, false);

    Wei overhead = provider.quoteDeliveryOverhead(targetChainId);
    if (maxTransactionFee > overhead) { unchecked {
      Wei maxFeeSubOverhead = maxTransactionFee - overhead;

      maximumRefundTarget = convertAmount(maxFeeSubOverhead, sourcePrice, targetPrice, false);

      Gas gasLimitUncapped = maxFeeSubOverhead.toGas(getCheckedGasPriceSource(targetChainId, provider));
      gasLimit = gasLimitUncapped.min(Gas.wrap(type(uint32).max));
    }}
  }

  function getAssetPricesWithBuffer(
    uint16 sourceChainId,
    uint16 targetChainId,
    IRelayProvider provider
  ) internal view returns (WeiPrice sourcePrice, WeiPrice targetPrice)
  {
    //percentage = premium/base
    //e.g. premium = 5, base = 100 => 5 % premium, targetPrice is inflated by 5 % before conversion
    (uint16 premium, uint16 base) =
      provider.getAssetConversionBuffer(targetChainId);

    uint32 factor;
    unchecked {factor = uint32(base) + premium;}

    sourcePrice = getCheckedAssetPrice(sourceChainId, provider).mul(base);
    targetPrice = getCheckedAssetPrice(targetChainId, provider).mul(factor);
  }

  function getCheckedAssetPrice(
    uint16 chainId,
    IRelayProvider provider
  ) internal view returns (WeiPrice price) {
    price = provider.quoteAssetPrice(chainId);
    if (WeiPrice.unwrap(price) == 0)
      revert RelayProviderQuotedBogusAssetPrice(address(provider), chainId, WeiPrice.unwrap(price));
  }

  function getCheckedGasPriceSource(
    uint16 targetChainId,
    IRelayProvider provider
  ) internal view returns (GasPrice gasPriceSource) {
    gasPriceSource = provider.quoteGasPrice(targetChainId);
    if (gasPriceSource.unwrap() == 0)
      revert RelayProviderQuotedBogusGasPrice(address(provider), targetChainId, gasPriceSource.unwrap());
  }

  function convertAmount(
    Wei inputAmount,
    WeiPrice inputPrice,
    WeiPrice outputPrice,
    bool roundUp
  ) internal pure returns (Wei ouputAmount) {
    uint numerator = inputAmount.unwrap() * inputPrice.unwrap();
    uint denominator = outputPrice.unwrap();
    ouputAmount = Wei.wrap((roundUp ? (numerator + denominator - 1) : numerator) / denominator);
  }
}
