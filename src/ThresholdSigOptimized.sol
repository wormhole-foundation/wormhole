// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import "wormhole-sdk/interfaces/IWormhole.sol";
import "wormhole-sdk/libraries/BytesParsing.sol";
import "wormhole-sdk/libraries/VaaLib.sol";
import { ProxyBase } from "wormhole-sdk/proxy/ProxyBase.sol";

contract ThresholdSigOptimized is ProxyBase {
  using BytesParsing for bytes;
  using VaaLib for bytes;

  mapping (uint32 => address) private _expiringThresholdAddrs;
  mapping (uint32 => uint32) private _expirationTimes;
  uint32 private _currentGuardianSetIndex;
  address private _currentThresholdAddr; //put in same storage slot as _currentGuardianSetIndex
                                         //to save gas on storage reads for most cases

  function _proxyConstructor(bytes calldata args) internal override {
    _currentThresholdAddr = abi.decode(args, (address));
  }

  function parseAndVerifyVMThreshold(
    bytes calldata encodedVaa
  ) external view returns (VaaBody memory) {
    ( uint32 guardianSetIndex,
      GuardianSignature[] memory sigs,
      uint envelopeOffset
    ) = encodedVaa.decodeVaaHeaderCdUnchecked();

    address thresholdAddr;
    if (guardianSetIndex != _currentGuardianSetIndex) {
      thresholdAddr = _expiringThresholdAddrs[guardianSetIndex];
      require(thresholdAddr != address(0), "invalid guardian set");
      require(
        _expirationTimes[guardianSetIndex] >= block.timestamp,
        "guardian set has expired"
      );
    }
    else
      thresholdAddr = _currentThresholdAddr;
    require(sigs.length == 1, "not a threshold signature vm");

    bytes32 vaaHash = encodedVaa.calcVaaDoubleHashCd(envelopeOffset);
    GuardianSignature memory sig = sigs[0];
    require(ecrecover(vaaHash, sig.v, sig.r, sig.s) == thresholdAddr, "threshold signature invalid");
    return encodedVaa.decodeVaaBodyStructCd(envelopeOffset);
  }
}