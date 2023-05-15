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

import "forge-std/console.sol";

abstract contract CoreRelayerBase is IWormholeRelayerBase {
  using WeiLib for Wei;
  using GasLib for Gas;

  //TODO AMO: see https://book.wormhole.com/wormhole/3_coreLayerContracts.html#consistency-levels
  //  15 is valid choice perhaps, but it feels messy... (seems like it should be 201?)
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
    (uint128 maxTransactionFee_, uint128 receiverValue_) = 
      checkFeesLessThanU128(maxTransactionFee, receiverValue);
    return Send(
      targetChainId,
      targetAddress,
      refundChainId,
      refundAddress,
      Wei.wrap(maxTransactionFee_), // todo: use intoWei
      Wei.wrap(receiverValue_),
      payload,
      vaaKeys,
      consistencyLevel,
      relayProviderAddress,
      relayParameters
    );
  }

  function checkFeesLessThanU128(
    uint256 maxTransactionFee,
    uint256 receiverValue
  ) internal pure returns (uint128 , uint128){
    if (maxTransactionFee > type(uint128).max )
      revert MaxTransactionFeeGreaterThanUint128();
    if ( receiverValue > type(uint128).max)
      revert ReceiverValueGreaterThanUint128();
    return (uint128(maxTransactionFee), uint128(receiverValue));
  }

  /** 
   * Calculate how much gas the relay provider can pay for on 'sendParams.targetChain' using
   *   'sendParams.newTransactionFee', and calculate how much value the relay provider will pass
   *   into 'sendParams.targetAddress'.
   */
  function convertSendToDeliveryInstruction(
    Send memory send
  ) internal view returns (DeliveryInstruction memory instruction) {
    IRelayProvider relayProvider = IRelayProvider(send.relayProviderAddress);

    instruction.targetChainId       = send.targetChainId;
    instruction.targetAddress       = send.targetAddress;
    instruction.refundChainId       = send.refundChainId;
    instruction.refundAddress       = send.refundAddress;
    instruction.maximumRefundTarget = calculateTargetDeliveryMaximumRefund(
                                        send.targetChainId, send.maxTransactionFee, relayProvider
                                      );
    instruction.receiverValueTarget = convertReceiverValueAmountToTarget(
                                        send.receiverValue, send.targetChainId, relayProvider
                                      );
    instruction.senderAddress       = toWormholeFormat(msg.sender);
    instruction.sourceRelayProvider = toWormholeFormat(send.relayProviderAddress);
    instruction.targetRelayProvider = relayProvider.getTargetChainAddress(send.targetChainId);
    instruction.vaaKeys             = send.vaaKeys;
    instruction.consistencyLevel    = send.consistencyLevel;
    instruction.payload             = send.payload;
    instruction.executionParameters = ExecutionParameters({
                                        gasLimit: calculateTargetGasDeliveryAmount(
                                          send.targetChainId, send.maxTransactionFee, relayProvider
                                        )
                                      });
  }

  /**
   * Given a targetChain, maxTransactionFee, and a relay provider, this function calculates what the
   *   maximum refund of the delivery transaction should be, in terms of target chain currency
   *
   * The maximum refund is the amount that would be refunded to refundAddress if the call to
   *   'receiveWormholeMessages' were to counterfactually take 0 gas.
   *
   * It does this by calculating (maxTransactionFee - deliveryOverhead) and converting (using the
   *   relay provider's prices) to target chain currency (where 'deliveryOverhead' is the
   *   relayProvider's base fee for delivering to targetChainId [in units of source chain currency])
   */
  function calculateTargetDeliveryMaximumRefund(
    uint16 targetChainId,
    Wei maxTransactionFee,
    IRelayProvider provider
  ) internal view returns (Wei maximumRefund) { unchecked {
    Wei overhead = provider.quoteDeliveryOverhead(targetChainId);
    if (maxTransactionFee <= overhead) 
      return Wei.wrap(0);

    (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChainId);
    uint32 numerator = uint32(denominator) + buffer;

    WeiPrice fromPrice = getCheckedAssetPrice(provider, getChainId());
    WeiPrice toPrice = getCheckedAssetPrice(provider, targetChainId);

    Wei remainder = maxTransactionFee - overhead;
    console.log("before convert asset calculate delivery max refund");
    return remainder.convertAsset(
      fromPrice, toPrice, denominator, numerator, false
    );
  }}

  /**
   * If the user specifies (for 'receiverValue) 'sourceAmount' of source chain currency, with relay
   *   provider 'provider', then this function calculates how much the relayer will pass into
   *   receiveWormholeMessages on the target chain (in target chain currency).
   *
   * The calculation simply converts this amount to target chain's currency, but also applies a
   *   multiplier of 'denominator/(denominator + buffer)' where these values are also specified
   *   by the relay provider 'provider'.
   */
  function convertReceiverValueAmountToTarget(
    Wei sourceAmount,
    uint16 targetChainId,
    IRelayProvider provider
  ) internal view returns (Wei targetAmount) { unchecked {
    (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChainId);
    uint32 numerator = uint32(denominator) + buffer;
    //multiplying with inverse of numerator/denominator, i.e. they are used flipped
    console.log("conert receiver value amount to target");
    targetAmount = assetConversionHelper(
      getChainId(), sourceAmount, targetChainId, denominator, numerator, false, provider
    );
  }}

  /**
   * Given a targetChainId, maxTransactionFee, and a relay provider, this function calculates what
   *   the gas limit of the delivery transaction should be.
   * It does this by calculating (maxTransactionFee - deliveryOverhead)/gasPrice where
   *  'deliveryOverhead' is the relayProvider's base fee for delivering to targetChain and
   *  'gasPrice' is the relayProvider's fee per unit of target chain gas.
   * Provider fees are quoted in units of the source chain currency.
   */
  function calculateTargetGasDeliveryAmount(
    uint16 targetChainId,
    Wei maxTransactionFee,
    IRelayProvider provider
  ) internal view returns (Gas gasAmount) { unchecked {
    Wei overhead = provider.quoteDeliveryOverhead(targetChainId);
    if (maxTransactionFee <= overhead) 
      return Gas.wrap(0);
    console.log("above to gas");
    uint256 gas = (maxTransactionFee - overhead).toGasU256(provider.quoteGasPrice(targetChainId));
    console.log("below to gas");
    return Gas.wrap(uint32(min(gas, uint256(type(uint32).max))));
  }}

  /**
   * Converts 'sourceAmount' of source chain currency to units of target chain currency using the
   *   prices of 'provider' and also multiplying by a specified fraction
   *   'multiplierNumerator/multiplierDenominator', rounding up or down specified by 'roundUp', and
   *   without performing intermediate rounding, i.e. the result should be as if float arithmetic
   *   was done and the rounding performed at the end
   */
  function assetConversionHelper(
    uint16 sourceChainId,
    Wei sourceAmount,
    uint16 targetChainId,
    uint32 multiplierNumerator,
    uint32 multiplierDenominator,
    bool roundUp,
    IRelayProvider provider
  ) internal view returns (Wei targetAmount) {
    //We probably call this multiple times during a single transaction, however since the relay
    //  provider contract is already hot at this point, calling it multiple times in a row
    //  shouldn't be too gas inefficient (about 100 gas per call) and local caching in storage
    //  would be ~equally expensive (SSTORE is 20k, but resetting to 0 refunds 19.9k, i.e. overall
    //  cost would also be ~100 gas).
    WeiPrice fromPrice = getCheckedAssetPrice(provider, sourceChainId);
    WeiPrice toPrice = getCheckedAssetPrice(provider, targetChainId);

    return sourceAmount.convertAsset(
      fromPrice, toPrice, multiplierNumerator, multiplierDenominator, roundUp
    );
  }

  function getCheckedAssetPrice(
    IRelayProvider provider,
    uint16 chainId
  ) internal view returns (WeiPrice price) {
    //if (chainId != getChainId())
    //  checkRelayProviderSupportsChain();
    price = provider.quoteAssetPrice(chainId);
    if (WeiPrice.unwrap(price) == 0)
      revert RelayProviderQuotedBogusAssetPrice(address(provider), chainId, WeiPrice.unwrap(price));
  }

  function checkRelayProviderSupportsChain(
    IRelayProvider relayProvider,
    uint16 targetChainId
  ) internal view {
    if (!relayProvider.isChainSupported(targetChainId))
      revert RelayProviderDoesNotSupportTargetChain(address(relayProvider), targetChainId);
  }
}
