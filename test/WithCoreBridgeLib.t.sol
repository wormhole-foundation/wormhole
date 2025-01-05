// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import { GasTestBase } from "./GasTestBase.sol";

import { CoreBridgeLib } from "wormhole-sdk/libraries/CoreBridge.sol";

contract WithCoreBridgeLib is GasTestBase {
  function testWithCoreBridgeLib() public view {
    this.withCoreBridgeLibOptimized(_tbOriginalVaa());
  }

  function withCoreBridgeLibOptimized(bytes calldata encodedVaa) external view {
    CoreBridgeLib.parseAndVerifyVaa(address(wormhole), encodedVaa);
  }
}