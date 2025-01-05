// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import { ManualSigning } from "./ManualSigning.sol";

import { BackwardsOptimized } from "core-bridge/BackwardsOptimized/BackwardsOptimized.sol";
import "wormhole-sdk/proxy/Proxy.sol";

contract BackwardsOptimizedGasCost is ManualSigning {
  BackwardsOptimized private _backwardsOptimized;

  function setUp() public {
    _setUpManualSigning(wormhole.getGuardianSet(wormhole.getCurrentGuardianSetIndex()).keys.length);
    _backwardsOptimized = BackwardsOptimized(payable(address(new Proxy(
      address(new BackwardsOptimized()),
      abi.encodePacked(_guardianAddrs)
    ))));
  }

  function testBackwardsOptimized() public view {
    (, bool success, string memory reason) =
      _backwardsOptimized.parseAndVerifyVM(_vaa);
    assertTrue(success, reason);
  }
}