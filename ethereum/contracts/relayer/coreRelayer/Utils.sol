// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../interfaces/relayer/TypedUnits.sol";

error NotAnEvmAddress(bytes32);

function pay(address payable receiver, Wei amount) returns (bool success) {
  uint256 amount_ = Wei.unwrap(amount);
  if (amount_ != 0)
    (success,) = receiver.call{value: amount_}("");
  else
    success = true;
}

function min(uint256 a, uint256 b) pure returns (uint256) {
  return a < b ? a : b;
}

function min(uint64 a, uint64 b) pure returns (uint64) {
  return a < b ? a : b;
}

function max(uint256 a, uint256 b) pure returns (uint256) {
  return a > b ? a : b;
}

function toWormholeFormat(address addr) pure returns (bytes32) {
  return bytes32(uint256(uint160(addr)));
}

function fromWormholeFormat(bytes32 whFormatAddress) pure returns (address) {
  if (uint256(whFormatAddress) >> 160 != 0)
    revert NotAnEvmAddress(whFormatAddress);
  return address(uint160(uint256(whFormatAddress)));
}

function fromWormholeFormatUnchecked(bytes32 whFormatAddress) pure returns (address) {
  return address(uint160(uint256(whFormatAddress)));
}
