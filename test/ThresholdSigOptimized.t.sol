// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import { ManualSigning } from "./ManualSigning.sol";

import { ThresholdSigOptimized } from "core-bridge/ThresholdSigOptimized.sol";

contract ThresholdSigOptimizedGasCost is ManualSigning {
  ThresholdSigOptimized private _thresholdSigOptimized;

  function setUp() public {
    _setUpManualSigning(1);
    _thresholdSigOptimized = new ThresholdSigOptimized(_guardianAddrs[0]);
  }

  function testThresholdSigOptimizations() public view {
    _thresholdSigOptimized.parseAndVerifyVMThreshold(_vaa);
  }
}