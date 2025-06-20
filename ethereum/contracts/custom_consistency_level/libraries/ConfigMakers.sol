// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import "../../libraries/external/BytesLib.sol";

library ConfigMakers {
    using BytesLib for bytes;

    uint8 public constant TYPE_ADDITIONAL_BLOCKS = 1;

    /// @notice Encodes an additional blocks custom consistency level configuration.
    /// @param consistencyLevel The consistency level to wait for.
    /// @param blocksToWait The number of additional blocks to wait after the consistency level is reached.
    /// @return bytes The encoded config.
    function makeAdditionalBlocksConfig(
        uint8 consistencyLevel,
        uint16 blocksToWait
    ) internal pure returns (bytes32) {
        bytes28 padding;
        return abi.encodePacked(TYPE_ADDITIONAL_BLOCKS, consistencyLevel, blocksToWait, padding)
            .toBytes32(0);
    }
}
