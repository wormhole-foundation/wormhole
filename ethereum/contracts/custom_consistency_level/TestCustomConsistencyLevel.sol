// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import "../interfaces/IWormhole.sol";
import "./interfaces/ICustomConsistencyLevel.sol";
import "./libraries/ConfigMakers.sol";

string constant testCustomConsistencyLevelVersion = "TestCustomConsistencyLevel-0.0.1";

/// @title TestCustomConsistencyLevel
/// @author Wormhole Project Contributors.
/// @notice The TestCustomConsistencyLevel contract can be used to test the custom consistency level functionality.
contract TestCustomConsistencyLevel {
    string public constant VERSION = testCustomConsistencyLevelVersion;

    ICustomConsistencyLevel public immutable customConsistencyLevel;
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
        ICustomConsistencyLevel(_customConsistencyLevel).configure(
            ConfigMakers.makeAdditionalBlocksConfig(_consistencyLevel, _blocks)
        );
    }

    // ==================== External Interface ===============================================

    function configure(uint8 _consistencyLevel, uint16 _blocks) external {
        customConsistencyLevel.configure(
            ConfigMakers.makeAdditionalBlocksConfig(_consistencyLevel, _blocks)
        );
    }

    function publishMessage(
        string memory str
    ) external payable returns (uint64 sequence) {
        nonce++;
        sequence = wormhole.publishMessage(nonce, bytes(str), 203);
    }
}
