// contracts/NFTBridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./NFTBridgeStructs.sol";

interface INFTBridge {

    function transferNFT(address token, uint256 tokenID, uint16 recipientChain, bytes32 recipient, uint32 nonce) external payable returns (uint64 sequence);

    function completeTransfer(bytes memory encodeVm) external;

    function encodeTransfer(NFTBridgeStructs.Transfer memory transfer) external pure returns (bytes memory encoded);

    function parseTransfer(bytes memory encoded) external pure returns (NFTBridgeStructs.Transfer memory transfer);

    function onERC721Received(address operator, address, uint256, bytes calldata) external view returns (bytes4);

    function isTransferCompleted(bytes32 hash) external view returns (bool);

    function wrappedAsset(uint16 tokenChainId, bytes32 tokenAddress) external view returns (address);

}