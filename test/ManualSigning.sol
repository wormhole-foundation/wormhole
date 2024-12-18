// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import { IWormhole } from "wormhole-sdk/interfaces/IWormhole.sol";
import { BytesParsing } from "wormhole-sdk/libraries/BytesParsing.sol";
import {
  PublishedMessage,
  VaaEncoding,
  AdvancedWormholeOverride
} from "wormhole-sdk/testing/WormholeOverride.sol";

import { GasTestBase } from "./GasTestBase.sol";

contract ManualSigning is GasTestBase {
  using VaaEncoding for IWormhole.VM;
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
  ) private view returns (IWormhole.VM memory vaa) { unchecked {
    vaa.version = 1;
    vaa.timestamp = pm.timestamp;
    vaa.nonce = pm.nonce;
    vaa.emitterChainId = pm.emitterChainId;
    vaa.emitterAddress = pm.emitterAddress;
    vaa.sequence = pm.sequence;
    vaa.consistencyLevel = pm.consistencyLevel;
    vaa.payload = pm.payload;
    vaa.guardianSetIndex = 0;

    bytes memory encodedBody = abi.encodePacked(
      pm.timestamp,
      pm.nonce,
      pm.emitterChainId,
      pm.emitterAddress,
      pm.sequence,
      pm.consistencyLevel,
      pm.payload
    );
    vaa.hash = keccak256(abi.encodePacked(keccak256(encodedBody)));

    vaa.signatures = new IWormhole.Signature[](signingGuardianIndices.length);
    for (uint i = 0; i < signingGuardianIndices.length; ++i) {
      (uint8 gi, ) = signingGuardianIndices.asUint8(i);
      (vaa.signatures[i].v, vaa.signatures[i].r, vaa.signatures[i].s) =
        vm.sign(_guardianPrivateKeys[uint(gi)], vaa.hash);
      vaa.signatures[i].guardianIndex = gi;
      vaa.signatures[i].v -= 27;
    }
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
