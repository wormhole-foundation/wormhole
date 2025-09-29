// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

interface ICustomConsistencyLevel {
    /// @notice The configuration for an emitter has been set.
    /// @dev Topic0
    ///      0xa37f0112e03d41de27266c1680238ff1548c0441ad1e73c82917c000eefdd5ea.
    /// @param emitterAddress The emitter address for which the config has been set.
    /// @param config The config data that was set.
    event ConfigSet(address emitterAddress, bytes32 config);

    /// @notice Sets / updates the configuration for a given emitter address (msg.sender).
    /// @param config The config used to determine the custom consistency level handling for an emitter.
    function configure(
        bytes32 config
    ) external;

    /// @notice Returns the configuration for a given emitter address.
    /// @param emitterAddress The emitter address for which the config has been set.
    /// @return The configuration.
    function getConfiguration(
        address emitterAddress
    ) external view returns (bytes32);
}
