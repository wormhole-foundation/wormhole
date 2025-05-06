// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import {eagerOr} from "wormhole-sdk/Utils.sol";
import {BytesParsing} from "wormhole-sdk/libraries/BytesParsing.sol";
import {VaaLib} from "wormhole-sdk/libraries/VaaLib.sol";

contract EIP712Encoding {

  bytes32 constant EIP712_DOMAIN_TYPE_HASH = keccak256(
      "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
  );
  bytes32 constant EIP712_NAME_HASH = keccak256("Wormhole VerificationV2");
  bytes32 constant EIP712_VERSION_HASH = keccak256("1");

  bytes32 constant REGISTER_TYPE_HASH = keccak256(
    "GuardianRegister(uint32 guardianSet,uint32 expirationTime,bytes32 id)"
  );

  bytes32 private _domainSeparator;

  constructor () {
    _domainSeparator = getDomainSeparator(block.chainid, address(this));
  }

  function getDomainSeparator(
    uint256 eth_chain_id, address verifyingContract
  ) internal pure returns (bytes32) {
    return keccak256(abi.encode(
      EIP712_DOMAIN_TYPE_HASH,
      EIP712_NAME_HASH,
      EIP712_VERSION_HASH,
      eth_chain_id,
      verifyingContract
    ));
  }

  function getRegisterGuardianHash(
    uint32 guardianSet, uint32 expirationTime, bytes32 id
  ) internal pure returns (bytes32) {
    return keccak256(abi.encode(
      REGISTER_TYPE_HASH,
      guardianSet,
      expirationTime,
      id
    ));
  }

  function getRegisterGuardianDigest(
    uint32 guardianSet, uint32 expirationTime, bytes32 id
  ) internal view returns (bytes32) {
    bytes32 idHash = getRegisterGuardianHash(guardianSet, expirationTime, id);
    return keccak256(abi.encodePacked("\x19\x01", _domainSeparator, idHash));
  }
}