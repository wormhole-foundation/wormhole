// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../interfaces/relayer/TypedUnits.sol";

error NotAnEvmAddress(bytes32);

function pay(address payable receiver, LocalNative amount) returns (bool success) {
  uint256 amount_ = LocalNative.unwrap(amount);
  if (amount_ != 0)
    // TODO: we currently ignore the return data. Some users of this function might want to bubble up the return value though.
    // Specifying a higher limit than 63/64 of the remaining gas caps it at that amount without throwing an exception.
    (success,) = returnLengthBoundedCall(receiver, new bytes(0), gasleft(), amount_, 0);
  else
    success = true;
}

function pay(address payable receiver, LocalNative amount, uint256 gasBound) returns (bool success) {
  uint256 amount_ = LocalNative.unwrap(amount);
  if (amount_ != 0)
    // TODO: we currently ignore the return data. Some users of this function might want to bubble up the return value though.
    // Specifying a higher limit than 63/64 of the remaining gas caps it at that amount without throwing an exception.
    (success,) = returnLengthBoundedCall(receiver, new bytes(0), gasBound, amount_, 0);
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


uint256 constant freeMemoryPtr = 0x40;
uint256 constant memoryWord = 32;
uint256 constant maskModulo32 = 0x1f;

/**
 * Overload with no 'value' and non-payable address
 */
function returnLengthBoundedCall(
  address callee,
  bytes memory callData,
  uint256 gasLimit,
  uint256 dataLengthBound
) returns (bool success, bytes memory returnedData) {
  return returnLengthBoundedCall(payable(callee), callData, gasLimit, 0, dataLengthBound);
}

/**
 * Implements call that truncates return data to a specific size to avoid excessive gas consumption for relayers
 * when a revert or unexpectedly large return value is produced by the call.
 *
 * @param returnedData Buffer of returned data truncated to the first `dataLengthBound` bytes.
 */
function returnLengthBoundedCall(
  address payable callee,
  bytes memory callData,
  uint256 gasLimit,
  uint256 value,
  uint256 dataLengthBound
) returns (bool success, bytes memory returnedData) {
  uint256 callDataLength = callData.length;
  assembly ("memory-safe") {
    returnedData := mload(freeMemoryPtr)
    let returnedDataBuffer := add(returnedData, memoryWord)
    let callDataBuffer := add(callData, memoryWord)

    success := call(gasLimit, callee, value, callDataBuffer, callDataLength, returnedDataBuffer, dataLengthBound)
    let returnedDataSize := returndatasize()
    switch lt(dataLengthBound, returnedDataSize)
    case 1 {
      returnedDataSize := dataLengthBound
    } default {}
    mstore(returnedData, returnedDataSize)

    // Here we update the free memory pointer.
    // We want to pad `returnedData` to memory word size, i.e. 32 bytes.
    // Note that negating bitwise `maskModulo32` produces a mask that aligns addressing to 32 bytes.
    // This allows us to pad the entire `bytes` structure (length + buffer) to 32 bytes at the end.
    // We add `maskModulo32` to get the next free memory "slot" in case the `returnedDataSize` is not a multiple of the memory word size.
    //
    // Rationale:
    // We do not care about the alignment of the free memory pointer. The solidity compiler documentation does not promise nor require alignment on it.
    // It does however lightly suggest to pad `bytes` structures to 32 bytes: https://docs.soliditylang.org/en/v0.8.20/assembly.html#example
    // Searching for "alignment" and "padding" in https://gitter.im/ethereum/solidity-dev
    // yielded the following at the time of writing – paraphrased:
    // > It's possible that the compiler cleans that padding in some cases. Users should not rely on the compiler never doing that.
    // This means that we want to ensure that the free memory pointer points to memory just after this padding for our `returnedData` `bytes` structure.
    let paddedPastTheEndOffset := and(add(returnedDataSize, maskModulo32), not(maskModulo32))
    let newFreeMemoryPtr := add(returnedDataBuffer, paddedPastTheEndOffset)
    mstore(freeMemoryPtr, newFreeMemoryPtr)
  }
}
