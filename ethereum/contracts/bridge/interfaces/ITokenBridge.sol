// contracts/Bridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

interface ITokenBridge {
    struct Transfer {
        uint8 payloadID;
        uint256 amount;
        bytes32 tokenAddress;
        uint16 tokenChain;
        bytes32 to;
        uint16 toChain;
        uint256 fee;
    }

    struct TransferWithPayload {
        uint8 payloadID;
        uint256 amount;
        bytes32 tokenAddress;
        uint16 tokenChain;
        bytes32 to;
        uint16 toChain;
        bytes32 fromAddress;
        bytes payload;
    }

    struct AssetMeta {
        uint8 payloadID;
        bytes32 tokenAddress;
        uint16 tokenChain;
        uint8 decimals;
        bytes32 symbol;
        bytes32 name;
    }

    function attestToken(address tokenAddress, uint32 nonce) external payable returns (uint64 sequence);

    function wrapAndTransferETH(uint16 recipientChain, bytes32 recipient, uint256 arbiterFee, uint32 nonce) external payable returns (uint64 sequence);

    function wrapAndTransferETHWithPayload(uint16 recipientChain, bytes32 recipient, uint32 nonce, bytes memory payload) external payable returns (uint64 sequence);

    function transferTokens(address token, uint256 amount, uint16 recipientChain, bytes32 recipient, uint256 arbiterFee, uint32 nonce) external payable returns (uint64 sequence);

    function transferTokensWithPayload(address token, uint256 amount, uint16 recipientChain, bytes32 recipient, uint32 nonce, bytes memory payload) external payable returns (uint64 sequence);

    function updateWrapped(bytes memory encodedVm) external returns (address token);

    function createWrapped(bytes memory encodedVm) external returns (address token);

    function completeTransferWithPayload(bytes memory encodedVm) external returns (bytes memory);

    function completeTransferAndUnwrapETHWithPayload(bytes memory encodedVm) external returns (bytes memory);

    function completeTransfer(bytes memory encodedVm) external;

    function completeTransferAndUnwrapETH(bytes memory encodedVm) external;

    function encodeAssetMeta(AssetMeta memory meta) external pure returns (bytes memory encoded);

    function encodeTransfer(Transfer memory transfer) external pure returns (bytes memory encoded);

    function encodeTransferWithPayload(TransferWithPayload memory transfer) external pure returns (bytes memory encoded);

    function parsePayloadID(bytes memory encoded) external pure returns (uint8 payloadID);

    function parseAssetMeta(bytes memory encoded) external pure returns (AssetMeta memory meta);

    function parseTransfer(bytes memory encoded) external pure returns (Transfer memory transfer);

    function parseTransferWithPayload(bytes memory encoded) external pure returns (TransferWithPayload memory transfer);

    function isTransferCompleted(bytes32 hash) external view returns (bool);

    function chainId() external view returns (uint16);

    function evmChainId() external view returns (uint256);

    function wrappedAsset(uint16 tokenChainId, bytes32 tokenAddress) external view returns (address);

    function outstandingBridged(address token) external view returns (uint256);

    function isWrappedAsset(address token) external view returns (bool);

    function finality() external view returns (uint8);
}