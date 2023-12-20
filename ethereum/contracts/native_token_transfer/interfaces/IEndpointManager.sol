// SPDX-License-Identifier: Apache 2
pragma solidity >=0.6.12 <0.9.0;

interface IEndpointManager {
    error CallerNotEndpoint(address caller);
    error DeliveryPaymentTooLow(
        uint256 requiredPayment,
        uint256 providedPayment
    );
    error SequenceAttestationAlreadyReceived(uint64 sequence, address endpoint);
    error UnexpectedEndpointManagerMessageType(uint8 msgType);
    error InvalidTargetChain(uint16 targetChain, uint16 thisChain);
    error InvalidEndpointZeroAddress();
    error AlreadyRegisteredEndpoint(address endpoint);
    error NonRegisteredEndpoint(address endpoint);
    error InvalidFork(uint256 evmChainId, uint256 blockChainId);

    event EndpointAdded(address endpoint);
    event EndpointRemoved(address endpoint);

    function transfer(
        uint256 amount,
        uint16 recipientChain,
        bytes32 recipient
    ) external payable returns (uint64 msgId);

    function attestationReceived(bytes memory payload) external;

    function getThreshold() external view returns (uint8);

    function getEndpoints() external view returns (address[] memory);

    function nextSequence() external view returns (uint64);
}
