// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import { GasTestBase } from "./GasTestBase.sol";

import { IWormhole                   } from "wormhole-sdk/interfaces/IWormhole.sol";
import { DEVNET_GUARDIAN_PRIVATE_KEY } from "wormhole-sdk/testing/Constants.sol";
import { VaaLib } from "wormhole-sdk/libraries/VaaLib.sol";
import {
  PublishedMessage,
  AdvancedWormholeOverride
} from "wormhole-sdk/testing/WormholeOverride.sol";

contract SingleSigGasCost is GasTestBase {
  using AdvancedWormholeOverride for IWormhole;
  using VaaLib for IWormhole.VM;

  bytes private _vaa;

  function setUp() public {
    uint256[] memory guardianSecrets = new uint256[](1);
    guardianSecrets[0] = DEVNET_GUARDIAN_PRIVATE_KEY;
    wormhole.setUpOverride(guardianSecrets);
    _vaa = wormhole.sign(_tbPublishedMsg()).encode();
  }

  function testGasSingleSig() public view {
    (, bool success, string memory reason) = wormhole.parseAndVerifyVM(_vaa);
    assertTrue(success, reason);
  }
}
