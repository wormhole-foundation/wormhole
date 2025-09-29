// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import "./interfaces/ICustomConsistencyLevel.sol";

/// @title CustomConsistencyLevel
/// @notice Tracks custom consistency configurations per integrator (emitter address).
contract CustomConsistencyLevel is ICustomConsistencyLevel {
    // Optional: keep a single in-contract version constant (no file-level duplicate).
    string public constant VERSION = "CustomConsistencyLevel-0.0.1";

    mapping(address => bytes32) private _configurations;

    // Layout: [255:248] version (8b), [247:240] consistency (8b), [239:224] blocks (16b), [223:0] reserved (must be zero)
    bytes32 private constant CONFIG_MASK = bytes32(
        (uint256(0xFF)   << 248) |   // version
        (uint256(0xFF)   << 240) |   // consistency
        (uint256(0xFFFF) << 224)     // blocks (16-bit)
    );

    error InvalidReservedBits(bytes32 provided, bytes32 masked);

    /// @inheritdoc ICustomConsistencyLevel
    function configure(bytes32 config) external override {
        // Validate BEFORE any state changes or events
        bytes32 masked = config & CONFIG_MASK;
        if (masked != config) revert InvalidReservedBits(config, masked);

        _configurations[msg.sender] = masked;
        emit ConfigSet(msg.sender, masked);
    }

    /// @inheritdoc ICustomConsistencyLevel
    function getConfiguration(address emitter) external view override returns (bytes32) {
        return _configurations[emitter];
    }
}