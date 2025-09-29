// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import "forge-std/Test.sol";
import "../contracts/custom_consistency_level/CustomConsistencyLevel.sol";
import "../contracts/custom_consistency_level/interfaces/ICustomConsistencyLevel.sol";

contract CustomConsistencyLevelTest is Test {
    CustomConsistencyLevel customConsistency;
    address emitter = address(0x123);

    // Field positions for testing
    uint256 constant SHIFT_VERSION = 248;     // [255:248] 8 bits
    uint256 constant SHIFT_CONSISTENCY = 240; // [247:240] 8 bits  
    uint256 constant SHIFT_BLOCKS = 224;      // [239:224] 16 bits
    uint256 constant SHIFT_RESERVED = 216;    // [223:216] 8 bits (should be zero)

    function setUp() public {
        customConsistency = new CustomConsistencyLevel();
    }

    function testConfigureRejectsInvalidBytes32() public {
        bytes32 config = bytes32(uint256(0x123456789abcdef)); // Has reserved bits
        
        vm.prank(emitter);
        vm.expectRevert();
        customConsistency.configure(config);
    }

    function testReservedBitsRejected() public {
        // Create config with reserved bits set
        uint8 version = 1;
        uint8 consistency = 200;
        uint16 blocks = 10;
        uint8 reservedJunk = 0xAA; // This should not be allowed
        
        bytes32 pollutedConfig = bytes32(
            (uint256(version) << SHIFT_VERSION) |
            (uint256(consistency) << SHIFT_CONSISTENCY) |
            (uint256(blocks) << SHIFT_BLOCKS) |
            (uint256(reservedJunk) << SHIFT_RESERVED)
        );
        
        vm.prank(emitter);
        vm.expectRevert();
        customConsistency.configure(pollutedConfig);
    }

    function testCleanConfigurationAccepted() public {
        uint8 version = 1;
        uint8 consistency = 200;
        uint16 intendedBlocks = 10;
        
        bytes32 cleanConfig = bytes32(
            (uint256(version) << SHIFT_VERSION) |
            (uint256(consistency) << SHIFT_CONSISTENCY) |
            (uint256(intendedBlocks) << SHIFT_BLOCKS)
            // No reserved bits set
        );
        
        vm.prank(emitter);
        customConsistency.configure(cleanConfig);
        
        bytes32 stored = customConsistency.getConfiguration(emitter);
        
        // Verify clean storage
        uint16 blocks16 = uint16(uint256(stored >> SHIFT_BLOCKS));
        uint8 storedReserved = uint8(uint256(stored >> SHIFT_RESERVED));
        
        assertEq(blocks16, intendedBlocks, "16-bit decode should return intended value");
        assertEq(storedReserved, 0, "Reserved bits should be zero");
    }

    function testFuzzReservedBitsAlwaysRejected(uint8 reservedBits) public {
        vm.assume(reservedBits != 0); // Only test non-zero reserved bits
        
        bytes32 config = bytes32(uint256(reservedBits) << SHIFT_RESERVED);
        
        vm.prank(emitter);
        vm.expectRevert();
        customConsistency.configure(config);
    }

    function testFuzzValidConfigurationAccepted(
        uint8 version,
        uint8 consistency, 
        uint16 blocks
    ) public {
        // Create clean config with only valid fields
        bytes32 config = bytes32(
            (uint256(version) << SHIFT_VERSION) |
            (uint256(consistency) << SHIFT_CONSISTENCY) |
            (uint256(blocks) << SHIFT_BLOCKS)
        );
        
        vm.prank(emitter);
        customConsistency.configure(config);
        
        bytes32 stored = customConsistency.getConfiguration(emitter);
        
        // Verify exact match
        assertEq(stored, config, "Clean config should be stored exactly");
        
        // Check that reserved bits are zero
        uint256 storedLower224 = uint256(stored) & ((1 << 224) - 1);
        assertEq(storedLower224, 0, "All reserved bits [223:0] should be zero");
    }

    function testEventEmittedWithCleanConfig() public {
        bytes32 cleanConfig = bytes32(
            (uint256(1) << SHIFT_VERSION) |
            (uint256(200) << SHIFT_CONSISTENCY) |
            (uint256(10) << SHIFT_BLOCKS)
        );
        
        vm.expectEmit(true, false, false, true);
        emit ICustomConsistencyLevel.ConfigSet(emitter, cleanConfig);
        
        vm.prank(emitter);
        customConsistency.configure(cleanConfig);
    }
}