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

    // Bit mask for valid configuration fields to prevent reserved bit pollution
    // Layout: [255:248] version, [247:240] consistency, [239:224] blocks, [223:0] reserved (must be zero)
    bytes32 private constant CONFIG_MASK = bytes32(
        (uint256(0xFF) << 248) | (uint256(0xFF) << 240) | (uint256(0xFFFF) << 224)
    );

    error InvalidReservedBits(bytes32 provided, bytes32 masked);

    // ==================== External Interface ===============================================

    /// @inheritdoc ICustomConsistencyLevel
    function configure(
        bytes32 config
    ) external override {
        // Enforce strict bit masking to prevent reserved bit pollution
        bytes32 masked = config & CONFIG_MASK;
        if (masked != config) {
            revert InvalidReservedBits(config, masked);
        }
        
        _configurations[msg.sender] = masked;
        emit ConfigSet(msg.sender, masked);
    }

    /// @inheritdoc ICustomConsistencyLevel
    function getConfiguration(
        address emitterAddress
    ) external view override returns (bytes32) {
        return _configurations[emitterAddress];
    }
}
