// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {eagerOr} from "wormhole-sdk/Utils.sol";
import {BytesParsing} from "wormhole-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";

bytes32 constant REGISTER_TYPE_HASH = keccak256(
  "GuardianRegister(uint32 guardianSet,uint256 nonce,bytes32 id)"
);

interface IERC5267 {
  function eip712Domain() external view returns (
    bytes1 fields,
    string memory name,
    string memory version,
    uint256 chainId,
    address verifyingContract,
    bytes32 salt,
    uint256[] memory extensions
  );
}

contract EIP712Encoding is IERC5267 {
  bytes32 constant EIP712_DOMAIN_TYPE_HASH = keccak256(
    "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
  );
  
  string constant EIP712_NAME = "Wormhole VerificationV2";
  string constant EIP712_VERSION = "1";

  bytes32 constant EIP712_NAME_HASH = keccak256(bytes(EIP712_NAME));
  bytes32 constant EIP712_VERSION_HASH = keccak256(bytes(EIP712_VERSION));


  function eip712Domain() external view returns (
    bytes1 fields,
    string memory name,
    string memory version,
    uint256 chainId,
    address verifyingContract,
    bytes32 salt,
    uint256[] memory extensions
  ) {
    return (
      bytes1(0x0F),
      EIP712_NAME,
      EIP712_VERSION,
      block.chainid,
      address(this),
      bytes32(0),
      new uint256[](0)
    );
  }

  function DOMAIN_SEPARATOR() public view returns (bytes32) {
    return getDomainSeparator(block.chainid, address(this));
  }

  function getDomainSeparator(
    uint256 ethChainId,
    address verifyingContract
  ) internal pure returns (bytes32) {
    return keccak256(abi.encode(
      EIP712_DOMAIN_TYPE_HASH,
      EIP712_NAME_HASH,
      EIP712_VERSION_HASH,
      ethChainId,
      verifyingContract
    ));
  }

  function getRegisterGuardianDigest(
    uint32 thresholdKeyIndex,
    uint256 nonce,
    bytes32 guardianId
  ) public view returns (bytes32) {
    bytes32 idHash = keccak256(abi.encode(
      REGISTER_TYPE_HASH,
      thresholdKeyIndex,
      nonce,
      guardianId
    ));

    return keccak256(abi.encodePacked("\x19\x01", DOMAIN_SEPARATOR(), idHash));
  }
}
