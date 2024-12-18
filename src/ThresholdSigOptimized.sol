// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import "wormhole-sdk/interfaces/IWormhole.sol";
import "wormhole-sdk/libraries/BytesParsing.sol";

contract ThresholdSigOptimized {
  using BytesParsing for bytes;

  mapping (uint32 => address) private _expiringThresholdAddrs;
  mapping (uint32 => uint32) private _expirationTimes;
  uint32 private _currentGuardianSetIndex;
  address private _currentThresholdAddr; //put in same storage slot as _currentGuardianSetIndex
                                         //to save gas on storage reads for most cases

  constructor(address thresholdAddr) {
    _currentThresholdAddr = thresholdAddr;
  }

  function parseAndVerifyVMThreshold(
    bytes calldata encodedVM
  ) external view returns (IWormhole.VM memory vm) {
    vm = _parseVM(encodedVM);

    uint32 guardianSetIndex = vm.guardianSetIndex;
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
    require(vm.signatures.length == 1, "not a threshold signature vm");
    IWormhole.Signature memory sig = vm.signatures[0];
    require(ecrecover(vm.hash, sig.v, sig.r, sig.s) == thresholdAddr, "threshold signature invalid");
  }

  function _parseVM(
    bytes calldata encodedVM
  ) internal pure returns (IWormhole.VM memory vm) { unchecked {
    uint offset = 0;

    (vm.version, offset) = encodedVM.asUint8CdUnchecked(offset);
    require(vm.version == 1, "VM version incompatible");

    (vm.guardianSetIndex, offset) = encodedVM.asUint32CdUnchecked(offset);

    uint signersLen;
    (signersLen, offset) = encodedVM.asUint8CdUnchecked(offset);

    vm.signatures = new IWormhole.Signature[](signersLen);
    for (uint i = 0; i < signersLen; ++i) {
      (vm.signatures[i].guardianIndex, offset) = encodedVM.asUint8CdUnchecked(offset);
      (vm.signatures[i].r, offset) = encodedVM.asBytes32CdUnchecked(offset);
      (vm.signatures[i].s, offset) = encodedVM.asBytes32CdUnchecked(offset);
      (vm.signatures[i].v, offset) = encodedVM.asUint8CdUnchecked(offset);
      vm.signatures[i].v += 27;
    }

    (bytes memory body, ) = encodedVM.sliceCdUnchecked(offset, encodedVM.length - offset);
    vm.hash = keccak256(abi.encodePacked(keccak256(body)));

    (vm.timestamp,        offset) = encodedVM.asUint32CdUnchecked(offset);
    (vm.nonce,            offset) = encodedVM.asUint32CdUnchecked(offset);
    (vm.emitterChainId,   offset) = encodedVM.asUint16CdUnchecked(offset);
    (vm.emitterAddress,   offset) = encodedVM.asBytes32CdUnchecked(offset);
    (vm.sequence,         offset) = encodedVM.asUint64CdUnchecked(offset);
    (vm.consistencyLevel, offset) = encodedVM.asUint8CdUnchecked(offset);

    (vm.payload, ) = encodedVM.sliceCdUnchecked(offset, encodedVM.length - offset);
  }}
}