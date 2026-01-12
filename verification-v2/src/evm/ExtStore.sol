// SPDX-License-Identifier: MIT
pragma solidity ^0.8.4;

/// @notice Library for computing contract addresses from their deployer and nonce.
/// @author Solady (https://github.com/vectorized/solady/blob/main/src/utils/LibRLP.sol)
/// @author Modified from Solmate (https://github.com/transmissions11/solmate/blob/main/src/utils/LibRLP.sol)
library LibRLP {
  /// @dev Returns the address where a contract will be stored if deployed via
  /// `deployer` with `nonce` using the `CREATE` opcode.
  /// For the specification of the Recursive Length Prefix (RLP)
  /// encoding scheme, please refer to p. 19 of the Ethereum Yellow Paper
  /// (https://ethereum.github.io/yellowpaper/paper.pdf)
  /// and the Ethereum Wiki (https://eth.wiki/fundamentals/rlp).
  ///
  /// Based on the EIP-161 (https://github.com/ethereum/EIPs/blob/master/EIPS/eip-161.md)
  /// specification, all contract accounts on the Ethereum mainnet are initiated with
  /// `nonce = 1`. Thus, the first contract address created by another contract
  /// is calculated with a non-zero nonce.
  ///
  /// The theoretical allowed limit, based on EIP-2681
  /// (https://eips.ethereum.org/EIPS/eip-2681), for an account nonce is 2**64-2.
  ///
  /// Caution! This function will NOT check that the nonce is within the theoretical range.
  /// This is for performance, as exceeding the range is extremely impractical.
  /// It is the user's responsibility to ensure that the nonce is valid
  /// (e.g. no dirty bits after packing / unpacking).
  ///
  /// Note: The returned result has dirty upper 96 bits. Please clean if used in assembly.
  function computeAddress(address deployer, uint256 nonce)
      internal
      pure
      returns (address deployed)
  {
    /// @solidity memory-safe-assembly
    assembly {
      for {} 1 {} {
        // The integer zero is treated as an empty byte string,
        // and as a result it only has a length prefix, 0x80,
        // computed via `0x80 + 0`.

        // A one-byte integer in the [0x00, 0x7f] range uses its
        // own value as a length prefix,
        // there is no additional `0x80 + length` prefix that precedes it.
        if iszero(gt(nonce, 0x7f)) {
          mstore(0x00, deployer)
          // Using `mstore8` instead of `or` naturally cleans
          // any dirty upper bits of `deployer`.
          mstore8(0x0b, 0x94)
          mstore8(0x0a, 0xd6)
          // `shl` 7 is equivalent to multiplying by 0x80.
          mstore8(0x20, or(shl(7, iszero(nonce)), nonce))
          deployed := keccak256(0x0a, 0x17)
          break
        }
        let i := 8
        // Just use a loop to generalize all the way with minimal bytecode size.
        for {} shr(i, nonce) { i := add(i, 8) } {}
        // `shr` 3 is equivalent to dividing by 8.
        i := shr(3, i)
        // Store in descending slot sequence to overlap the values correctly.
        mstore(i, nonce)
        mstore(0x00, shl(8, deployer))
        mstore8(0x1f, add(0x80, i))
        mstore8(0x0a, 0x94)
        mstore8(0x09, add(0xd6, i))
        deployed := keccak256(0x09, add(0x17, i))
        break
      }
    }
  }
}

/// @notice Read and write to persistent storage at a fraction of the cost.
/// @author Solady (https://github.com/vectorized/solady/blob/main/src/utils/SSTORE2.sol)
/// @author Saw-mon-and-Natalie (https://github.com/Saw-mon-and-Natalie)
/// @author Modified from Solmate (https://github.com/transmissions11/solmate/blob/main/src/utils/SSTORE2.sol)
/// @author Modified from 0xSequence (https://github.com/0xSequence/sstore2/blob/master/contracts/SSTORE2.sol)
/// stripped down version of SSTORE2.sol
library SSTORE2 {
  /*´:°•.°+.*•´.*:˚.°*.˚•´.°:°•.°•.*•´.*:˚.°*.˚•´.°:°•.°+.*•´.*:*/
  /*                         CONSTANTS                          */
  /*.•°:°.´+˚.*°.˚:*.´•*.+°.•°:´*.´•*.•°.•°:°.´:•˚°.*°.˚:*.´+°.•*/

  /// @dev We skip the first byte as it's a STOP opcode,
  /// which ensures the contract can't be called.
  uint256 internal constant DATA_OFFSET = 1;

  /*´:°•.°+.*•´.*:˚.°*.˚•´.°:°•.°•.*•´.*:˚.°*.˚•´.°:°•.°+.*•´.*:*/
  /*                        CUSTOM ERRORS                       */
  /*.•°:°.´+˚.*°.˚:*.´•*.+°.•°:´*.´•*.•°.•°:°.´:•˚°.*°.˚:*.´+°.•*/

  /// @dev Unable to deploy the storage contract.
  error DeploymentFailed();

  /// @dev The storage contract address is invalid.
  error InvalidPointer();

  /// @dev Attempt to read outside of the storage contract's bytecode bounds.
  error ReadOutOfBounds();

  /*´:°•.°+.*•´.*:˚.°*.˚•´.°:°•.°•.*•´.*:˚.°*.˚•´.°:°•.°+.*•´.*:*/
  /*                         WRITE LOGIC                        */
  /*.•°:°.´+˚.*°.˚:*.´•*.+°.•°:´*.´•*.•°.•°:°.´:•˚°.*°.˚:*.´+°.•*/

  /// @dev Writes `data` into the bytecode of a storage contract and returns its address.
  function write(bytes memory data) internal returns (address pointer) {
    /// @solidity memory-safe-assembly
    assembly {
      let originalDataLength := mload(data)

      // Add 1 to data size since we are prefixing it with a STOP opcode.
      let dataSize := add(originalDataLength, DATA_OFFSET)

      /**
        * ------------------------------------------------------------------------------+
        * Opcode      | Mnemonic        | Stack                   | Memory              |
        * ------------------------------------------------------------------------------|
        * 61 dataSize | PUSH2 dataSize  | dataSize                |                     |
        * 80          | DUP1            | dataSize dataSize       |                     |
        * 60 0xa      | PUSH1 0xa       | 0xa dataSize dataSize   |                     |
        * 3D          | RETURNDATASIZE  | 0 0xa dataSize dataSize |                     |
        * 39          | CODECOPY        | dataSize                | [0..dataSize): code |
        * 3D          | RETURNDATASIZE  | 0 dataSize              | [0..dataSize): code |
        * F3          | RETURN          |                         | [0..dataSize): code |
        * 00          | STOP            |                         |                     |
        * ------------------------------------------------------------------------------+
        * @dev Prefix the bytecode with a STOP opcode to ensure it cannot be called.
        * Also PUSH2 is used since max contract size cap is 24,576 bytes which is less than 2 ** 16.
        */
      mstore(
        // Do a out-of-gas revert if `dataSize` is more than 2 bytes.
        // The actual EVM limit may be smaller and may change over time.
        add(data, gt(dataSize, 0xffff)),
        // Left shift `dataSize` by 64 so that it lines up with the 0000 after PUSH2.
        or(0xfd61000080600a3d393df300, shl(0x40, dataSize))
      )

      // Deploy a new contract with the generated creation code.
      pointer := create(0, add(data, 0x15), add(dataSize, 0xa))

      // If `pointer` is zero, revert.
      if iszero(pointer) {
        // Store the function selector of `DeploymentFailed()`.
        mstore(0x00, 0x30116425)
        // Revert with (offset, size).
        revert(0x1c, 0x04)
      }

      // Restore original length of the variable size `data`.
      mstore(data, originalDataLength)
    }
  }

  /*´:°•.°+.*•´.*:˚.°*.˚•´.°:°•.°•.*•´.*:˚.°*.˚•´.°:°•.°+.*•´.*:*/
  /*                         READ LOGIC                         */
  /*.•°:°.´+˚.*°.˚:*.´•*.+°.•°:´*.´•*.•°.•°:°.´:•˚°.*°.˚:*.´+°.•*/

  /// @dev Returns all the `data` from the bytecode of the storage contract at `pointer`.
  function read(address pointer) internal view returns (bytes memory data) {
    /// @solidity memory-safe-assembly
    assembly {
      let pointerCodesize := extcodesize(pointer)
      if iszero(pointerCodesize) {
        // Store the function selector of `InvalidPointer()`.
        mstore(0x00, 0x11052bb4)
        // Revert with (offset, size).
        revert(0x1c, 0x04)
      }
      // Offset all indices by 1 to skip the STOP opcode.
      let size := sub(pointerCodesize, DATA_OFFSET)

      // Get the pointer to the free memory and allocate
      // enough 32-byte words for the data and the length of the data,
      // then copy the code to the allocated memory.
      // Masking with 0xffe0 will suffice, since contract size is less than 16 bits.
      data := mload(0x40)
      mstore(0x40, add(data, and(add(size, 0x3f), 0xffe0)))
      mstore(data, size)
      mstore(add(add(data, 0x20), size), 0) // Zeroize the last slot.
      extcodecopy(pointer, add(data, 0x20), DATA_OFFSET, size)
    }
  }
}

//assumes account nonce is 0 when deployed
abstract contract ExtStore {
  uint64 internal _nonce; //see EIP-2681

  function _extWrite(bytes memory data) internal returns (uint64 index) { unchecked {    
    SSTORE2.write(data);
    return _nonce++;
  }}

  function _extRead(uint64 index) internal view returns (bytes memory data) { unchecked {
    return SSTORE2.read(LibRLP.computeAddress(address(this), index+1));
  }}
}
