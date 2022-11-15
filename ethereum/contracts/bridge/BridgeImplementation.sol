// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "./Bridge.sol";


contract BridgeImplementation is Bridge {
    // Beacon getter for the token contracts
    function implementation() public view returns (address) {
        return tokenImplementation();
    }

    function initialize() initializer public virtual {
        // this function needs to be exposed for an upgrade to pass
        address tokenContract;
        uint256 chain = chainId();

        // A previous upgrade
        // (https://github.com/wormhole-foundation/wormhole/commit/cb161db5e01970cc92d80fc679e506db054f8368)
        // set the token contract implementations for a number of chains,
        // excluding arbitrum and optimism, both of which were deployed with
        // that `initialize` contract, meaning there's a potential for a
        // temporary DoS.
        //
        // Performing this upgrade closes that DoS window.
        //
        // *insert rant about implicit variable initialisation*
        if        (chain == 23) { // arbitrum
            tokenContract = 0x53B56de645B9de6e5a40acE047D1c74E8B42Eccb;
        } else if (chain == 24) { // optimism
            tokenContract = 0xb91e3638F82A1fACb28690b37e3aAE45d2c33808;
        } else {
            revert("Chain not handled.");
        }

        setTokenImplementation(tokenContract);
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
