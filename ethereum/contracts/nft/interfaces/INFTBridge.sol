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

     struct RegisterChain {
        bytes32 module;
        uint8 action;
        uint16 chainId;

        uint16 emitterChainID;
        bytes32 emitterAddress;
    }

    struct UpgradeContract {
        bytes32 module;
        uint8 action;
        uint16 chainId;

        bytes32 newContract;
    }

    struct RecoverChainId {
        bytes32 module;
        uint8 action;

        uint256 evmChainId;
        uint16 newChainId;
    }

    event ContractUpgraded(address indexed oldContract, address indexed newContract);

    function transferNFT(address token, uint256 tokenID, uint16 recipientChain, bytes32 recipient, uint32 nonce) external payable returns (uint64 sequence);

    function completeTransfer(bytes memory encodeVm) external;

    function encodeTransfer(Transfer memory transfer) external pure returns (bytes memory encoded);

    function parseTransfer(bytes memory encoded) external pure returns (Transfer memory transfer);

    function onERC721Received(address operator, address, uint256, bytes calldata) external view returns (bytes4);

    function governanceActionIsConsumed(bytes32 hash) external view returns (bool);

    function isInitialized(address impl) external view returns (bool);

    function isTransferCompleted(bytes32 hash) external view returns (bool);

    function wormhole() external view returns (IWormhole);

    function chainId() external view returns (uint16);

    function evmChainId() external view returns (uint256);

    function isFork() external view returns (bool);

    function governanceChainId() external view returns (uint16);

    function governanceContract() external view returns (bytes32);

    function wrappedAsset(uint16 tokenChainId, bytes32 tokenAddress) external view returns (address);

    function bridgeContracts(uint16 chainId_) external view returns (bytes32);

    function tokenImplementation() external view returns (address);

    function isWrappedAsset(address token) external view returns (bool);

    function splCache(uint256 tokenId) external view returns (SPLCache memory);

    function finality() external view returns (uint8);

    function initialize() external;

    function implementation() external view returns (address);

    function registerChain(bytes memory encodedVM) external;

    function upgrade(bytes memory encodedVM) external;

    function submitRecoverChainId(bytes memory encodedVM) external;

    function parseRegisterChain(bytes memory encoded) external pure returns(RegisterChain memory chain);

    function parseUpgrade(bytes memory encoded) external pure returns(UpgradeContract memory chain);

    function parseRecoverChainId(bytes memory encodedRecoverChainId) external pure returns (RecoverChainId memory rci);
}
