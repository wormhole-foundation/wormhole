// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import {Test, console} from "forge-std/Test.sol";
import {CustomConsistencyLevel} from
    "../contracts/custom_consistency_level/CustomConsistencyLevel.sol";
import {ConfigMakers} from "../contracts/custom_consistency_level/libraries/ConfigMakers.sol";

contract CustomConsistencyLevelTest is Test {
    CustomConsistencyLevel public customConsistencyLevel;

    address public userA = address(0x123);
    address public userB = address(0x456);
    address public guardian = address(0x789);

    function setUp() public {
        customConsistencyLevel = new CustomConsistencyLevel();
    }

    function test_makeAdditionalBlocksConfig() public {
        bytes32 expected = 0x01c9002a00000000000000000000000000000000000000000000000000000000;
        bytes32 result = ConfigMakers.makeAdditionalBlocksConfig(201, 42);
        assertEq(expected, result);
    }

    function test_configure() public {
        vm.startPrank(userA);
        customConsistencyLevel.configure(ConfigMakers.makeAdditionalBlocksConfig(201, 42));

        vm.startPrank(guardian);
        assertEq(
            0x01c9002a00000000000000000000000000000000000000000000000000000000,
            customConsistencyLevel.getConfiguration(userA)
        );
        assertEq(bytes32(0), customConsistencyLevel.getConfiguration(userB));
    }
}
