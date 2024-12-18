// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import "wormhole-sdk/interfaces/IWormhole.sol";
import "wormhole-sdk/libraries/BytesParsing.sol";
import "./ExtStore.sol";

contract BackwardsOptimized is ExtStore {
  using BytesParsing for bytes;

  mapping(uint32 => uint32) private _guardianSetExpirationTimes;
  uint32 private _currentGuardianSetIndex;

  constructor(address[] memory guardians) ExtStore() {
    // bytes memory data = new bytes(0);
    // for (uint i = 0; i < guardians.length; ++i)
    //   data = abi.encodePacked(data, guardians[i]);

    //despite the "packed" it will actually pad each entry to 32 bytes
    _currentGuardianSetIndex = uint32(_extWrite(abi.encodePacked(guardians)));
  }

  function parseAndVerifyVM(
    bytes calldata encodedVM
  ) external view returns (IWormhole.VM memory vm, bool valid, string memory reason) { unchecked {
    vm = _parseVM(encodedVM);

    uint32 guardianSetIndex = vm.guardianSetIndex;
    address[] memory guardianAddrs = _getGuardianAddresses(guardianSetIndex);
    require(guardianAddrs.length != 0, "invalid guardian set");
    require(
      guardianSetIndex == _currentGuardianSetIndex ||
      _guardianSetExpirationTimes[guardianSetIndex] >= block.timestamp,
      "guardian set has expired"
    );
    require(vm.signatures.length > guardianAddrs.length * 2 / 3, "no quorum");
    _verifySignatures(vm.hash, vm.signatures, guardianAddrs);

    //backwards compatible nonsense:
    valid = true;
    reason = "";
  }}

  //does not check quorum!
  function _verifySignatures(
    bytes32 hash,
    IWormhole.Signature[] memory signatures,
    address[] memory guardianAddrs
  ) internal pure { unchecked {
    uint sigLen = signatures.length;
    uint guardianCount = guardianAddrs.length;
    int lastIndex = -1;
    for (uint i = 0; i < sigLen; ++i) {
      IWormhole.Signature memory sig = signatures[i];
      uint idx = sig.guardianIndex;
      require(int(idx) > lastIndex, "signature indices must be ascending");
      require(ecrecover(hash, sig.v, sig.r, sig.s) == guardianAddrs[idx], "VM signature invalid");
      lastIndex = int(idx);
    }
    require(lastIndex < int(guardianCount), "guardian index out of bounds");
  }}

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

  function getGuardianSet(uint32 index) external view returns (IWormhole.GuardianSet memory ret) {
    ret.keys = _getGuardianAddresses(index);
    ret.expirationTime = _guardianSetExpirationTimes[index];
  }

  function getCurrentGuardianSetIndex() external view returns (uint32) { unchecked {
    return _currentGuardianSetIndex;
  }}

  function _getGuardianAddresses(uint32 index) internal view returns (address[] memory ret) {
    bytes memory data = _extRead(index);
    assembly ("memory-safe") {
      ret := data
      mstore(ret, div(mload(ret), 32))
    }
  }
}