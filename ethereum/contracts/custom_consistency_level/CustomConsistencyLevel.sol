// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import "./interfaces/ICustomConsistencyLevel.sol";

string constant customConsistencyLevelVersion = "CustomConsistencyLevel-0.0.1";

/// @title CustomConsistencyLevel
/// @author Wormhole Project Contributors.
/// @notice The CustomConsistencyLevel contract is an immutable contract that tracks custom consistency level configurations per integrator (emitter address).
contract CustomConsistencyLevel is ICustomConsistencyLevel {
    string public constant VERSION = customConsistencyLevelVersion;

    mapping(address => bytes32) private _configurations;

    // ==================== External Interface ===============================================

    /// @inheritdoc ICustomConsistencyLevel
    function configure(
        bytes32 config
    ) external override {
        _configurations[msg.sender] = config;
        emit ConfigSet(msg.sender, config);
    }

    /// @inheritdoc ICustomConsistencyLevel
    function getConfiguration(
        address emitterAddress
    ) external view override returns (bytes32) {
        return _configurations[emitterAddress];
    }
}
