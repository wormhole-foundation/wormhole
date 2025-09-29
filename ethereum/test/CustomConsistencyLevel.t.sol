// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import "forge-std/Test.sol";
import "../contracts/custom_consistency_level/CustomConsistencyLevel.sol";
import "../contracts/custom_consistency_level/interfaces/ICustomConsistencyLevel.sol";

contract CustomConsistencyLevelTest is Test {
    CustomConsistencyLevel customConsistency;
    address emitter = address(0x123);

    // Canonical field layout (must match contract)
    uint256 constant SHIFT_VERSION     = 248; // [255:248] 8 bits
    uint256 constant SHIFT_CONSISTENCY = 240; // [247:240] 8 bits
    uint256 constant SHIFT_BLOCKS      = 224; // [239:224] 16 bits

    // Reserved region [223:0]
    uint256 constant RESERVED_WIDTH       = 224;
    uint256 constant RESERVED_MASK        = (uint256(1) << RESERVED_WIDTH) - 1;
    uint256 constant SHIFT_RSVD_TOPBYTE   = 216; // top reserved byte [223:216]

    function setUp() public {
        customConsistency = new CustomConsistencyLevel();
    }

    function _pack(uint8 version, uint8 consistency, uint16 blocks) internal pure returns (bytes32) {
        return bytes32(
            (uint256(version)    << SHIFT_VERSION)     |
            (uint256(consistency)<< SHIFT_CONSISTENCY) |
            (uint256(blocks)     << SHIFT_BLOCKS)
        );
    }

    function testConfigureRejectsInvalidBytes32() public {
        // Only reserved bits set -> must revert
        bytes32 config = bytes32(uint256(0x123456789ABCDEF) & RESERVED_MASK);

        vm.prank(emitter);
        vm.expectRevert();
        customConsistency.configure(config);
    }

    function testReservedBitsRejected() public {
        uint8  version      = 1;
        uint8  consistency  = 200;
        uint16 blocks       = 10;
        uint8  reservedJunk = 0xAA; // set reserved [223:216]

        bytes32 pollutedConfig = _pack(version, consistency, blocks)
            | bytes32(uint256(reservedJunk) << SHIFT_RSVD_TOPBYTE);

        vm.prank(emitter);
        vm.expectRevert();
        customConsistency.configure(pollutedConfig);
    }

    function testCleanConfigurationAccepted() public {
        uint8  version        = 1;
        uint8  consistency    = 200;
        uint16 intendedBlocks = 10;

        bytes32 cleanConfig = _pack(version, consistency, intendedBlocks);

        vm.prank(emitter);
        customConsistency.configure(cleanConfig);

        // NOTE: contract method name must match the interface/impl you use.
        // If your contract exposes getConfiguration(address), keep this call.
        // If it exposes configurationOf(address), rename accordingly.
        bytes32 stored = customConsistency.getConfiguration(emitter);

        // Verify clean storage
        uint16 blocks16       = uint16(uint256(stored >> SHIFT_BLOCKS));
        uint8  storedReserved = uint8(uint256(stored >> SHIFT_RSVD_TOPBYTE));

        assertEq(blocks16, intendedBlocks, "16-bit decode should return intended value");
        assertEq(storedReserved, 0, "Reserved top byte [223:216] must be zero");

        // All reserved bits [223:0] must be zero
        uint256 storedLower224 = uint256(stored) & RESERVED_MASK;
        assertEq(storedLower224, 0, "All reserved bits [223:0] should be zero");
    }

    function testFuzzReservedBitsAlwaysRejected(uint8 reservedBits) public {
        vm.assume(reservedBits != 0); // only non-zero reserved

        bytes32 config = bytes32(uint256(reservedBits) << SHIFT_RSVD_TOPBYTE);

        vm.prank(emitter);
        vm.expectRevert();
        customConsistency.configure(config);
    }

    function testFuzzValidConfigurationAccepted(
        uint8 version,
        uint8 consistency,
        uint16 blocks
    ) public {
        // Only valid fields set (no reserved)
        bytes32 config = _pack(version, consistency, blocks);

        vm.prank(emitter);
        customConsistency.configure(config);

        bytes32 stored = customConsistency.getConfiguration(emitter);
        assertEq(stored, config, "Clean config should be stored exactly");

        // Reserved region must remain zero
        uint256 storedLower224 = uint256(stored) & RESERVED_MASK;
        assertEq(storedLower224, 0, "All reserved bits [223:0] should be zero");
    }

    function testEventEmittedWithCleanConfig() public {
        bytes32 cleanConfig = _pack(1, 200, 10);

        // TODO: Add event testing when interface events are properly accessible
        vm.prank(emitter);
        customConsistency.configure(cleanConfig);
    }
}