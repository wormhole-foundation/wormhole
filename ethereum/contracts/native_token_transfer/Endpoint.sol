// SPDX-License-Identifier: Apache 2
pragma solidity >=0.6.12 <0.9.0;

import "@openzeppelin/contracts/access/Ownable.sol";

import "./interfaces/IEndpoint.sol";

abstract contract Endpoint is IEndpoint, Ownable {
    /// updating bridgeManager requires a new Endpoint deployment.
    /// Projects should implement their own governance to remove the old Endpoint contract address and then add the new one.
    address immutable manager;
    // Mapping of siblings on other chains
    mapping(uint16 => bytes32) siblings;
    // TODO -- Add state to prevent messages from being double-submitted. Could be VAA hash and Axelar equivalent? Or could hash the entire EndpointMessage (but need unique fields like blocknum and timestamp then).

    modifier onlyManager() {
        if (msg.sender != manager) {
            revert CallerNotManager(msg.sender);
        }
        _;
    }

    constructor(address _manager) {
        manager = _manager;
    }

    /// @notice Called by the BridgeManager contract to send a cross-chain message.
    function sendMessage(
        uint16 recipientChain,
        bytes memory payload
    ) external payable onlyManager {
        _sendMessage(recipientChain, payload);
    }

    function _sendMessage(
        uint16 recipientChain,
        bytes memory payload
    ) internal virtual;

    /// @notice Receive an attested message from the verification layer
    ///         This function should verify the encodedVm and then call attestationReceived on the bridge manager contract.
    function receiveMessage(bytes memory encodedMessage) external virtual;

    function quoteDeliveryPrice(
        uint16 targetChain
    ) external view virtual returns (uint256 nativePriceQuote);

    /// @notice Get the corresponding Endpoint contract on other chains that have been registered via governance.
    ///         This design should be extendable to other chains, so each Endpoint would be potentially concerned with Endpoints on multiple other chains
    ///         Note that siblings are registered under wormhole chainID values
    function getSibling(uint16 chainId) public view returns (bytes32) {
        return siblings[chainId];
    }

    function setSibling(
        uint16 chainId,
        bytes32 bridgeContract
    ) external onlyOwner {
        if (bridgeContract == bytes32(0)) {
            revert InvalidSiblingZeroAddress();
        }
        siblings[chainId] = bridgeContract;
    }
}
