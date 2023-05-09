// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.19;

import {IWormhole} from "../../interfaces/IWormhole.sol";
import {IRelayProvider} from "../../interfaces/relayer/IRelayProvider.sol";
import {toWormholeFormat, min, pay} from "./Utils.sol";
import {
  NoDeliveryInProgress,
  ReentrantDelivery,
  ForwardRequestFromWrongAddress,
  RelayProviderDoesNotSupportTargetChain,
  RelayProviderQuotedBogusAssetPrice,
  Send,
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

abstract contract CoreRelayerBase is IWormholeRelayerBase {
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

  function publishAndPay(
    uint256 wormholeMessageFee,
    uint256 maxTransactionFee,
    uint256 receiverValue,
    bytes memory message,
    uint8 consistencyLevel,
    IRelayProvider relayProvider
  ) internal returns (uint64 sequence) { 
    sequence =
      getWormhole().publishMessage{value: wormholeMessageFee}(0, message, consistencyLevel);

    emit SendEvent(sequence, maxTransactionFee, receiverValue);

    uint256 amount;
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
    uint256 maxTransactionFee,
    IRelayProvider provider
  ) internal view returns (uint256 maximumRefund) { unchecked {
    uint256 overhead = provider.quoteDeliveryOverhead(targetChainId);
    if (maxTransactionFee > overhead) { 
      (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChainId);
      uint256 remainder = maxTransactionFee - overhead;
      uint256 numerator = uint256(denominator) + buffer;
      maximumRefund = assetConversionHelper(
        getChainId(), remainder, targetChainId, denominator, numerator, false, provider
      );
    }
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
    uint256 sourceAmount,
    uint16 targetChainId,
    IRelayProvider provider
  ) internal view returns (uint256 targetAmount) { unchecked {
    (uint16 buffer, uint16 denominator) = provider.getAssetConversionBuffer(targetChainId);
    uint256 numerator = uint256(denominator) + buffer;
    //multiplying with inverse of numerator/denominator, i.e. they are used flipped
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
    uint256 maxTransactionFee,
    IRelayProvider provider
  ) internal view returns (uint32 gasAmount) { unchecked {
    uint256 overhead = provider.quoteDeliveryOverhead(targetChainId);
    if (maxTransactionFee > overhead)
      gasAmount = uint32(
        min(
          (maxTransactionFee - overhead) / provider.quoteGasPrice(targetChainId),
          type(uint32).max
        )
      );
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
    uint256 sourceAmount,
    uint16 targetChainId,
    uint256 multiplierNumerator,
    uint256 multiplierDenominator,
    bool roundUp,
    IRelayProvider provider
  ) internal view returns (uint256 targetAmount) {
    //We probably call this multiple times during a single transaction, however since the relay
    //  provider contract is already hot at this point, calling it multiple times in a row
    //  shouldn't be too gas inefficient (about 100 gas per call) and local caching in storage
    //  would be ~equally expensive (SSTORE is 20k, but resetting to 0 refunds 19.9k, i.e. overall
    //  cost would also be ~100 gas).
    uint256 sourceNativeCurrencyPrice = getCheckedAssetPrice(provider, sourceChainId);
    uint256 targetNativeCurrencyPrice = getCheckedAssetPrice(provider, targetChainId);

    uint256 numerator = sourceAmount * sourceNativeCurrencyPrice * multiplierNumerator;
    uint256 denominator = targetNativeCurrencyPrice * multiplierDenominator;
    targetAmount = (roundUp ? (numerator + denominator - 1) : numerator) / denominator;
  }

  function getCheckedAssetPrice(
    IRelayProvider provider,
    uint16 chainId
  ) internal view returns (uint256 price) {
    //if (chainId != getChainId())
    //  checkRelayProviderSupportsChain();
    price = provider.quoteAssetPrice(chainId);
    if (price == 0)
      revert RelayProviderQuotedBogusAssetPrice(address(provider), chainId, price);
  }

  function checkRelayProviderSupportsChain(
    IRelayProvider relayProvider,
    uint16 targetChainId
  ) internal view {
    if (!relayProvider.isChainSupported(targetChainId))
      revert RelayProviderDoesNotSupportTargetChain(address(relayProvider), targetChainId);
  }
}
