// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import "wormhole-sdk/interfaces/IWormhole.sol";
import "wormhole-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";
import { ProxyBase } from "wormhole-sdk/proxy/ProxyBase.sol";
import "./ExtStore.sol";

contract BackwardsOptimized is ProxyBase, ExtStore {
  using BytesParsing for bytes;
  using VaaLib for bytes;

  mapping(uint32 => uint32) private _guardianSetExpirationTimes;
  uint32 private _currentGuardianSetIndex;

  constructor() ExtStore() {}

  function _proxyConstructor(bytes calldata args) internal override {
    // bytes memory data = new bytes(0);
    // for (uint i = 0; i < guardians.length; ++i)
    //   data = abi.encodePacked(data, guardians[i]);

    //despite the "packed" it will actually pad each entry to 32 bytes
    _currentGuardianSetIndex = uint32(_extWrite(abi.encodePacked(args)));
  }

  function parseAndVerifyVM(
    bytes calldata encodedVm
  ) external view returns (IWormhole.VM memory vm, bool valid, string memory reason) { unchecked {
    vm = encodedVm.decodeVmStructCd();

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