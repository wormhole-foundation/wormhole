// SPDX-License-Identifier: Apache 2
pragma solidity >=0.6.12 <0.9.0;

interface IEndpoint {
    error CallerNotManager(address caller);
    error InvalidSiblingZeroAddress();

    function sendMessage(
        uint16 recipientChain,
        bytes memory payload
    ) external payable;

    function receiveMessage(bytes memory encodedMessage) external;

    function quoteDeliveryPrice(
        uint16 targetChain
    ) external view returns (uint256 nativePriceQuote);

    function getSibling(uint16 chainId) external view returns (bytes32);
}
