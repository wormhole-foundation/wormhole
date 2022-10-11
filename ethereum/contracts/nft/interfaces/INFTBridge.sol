// contracts/NFTBridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../interfaces/IWormhole.sol";

interface INFTBridge {
    struct Transfer {
        bytes32 tokenAddress;
        uint16 tokenChain;
        bytes32 symbol;
        bytes32 name;
        uint256 tokenID;
        string uri;
        bytes32 to;
        uint16 toChain;
    }

    struct SPLCache {
        bytes32 name;
        bytes32 symbol;
    }

    function transferNFT(address token, uint256 tokenID, uint16 recipientChain, bytes32 recipient, uint32 nonce) external payable returns (uint64 sequence);

    function completeTransfer(bytes memory encodeVm) external;

    function encodeTransfer(Transfer memory transfer) external pure returns (bytes memory encoded);

    function parseTransfer(bytes memory encoded) external pure returns (Transfer memory transfer);

    function onERC721Received(address operator, address, uint256, bytes calldata) external view returns (bytes4);

    function isTransferCompleted(bytes32 hash) external view returns (bool);

    function wormhole() external view returns (IWormhole);

    function chainId() external view returns (uint16);

    function evmChainId() external view returns (uint256);

    function wrappedAsset(uint16 tokenChainId, bytes32 tokenAddress) external view returns (address);

    function isWrappedAsset(address token) external view returns (bool);

    function splCache(uint256 tokenId) external view returns (SPLCache memory);

    function finality() external view returns (uint8);
}