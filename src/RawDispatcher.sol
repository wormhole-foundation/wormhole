// SPDX-License-Identifier: MIT

pragma solidity ^0.8.0;

abstract contract RawDispatcher {
  uint8 public constant VERSION = 1;

  function exec768() external payable returns (bytes memory) { return _exec(msg.data[4:]); }
  function get1959() external view    returns (bytes memory) { return  _get(msg.data[4:]); }

  function _exec(bytes calldata data) internal      virtual returns (bytes memory);
  function  _get(bytes calldata data) internal view virtual returns (bytes memory);
}
