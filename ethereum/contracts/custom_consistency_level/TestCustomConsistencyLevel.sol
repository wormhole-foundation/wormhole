// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import "../interfaces/IWormhole.sol";
import "./interfaces/ICustomConsistencyLevel.sol";
import "./libraries/ConfigMakers.sol";

/// @title TestCustomConsistencyLevel
/// @notice Harness to exercise CustomConsistencyLevel end-to-end.
contract TestCustomConsistencyLevel {
    string public constant VERSION = "TestCustomConsistencyLevel-0.0.1";

    ICustomConsistencyLevel public customConsistencyLevel;
    IWormhole public immutable wormhole;
    uint32 public nonce;

    constructor(
        address _wormhole,
        address _customConsistencyLevel,
        uint8 _consistencyLevel,
        uint16 _blocks
    ) {
        wormhole = IWormhole(_wormhole);
        customConsistencyLevel = ICustomConsistencyLevel(_customConsistencyLevel);

        // configure validated, canonicalized word
        customConsistencyLevel.configure(
            ConfigMakers.makeAdditionalBlocksConfig(_consistencyLevel, _blocks)
        );
    }

    /// Update configuration via the same canonical maker
    function configure(uint8 _consistencyLevel, uint16 _blocks) external {
        customConsistencyLevel.configure(
            ConfigMakers.makeAdditionalBlocksConfig(_consistencyLevel, _blocks)
        );
    }

    /// Publish a message respecting the Wormhole message fee and caller-provided consistency
    function publishMessage(
        uint8 consistencyLevel,
        string memory str
    ) external payable returns (uint64 sequence) {
        // Enforce the Wormhole fee contractually
        require(msg.value == wormhole.messageFee(), "incorrect message fee");

        unchecked { ++nonce; }
        // Forward exactly what the caller specifies; avoid magic numbers like 203
        sequence = wormhole.publishMessage{value: msg.value}(nonce, bytes(str), consistencyLevel);
    }
}