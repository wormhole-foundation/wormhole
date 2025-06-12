// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

struct ShardInfo {
  bytes32 shard;
  bytes32 id;
}

contract ThresholdVerificationState {
	error InvalidThresholdKeyIndex();
	error InvalidThresholdKeyAddress();
	error InvalidGuardianIndex();
	error GuardianSetsNotComplete();

  struct ThresholdKeyInfo {
    uint256 pubkey;
    uint32 expirationTime;
    uint8 shardCount;
    uint40 shardBase;
    uint32 guardianSetIndex;
  }

  uint32 private _currentThresholdKeyIndex = type(uint32).max;
  mapping (uint32 => ThresholdKeyInfo) private _thresholdKeyData;
  ShardInfo[] private _shardData;

  // Get the current threshold signature info
  // NOTE: This will panic if the threshold data is empty
  function _getCurrentThresholdInfo() internal view returns (ThresholdKeyInfo memory info, uint32 index) {
    unchecked {
      index = _currentThresholdKeyIndex;
      info = _thresholdKeyData[index];
    }
  }

  // NOTE: This will panic if the guardian set index is out of bounds
  function _getThresholdInfo(uint32 thresholdKeyIndex) internal view returns (ThresholdKeyInfo memory info) {
    unchecked {
      info = _thresholdKeyData[thresholdKeyIndex];
    }
  }

  function _appendThresholdKey(
    uint32 currentGuardianSetIndex,
    uint32 thresholdKeyIndex,
    uint256 pubkey,
    uint32 expirationDelaySeconds,
    ShardInfo[] memory shards
  ) internal {
    unchecked {
      // Verify that it is a new key
      require(_currentThresholdKeyIndex == type(uint32).max || thresholdKeyIndex > _currentThresholdKeyIndex, InvalidThresholdKeyIndex());

      // If there is a previous threshold key that is now expired, store the expiration time
      if (_currentThresholdKeyIndex > 0) {
        _thresholdKeyData[_currentThresholdKeyIndex].expirationTime = uint32(block.timestamp) + expirationDelaySeconds;
      }

      // Store the new threshold info
      _thresholdKeyData[thresholdKeyIndex] = ThresholdKeyInfo({
        pubkey: pubkey,
        expirationTime: 0,
        shardCount: uint8(shards.length),
        shardBase: uint40(_shardData.length),
        guardianSetIndex: currentGuardianSetIndex
      });

      _currentThresholdKeyIndex = thresholdKeyIndex;

      // Store the shard data
      // TODO: Assembly block could be used here to save gas
      for (uint256 i = 0; i < shards.length; i++) {
        _shardData.push(shards[i]);
      }
    }
  }

  // NOTE: This will panic if the guardian set index is out of bounds
  function _getShards(uint32 thresholdKeyIndex) internal view returns (ShardInfo[] memory) {
    unchecked {
      ThresholdKeyInfo memory info = _getThresholdInfo(thresholdKeyIndex);
      uint8 shardCount = info.shardCount;
      uint40 shardBase = info.shardBase;

      ShardInfo[] memory shards = new ShardInfo[](shardCount);
      for (uint256 i = 0; i < shardCount; i++) {
        shards[i] = _shardData[shardBase + i];
      }

      return shards;
    }
  }

  // NOTE: This will panic if the guardian set index is out of bounds
  function _registerGuardian(uint32 thresholdKeyIndex, uint8 guardianIndex, bytes32 id) internal {
    unchecked {
      ThresholdKeyInfo memory info = _getThresholdInfo(thresholdKeyIndex);
      uint8 shardCount = info.shardCount;
      uint40 shardBase = info.shardBase;

      require(guardianIndex < shardCount, InvalidGuardianIndex());
      _shardData[shardBase + guardianIndex].id = id;
    }
  }
}
