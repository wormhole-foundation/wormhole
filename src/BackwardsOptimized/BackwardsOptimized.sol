// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.24;

import "wormhole-sdk/interfaces/IWormhole.sol";
import "wormhole-sdk/libraries/BytesParsing.sol";
import "wormhole-sdk/libraries/CoreBridge.sol";
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
    require(
      CoreBridgeLib.isVerifiedByQuorumMem(
        vm.hash,
        VaaLib.asGuardianSignatures(vm.signatures),
        guardianAddrs
      ),
      "VM not verified by quorum"
    );

    //backwards compatible nonsense:
    valid = true;
    reason = "";
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