// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

contract ThresholdVerificationState {
	error InvalidThresholdKeyIndex();
	error InvalidThresholdKeyAddress();
	error InvalidGuardianIndex();

  struct ShardInfo {
    bytes32 shard;
    bytes32 id;
  }

  // Threshold data is stored in a single array with stride 2:
  //   pubkey (32 bytes)
  //   expiration time (4 bytes)
  //   shard base (5 bytes)
  //   shard count (1 byte)
  uint256[] private _thresholdData;
  bytes32[] private _shardData;

  // Get the current threshold signature info
  function _getCurrentThresholdInfo() internal view returns (uint256 pubkey, uint32 index) {
    unchecked {
      uint256 length = _thresholdData.length;
      // NOTE: We assume that at least one threshold key has been set
      pubkey = _thresholdData[length - 2];
      index = uint32(length >> 1);
    }
  }

  function _getThresholdInfo(uint32 index) internal view returns (uint256 pubkey, uint32 expirationTime) {
    unchecked {
      uint256 offset = index << 1;
      require(offset < _thresholdData.length, InvalidThresholdKeyIndex());
      pubkey = _thresholdData[offset];
      expirationTime = _thresholdDataExpirationTime(_thresholdData[offset + 1]);
    }
  }
	
  function _appendThresholdKey(
    uint32 newIndex,
    uint256 pubkey,
    uint32 expirationDelaySeconds,
    ShardInfo[] memory shards
  ) internal {
    unchecked {
      // Verify the new address is not the zero address
      // This prevents errors from ecrecover returning the zero address
      // NOTE: This is actually already checked in ecrecover, but there's no harm in preventing that case here
      // NOTE: The pubkey is also known to be <= HALF_Q, based on the decoding in _decodeThresholdKeyUpdatePayload
      require(pubkey != 0, InvalidThresholdKeyAddress());

      // Get the current threshold info and verify the new index is sequential
      (, uint32 index) = _getCurrentThresholdInfo();
      require(newIndex == index + 1, InvalidThresholdKeyIndex());

      // Store the expiration time and current threshold address in past threshold info
      uint32 expirationTime = uint32(block.timestamp) + expirationDelaySeconds;
      _setThresholdDataExpirationTime(index, expirationTime);

      // Store the new threshold info
      _thresholdData.push(pubkey);
      _thresholdData.push(_createThresholdData(uint8(shards.length), uint40(_shardData.length)));
    }
  }

  function _getShards(uint32 guardianSet) internal view returns (ShardInfo[] memory) {
    unchecked {
      uint256 offset = guardianSet << 1;
      require(offset < _thresholdData.length, InvalidThresholdKeyIndex());
      (uint8 shardCount, uint40 shardBase) = _thresholdDataShardSlice(_thresholdData[offset + 1]);
      
      ShardInfo[] memory shards = new ShardInfo[](shardCount);
      uint256 ptr = shardBase;
      for (uint256 i = 0; i < shardCount; i++) {
        shards[i].shard = _shardData[ptr++];
        shards[i].id = _shardData[ptr++];
      }

      return shards;
    }
  }

  function _getShardsRaw(
    uint32 guardianSet
  ) internal view returns (uint shardCount, bytes32[] memory rawShards) {
    ShardInfo[] memory shards = _getShards(guardianSet);
    shardCount = shards.length;
    assembly ("memory-safe") {
      rawShards := shards
      mstore(rawShards, mul(shardCount, 2))
    }
  }

  function _registerGuardian(uint32 guardianSet, uint8 guardian, bytes32 id) internal {
    unchecked {
      uint256 offset = guardianSet << 1;
      require(offset < _thresholdData.length, InvalidThresholdKeyIndex());
      (uint8 shardCount, uint40 shardBase) = _thresholdDataShardSlice(_thresholdData[offset + 1]);
      require(guardian < shardCount, InvalidGuardianIndex());
      _shardData[shardBase + (guardian << 1)] = id;
    }
  }

  function _createThresholdData(uint8 shardCount, uint40 shardBase) internal pure returns (uint256) {
    return (shardCount << 32) | (shardBase << 40);
  }

  function _thresholdDataExpirationTime(uint256 data) internal pure returns (uint32) {
    unchecked {
      return uint32(data & 0xFFFFFFFF);
    }
  }

  function _thresholdDataShardSlice(uint256 data) internal pure returns (uint8 shardCount, uint40 shardBase) {
    unchecked {
      shardCount = uint8((data >> 32) & 0xFF);
      shardBase = uint40(data >> 40);
    }
  }

  function _setThresholdDataExpirationTime(uint32 index, uint32 expirationTime) internal {
    unchecked {
      _thresholdData[index << 1] |= expirationTime;
    }
  }
}
