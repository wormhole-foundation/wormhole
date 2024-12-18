// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import { ManualSigning } from "./ManualSigning.sol";

import { BackwardsOptimized } from "core-bridge/BackwardsOptimized/BackwardsOptimized.sol";

contract BackwardsOptimizedGasCost is ManualSigning {
  BackwardsOptimized private _backwardsOptimized;

  function setUp() public {
    _setUpManualSigning(wormhole.getGuardianSet(wormhole.getCurrentGuardianSetIndex()).keys.length);
    _backwardsOptimized = new BackwardsOptimized(_guardianAddrs);
  }

  function testGscdOptimizations() public view {
    (, bool success, string memory reason) =
      _backwardsOptimized.parseAndVerifyVM(_vaa);
    assertTrue(success, reason);
  }
}