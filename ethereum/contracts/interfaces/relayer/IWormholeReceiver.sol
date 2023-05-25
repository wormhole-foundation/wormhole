// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

/**
 * @notice DeliveryData is the struct that the relay provider passes into `deliver()` containing an
 *     array of the signed wormhole messages that are to be relayed
 *
 * @custom:member sourceAddress - the (wormhole format) address on the sending chain which requested
 *     this delivery.
 * @custom:member sourceChainId - the wormhole chain ID where this delivery was requested.
 * @custom:member maximumRefund - the maximum refund that can possibly be awarded at the end of this
 *     delivery, assuming no gas is used by receiveWormholeMessages.
 * @custom:member deliveryHash - the VAA hash of the deliveryVAA. 
 * @custom:member payload - an arbitrary message which was included in the delivery by the
 *     requester.
 */
struct DeliveryData {
  bytes32 sourceAddress;
  uint16 sourceChainId;
  uint256 targetChainRefundPerGasUnused;
  bytes32 deliveryHash;
  bytes payload;
}

interface IWormholeReceiver {
   /**
     * @notice When a `send` is performed with this contract as the target, this function will be
     *     invoked.
     *   To get the address that will invoke this contract, call the `getDeliveryAddress()` function
     *     on this chain (the target chain)'s WormholeRelayer contract
     *
     * NOTE: This function should be restricted such that only `getDeliveryAddress()` can call it.
     *
     * We also recommend that this function:
     *   - Stores all received `deliveryData.deliveryHash`s in a mapping `(bytes32 => bool)`, and
     *       on every call, checks that deliveryData.deliveryHash has not already been stored in the
     *       map (This is to prevent other users maliciously trying to relay the same message)
     *   - Checks that `deliveryData.sourceChain` and `deliveryData.sourceAddress` are indeed who
     *       you expect to have requested the calling of `send` or `forward` on the source chain
     *
     * The invocation of this function corresponding to the `send` request will have msg.value equal
     *   to the receiverValue specified in the send request.
     *
     * If the invocation of this function reverts or exceeds the gas limit (`maxTransactionFee`)
     *   specified by the send requester, this delivery will result in a `ReceiverFailure`.
     *
     * @param deliveryData - This struct contains information about the delivery which is being
     *     performed
     * @param signedVaas - Additional VAAs which were requested to be included in this delivery.
     *   They are guaranteed to all be included and in the same order as was specified in the
     *     delivery request.
     * NOTE: These signedVaas are NOT verified by the Wormhole core contract prior to being provided
     *     to this call. Always make sure `parseAndVerify()` is called on the Wormhole core contract
     *     before trusting the content of a raw VAA, otherwise the VAA may be invalid or malicious.
     */
  function receiveWormholeMessages(
    DeliveryData memory deliveryData,
    bytes[] memory signedVaas
  ) external payable;
}
