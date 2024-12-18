// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import { GasTestBase } from "./GasTestBase.sol";

import { IWormhole } from "wormhole-sdk/interfaces/IWormhole.sol";
import {
  PublishedMessage,
  VaaEncoding,
  AdvancedWormholeOverride
} from "wormhole-sdk/testing/WormholeOverride.sol";

//this test should have the same gas cost as OriginalGasCost
contract ThirteenSigGasCost is GasTestBase {
  using AdvancedWormholeOverride for IWormhole;
  using VaaEncoding for IWormhole.VM;

  bytes private _vaa;

  function setUp() public {
    wormhole.setUpOverride();
    _vaa = wormhole.sign(_tbPublishedMsg()).encode();
  }

  function testGasMultipleSig() public view {
    (, bool success, string memory reason) = wormhole.parseAndVerifyVM(_vaa);
    assertTrue(success, reason);
  }
}
