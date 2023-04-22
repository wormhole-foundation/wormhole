// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

interface IWormholeReceiver {

     /**
     * @notice This function is invoked by the WormholeDelivery contract when a 'send' or 'forward' is performed with this contract as the target.
     * 
     * NOTE: Restrict this function's access to the WormholeDelivery address for the deployed chain to avoid bypassing delivery logic and disrupting refund and forwarding mechanisms.
     * 
     * Deliveries should be expected to be performed at least once, and potentially multiple times.
     * 
     * The msg.value for this call will be equal to the receiverValue specified in the send request.
     *
     * A ReceiverFailure will occur if the interface is improperly implemented, reverts, or exceeds the specified gasLimit (maxTransactionFee).
     * 
     * @param deliveryData - This struct contains information about the delivery which is being performed.
     * - sourceAddress - the (wormhole format) address on the sending chain which requested this delivery. Any address is able to initiate a delivery to anywhere else.
     * - sourceChain - the wormhole chain ID where this delivery was requested.
     * - maximumRefund - the maximum refund that can be awarded at the end of this delivery, assuming no gas is used by receiveWormholeMessages.
     * - deliveryHash - the VAA hash of the deliveryVAA. Store this hash in state for replay protection if you don't want to process this delivery multiple times.
     * - payload - an optional arbitrary message included in the delivery by the requester.
     * @param signedVaas - Additional VAAs requested to be included in this delivery. They are guaranteed to be included in the same order as specified in the delivery request.
     * NOTE: These signedVaas are NOT verified by the Wormhole core contract prior to being provided to this call. 
     * Ensure parseAndVerify is called on the Wormhole core contract before trusting the content of a raw VAA to prevent using invalid or malicious VAAs.
     */
    function receiveWormholeMessages(DeliveryData memory deliveryData, bytes[] memory signedVaas) external payable;


    /**
     * @notice DeliveryData is the struct passed into the 'deliver' function containing information about the delivery being performed.
     *
     * @custom:member sourceAddress - The wormhole format address on the sending chain that requested this delivery. Any address can initiate a delivery to any other address.
     * @custom:member sourceChain - The wormhole chain ID where this delivery was requested.
     * @custom:member maximumRefund - The maximum refund that can be awarded at the end of this delivery, assuming no gas is used by receiveWormholeMessages.
     * @custom:member deliveryHash - The VAA hash of the deliveryVAA. Store this hash in state for replay protection if you don't want to process this delivery multiple times.
     * @custom:member payload - An optional arbitrary message included in the delivery by the requester.
     */
    struct DeliveryData {
        bytes32 sourceAddress;
        uint16 sourceChain;
        uint256 maximumRefund;
        bytes32 deliveryHash;
        bytes payload;
    }
}
