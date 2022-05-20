// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "./NFTBridge.sol";


contract NFTBridgeImplementation is NFTBridge {
    // Beacon getter for the token contracts
    function implementation() public view returns (address) {
        return tokenImplementation();
    }

    function initialize() initializer public virtual {
        // this function needs to be exposed for an upgrade to pass
        uint8 finality;
        uint256 chain = block.chainid;

        // Chain ids explicitly enumerated (from truffle-config.js)

        if (chain == 1) {
            // ethereum
            finality = 15;
        } else if (chain == 4) {
            // rinkeby
            finality = 15;
        } else if (chain == 5) {
            // goerli
            finality = 15;
        } else if (chain == 56) {
            // bsc
            finality = 15;
        } else if (chain == 97) {
            // bsc testnet
            finality = 15;
        } else if (chain == 137) {
            // polygon
            finality = 15;
        } else if (chain == 80001) {
            // polygon testnet
            finality = 15;
        } else if (chain == 43114) {
            // avalanche
            finality = 1;
        } else if (chain == 43113) {
            // avalanche testnet
            finality = 1;
        } else if (chain == 42262) {
            // oasis
            finality = 1;
        } else if (chain == 0x4e454152) {
            // aurora
            finality = 1;
        } else if (chain == 0x4e454153) {
            // aurora testnet
            finality = 1;
        } else if (chain == 250) {
            // fantom
            finality = 1;
        } else if (chain == 0xfa2) {
            // fantom testnet
            finality = 1;
        } else if (chain == 686) {
            // karura
            finality = 1;
        } else if (chain == 596) {
            // karura testnet
            finality = 1;
        } else if (chain == 597) {
            // acala testnet
            finality = 1;
        } else if (chain == 8217) {
            // klaytn
            finality = 1;
        } else if (chain == 1001) {
            // klaytn
            finality = 1;
        } else if (chain == 42220) {
            // celo
            finality = 1;
        } else if (chain == 44787) {
            // celo testnet
            finality = 1;
        } else if (chain == 1287) {
            // moonbeam testnet
            finality = 1;
        } else {
            revert("Unknown chain id.");
        }

        setFinality(finality);
    }

    modifier initializer() {
        address impl = ERC1967Upgrade._getImplementation();

        require(
            !isInitialized(impl),
            "already initialized"
        );

        setInitialized(impl);

        _;
    }
}
