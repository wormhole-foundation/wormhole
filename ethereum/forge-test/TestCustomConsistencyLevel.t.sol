// SPDX-License-Identifier: Apache-2.0
pragma solidity ^0.8.0;

import {Test, console} from "forge-std/Test.sol";
import {TestCustomConsistencyLevel} from
    "../contracts/custom_consistency_level/TestCustomConsistencyLevel.sol";
import {CustomConsistencyLevel} from
    "../contracts/custom_consistency_level/CustomConsistencyLevel.sol";
import {ConfigMakers} from "../contracts/custom_consistency_level/libraries/ConfigMakers.sol";

contract TestCustomConsistencyLevelTest is Test {
    CustomConsistencyLevel public customConsistencyLevel;
    TestCustomConsistencyLevel public testCustomConsistencyLevel;

    address public userA = address(0x123);
    address public userB = address(0x456);
    address public guardian = address(0x789);
    address public wormhole = address(0x123456);

    function setUp() public {
        customConsistencyLevel = new CustomConsistencyLevel();
        testCustomConsistencyLevel =
            new TestCustomConsistencyLevel(wormhole, address(customConsistencyLevel), 201, 5);
    }

    function test_configure() public {
        vm.startPrank(guardian);
        assertEq(
            0x01c9000500000000000000000000000000000000000000000000000000000000,
            customConsistencyLevel.getConfiguration(address(testCustomConsistencyLevel))
        );
        assertEq(bytes32(0), customConsistencyLevel.getConfiguration(userB));
    }
}
