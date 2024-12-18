// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import { GasTestBase } from "./GasTestBase.sol";

contract OriginalGasCost is GasTestBase {
  function testOriginalGasCost() public view {
    (, bool success, string memory reason) = wormhole.parseAndVerifyVM(_tbOriginalVaa());
    assertTrue(success, reason);
  }
}
