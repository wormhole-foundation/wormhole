// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import { IWormhole } from "wormhole-sdk/interfaces/IWormhole.sol";
import { BytesParsing } from "wormhole-sdk/libraries/BytesParsing.sol";
import { GuardianSignature, Vaa, VaaLib } from "wormhole-sdk/libraries/VaaLib.sol";
import {
  PublishedMessage,
  AdvancedWormholeOverride
} from "wormhole-sdk/testing/WormholeOverride.sol";

import { GasTestBase } from "./GasTestBase.sol";

contract ManualSigning is GasTestBase {
  using VaaLib for IWormhole.VM;
  using BytesParsing for bytes;

  bytes internal _vaa;
  address[] internal _guardianAddrs;

  uint256[] private _guardianPrivateKeys;

  //not nicely exported by forge, so we copy it here
  function _makeAddrAndKey(
    string memory name
  ) private returns (address addr, uint256 privateKey) {
    privateKey = uint256(keccak256(abi.encodePacked(name)));
    addr = vm.addr(privateKey);
    vm.label(addr, name);
  }

  function _sign(
    PublishedMessage memory pm,
    bytes memory signingGuardianIndices
  ) private view returns (Vaa memory vaa) { unchecked {
    vaa.header.guardianSetIndex = 0;
    bytes32 hash = VaaLib.calcDoubleHash(pm);
    vaa.header.signatures = new GuardianSignature[](signingGuardianIndices.length);
    for (uint i = 0; i < signingGuardianIndices.length; ++i) {
      (uint8 gi, ) = signingGuardianIndices.asUint8Mem(i);
      (vaa.header.signatures[i].v, vaa.header.signatures[i].r, vaa.header.signatures[i].s) =
        vm.sign(_guardianPrivateKeys[uint(gi)], hash);
      vaa.header.signatures[i].guardianIndex = gi;
    }
    vaa.envelope = pm.envelope;
    vaa.payload = pm.payload;
  }}

  function _setUpManualSigning(uint guardianCount) internal {
    for (uint i = 0; i < guardianCount; ++i) {
      (address ga, uint256 gpk) = _makeAddrAndKey(string.concat("guardian", vm.toString(i + 1)));
      _guardianAddrs.push(ga);
      _guardianPrivateKeys.push(gpk);
    }

    uint quorum = guardianCount * 2 / 3 + 1;
    bytes memory signingGuardianIndices = new bytes(0);
    for (uint i = 0; i < quorum; ++i)
      signingGuardianIndices = abi.encodePacked(signingGuardianIndices, uint8(i));

    _vaa = _sign(_tbPublishedMsg(), signingGuardianIndices).encode();
  }
}
