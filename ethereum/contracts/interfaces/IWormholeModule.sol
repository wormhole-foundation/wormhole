// interfaces/contracts/IWormholeModule.sol
// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

/**
 * Interface for Wormhole modules. The main purpose of this is module is to have
 * a unified way of querying the module's name so we can sanity check this
 * during contract upgrades.
 */
interface IWormholeModule {
    /**
     * Returns the module's name.
     */
    function getModule() external returns (bytes32);

    /**
     * Ensures that the current module is the same as some `other` module.
     * Typically this would be added to a contract upgrade call, to check that
     * the new implementation is what we expect it to be.
     *
     * Note that this is merely a sanity check, and has no security
     * implications. A malicious contract can simply implement this interface
     * and lie about which method it is.
     */
    modifier sameModule(address other) {
        require(this.getModule() == IWormholeModule(other).getModule(), "Invalid module");

        _;
    }
}

