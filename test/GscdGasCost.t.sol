// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import { ManualSigning } from "./ManualSigning.sol";

import { Wormhole } from "core-bridge/gscd/Wormhole.sol";
import { Setup } from "core-bridge/gscd/Setup.sol";
import { Implementation } from "core-bridge/gscd/Implementation.sol";

contract GscdGasCost is ManualSigning {

  Implementation private _gscdCore;
  bytes private _encodedGuardianSet;

  function setUp() public {
    _setUpManualSigning(wormhole.getGuardianSet(wormhole.getCurrentGuardianSetIndex()).keys.length);

    _gscdCore = Implementation(payable(address(new Wormhole(
      address(new Setup()),
      abi.encodeWithSignature(
        "setup(address,address[],uint16,uint16,bytes32,uint256)",
        address(new Implementation()),
        _guardianAddrs,
        wormhole.chainId(),
        wormhole.governanceChainId(),
        wormhole.governanceContract(),
        wormhole.evmChainId()
      )
    ))));

    _encodedGuardianSet = abi.encodePacked(
      _guardianAddrs[0],
      _guardianAddrs[1],
      _guardianAddrs[2],
      _guardianAddrs[3],
      _guardianAddrs[4],
      _guardianAddrs[5],
      _guardianAddrs[6],
      _guardianAddrs[7],
      _guardianAddrs[8],
      _guardianAddrs[9],
      _guardianAddrs[10],
      _guardianAddrs[11],
      _guardianAddrs[12],
      _guardianAddrs[13],
      _guardianAddrs[14],
      _guardianAddrs[15],
      _guardianAddrs[16],
      _guardianAddrs[17],
      _guardianAddrs[18],
      uint32(0)
    );
  }

  function testGscdOptimizations() public view {
    (, bool success, string memory reason) =
      _gscdCore.parseAndVerifyVMOptimized(_vaa, _encodedGuardianSet, 0);
    assertTrue(success, reason);
  }
}