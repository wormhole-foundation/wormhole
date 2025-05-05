// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

contract ThresholdVerificationState {
  uint256 constant SHARD_INFO_SIZE = 32 + 32;

	error InvalidThresholdKeyIndex();
	error InvalidThresholdKeyAddress();

  struct ShardInfo {
    bytes32 shard;
    bytes32 tlsKey;
  }

	// Current threshold info is stored in a single slot
  // Format:
  //   index (32 bits)
  //   address (160 bits)
  uint256 private _currentThresholdInfo;

  // Past threshold info is stored in an array
  // Format:
  //   expiration time (32 bits)
  //   address (160 bits)
  uint256[] private _pastThresholdInfo;
  
  ShardInfo[][] private _shards;
	
  function _decodeThresholdInfo(uint256 info) private pure returns (address addr, uint32 index) {
    return (address(uint160(info >> 32)), uint32(info & 0xFFFFFFFF));
  }

  function _encodeThresholdInfo(address addr, uint32 index) private pure returns (uint256) {
    return (uint256(uint160(addr)) << 32) | uint256(index);
  }

  // Get the current threshold signature info
  function _getCurrentThresholdInfo() internal view returns (address addr, uint32 index) {
    return _decodeThresholdInfo(_currentThresholdInfo);
  }

  // Get the past threshold signature info
  function _getPastThresholdInfo(uint32 index) internal view returns (
    address addr,
    uint32 expirationTime
  ) {
    require(index < _pastThresholdInfo.length, InvalidThresholdKeyIndex());
    return _decodeThresholdInfo(_pastThresholdInfo[index]);
  }

  function _getThresholdInfo(uint32 index) internal view returns (address thresholdAddr, uint32 expirationTime) {
    (address currentAddr, uint32 currentIndex) = _getCurrentThresholdInfo();
    return index == currentIndex ? (currentAddr, 0) : _getPastThresholdInfo(index);
  }
	
  function _appendThresholdKey(
    uint32 newIndex,
    address newAddr,
    uint32 expirationDelaySeconds,
    ShardInfo[] memory shards
  ) internal {
    unchecked {
      // Verify the new address is not the zero address
      // This prevents errors from ecrecover returning the zero address
      require(newAddr != address(0), InvalidThresholdKeyAddress());

      // Get the current threshold info and verify the new index is sequential
      (address currentAddr, uint32 index) = _getCurrentThresholdInfo();
      require(newIndex == index + 1, InvalidThresholdKeyIndex());

      // Store the expiration time and current threshold address in past threshold info
      uint32 expirationTime = uint32(block.timestamp) + expirationDelaySeconds;
      _pastThresholdInfo.push(_encodeThresholdInfo(currentAddr, expirationTime));

      // Update the current threshold info
      _currentThresholdInfo = _encodeThresholdInfo(newAddr, newIndex);

      // Push the shards
      _shards.push(shards);
    }
  }

  function _getShards(uint32 guardianSetIndex) internal view returns (ShardInfo[] memory shards) {
    return _shards[guardianSetIndex];
  }

  function _getShardsRaw(
    uint32 guardianSetIndex
  ) internal view returns (uint shardCount, bytes32[] memory rawShards) {
    ShardInfo[] memory shards = _getShards(guardianSetIndex);
    shardCount = shards.length;
    assembly {
      rawShards := shards
      mstore(rawShards, mul(shardCount, 2))
    }
  }

  function _registerTLSKey(
    uint32 guardianSetIndex,
    uint8 guardianIndex,
    bytes32 tlsKey
  ) internal {
    _shards[guardianSetIndex][guardianIndex].tlsKey = tlsKey;
  }

}
