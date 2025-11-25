pragma solidity ^0.8.0;

import "../interfaces/IWormhole.sol";
import "../libraries/external/BytesLib.sol";

/**
 * @title WormholeDelegatedGuardians
 * @author Wormhole Project Contributors.
 * @notice The WormholeDelegatedGuardians contract is an immutable contract that configures delegated guardians per chain id
 */
contract WormholeDelegatedGuardians {
  using BytesLib for bytes;

  struct ConfigPayload {
    uint256 configIndex;
    DelegatedGuardianPayload[] configs;
  }

  struct DelegatedGuardianPayload {
    uint16 chainId;
    uint8 threshold;
    address[] keys;
  }

  struct DelegatedGuardianSet {
    uint16 chainId;
    uint32 timestamp;
    uint8 threshold;
    address[] keys;
  }

  enum Action {
    SET_CONFIG
  }

  // "DelegatedGuardians" left padded
  bytes32 private constant MODULE = 0x000000000000000000000000000044656C656761746564477561726469616E73;
  IWormhole private immutable wormhole;
  mapping(uint16 => DelegatedGuardianSet[]) private delegatedGuardianSets;
  uint16[] private chainIds;
  mapping(bytes32 => bool) public governanceActionsConsumed;
  uint256 private nextConfigIndex;

  event ChainConfigSet(uint256 configIndex, uint16 chainId, uint8 threshold, address[] keys);

  error InvalidModule(bytes32 module);
  error InvalidAction(uint8 action);
  error InvalidChainId(uint16 chainId);
  error InvalidGovernanceChainId(uint16 chainId);
  error InvalidGovernanceContract(bytes32 contractAddress);
  error GovernanceActionAlreadyConsumed(bytes32 digest);
  error InvalidNextConfigIndex(uint256 nextConfigIndex);
  error InvalidConfig(uint16 chainId);

  constructor(address wormholeAddress) {
    wormhole = IWormhole(wormholeAddress);
  }

  /**
   * Governed function to submit a new config for multiple chains
   * @param vaa encoded VAA
   * @dev Configure 50 chains in a single VAA: ~14M gas
   * Governance Payload format:
   *    Config index (32 bytes)
   *    Number of configs (1 byte)
   *    For each config:
   *      Chain ID (2 bytes)
   *      Threshold (1 byte)
   *      Number of keys (1 byte)
   *      For each key (20 bytes)
   */
  function submitConfig(bytes calldata vaa) public {
    IWormhole.VM memory vm = _verifyGovernanceVAA(vaa);
    bytes memory payload = _parseConfigMessage(vm.payload);
    ConfigPayload memory configPayload = _decodeConfigPayload(payload);

    if (configPayload.configIndex != nextConfigIndex) {
      revert InvalidNextConfigIndex(configPayload.configIndex);
    }

    for (uint256 i = 0; i < configPayload.configs.length; i++) {
      DelegatedGuardianPayload memory config = configPayload.configs[i];
      _processGovernanceConfig(config);
    }

    nextConfigIndex++;
  }

  function getConfig() public view returns (DelegatedGuardianSet[] memory) {
    DelegatedGuardianSet[] memory configs = new DelegatedGuardianSet[](chainIds.length);
    for (uint256 i = 0; i < chainIds.length; i++) {
      DelegatedGuardianSet[] storage set = delegatedGuardianSets[chainIds[i]];
      DelegatedGuardianSet memory latestSet = set[set.length - 1];
      configs[i] = latestSet;
    }
    return configs;
  }

  function getConfig(uint16 _chainId) public view returns (DelegatedGuardianSet memory) {
    DelegatedGuardianSet[] storage set = delegatedGuardianSets[_chainId];
    return set[set.length - 1];
  }

  function getHistoricalConfig(uint16 _chainId) public view returns (DelegatedGuardianSet[] memory) {
    return delegatedGuardianSets[_chainId];
  }

  function getHistoricalConfigLength(uint16 _chainId) public view returns (uint256) {
    return delegatedGuardianSets[_chainId].length;
  }

  function getHistoricalConfig(uint16 _chainId, uint256 _index) public view returns (DelegatedGuardianSet memory) {
    return delegatedGuardianSets[_chainId][_index];
  }

  function chainIdsLength() public view returns (uint256) {
    return chainIds.length;
  }

  function getChainIds() public view returns (uint16[] memory) {
    return chainIds;
  }

  function getChainId(uint256 index) public view returns (uint16) {
    return chainIds[index];
  }

  function _processGovernanceConfig(DelegatedGuardianPayload memory config) internal {
    if((config.threshold == 0 && config.keys.length != 0) || (config.threshold != 0 && config.keys.length == 0)) {
        revert InvalidConfig(config.chainId);
      }
      if(delegatedGuardianSets[config.chainId].length == 0) {
        chainIds.push(config.chainId);
      }
      delegatedGuardianSets[config.chainId].push(DelegatedGuardianSet({
        chainId: config.chainId,
        timestamp: uint32(block.timestamp),
        threshold: config.threshold,
        keys: config.keys
      }));
      emit ChainConfigSet(nextConfigIndex, config.chainId, config.threshold, config.keys);
  }

  function _decodeConfigPayload(bytes memory payload) private pure returns (ConfigPayload memory) {
    uint256 offset = 0;
    
    uint256 configIndex = payload.toUint256(offset);
    offset += 32;
    
    uint256 configsLength = payload.toUint8(offset);
    offset += 1;
    
    DelegatedGuardianPayload[] memory configs = new DelegatedGuardianPayload[](configsLength);
    
    for (uint256 i = 0; i < configsLength; i++) {
      configs[i].chainId = payload.toUint16(offset);
      offset += 2;
      
      configs[i].threshold = payload.toUint8(offset);
      offset += 1;
      
      uint256 keysLength = payload.toUint8(offset);
      offset += 1;
      
      configs[i].keys = new address[](keysLength);
      
      for (uint256 j = 0; j < keysLength; j++) {
        configs[i].keys[j] = payload.toAddress(offset);
        offset += 20;
      }
    }
    
    return ConfigPayload({
      configIndex: configIndex,
      configs: configs
    });
  }

  function _verifyGovernanceVAA(
      bytes memory encodedVM
  ) internal returns (IWormhole.VM memory parsedVM) {
      (IWormhole.VM memory vm, bool valid, string memory reason) =
          wormhole.parseAndVerifyVM(encodedVM);

      if (!valid) {
          revert(reason);
      }

      if (vm.emitterChainId != wormhole.governanceChainId()) {
          revert InvalidGovernanceChainId(vm.emitterChainId);
      }

      if (vm.emitterAddress != wormhole.governanceContract()) {
          revert InvalidGovernanceContract(vm.emitterAddress);
      }

      _replayProtect(vm.hash);

      return vm;
  }

  function _replayProtect(
      bytes32 digest
  ) internal {
      if (governanceActionsConsumed[digest]) {
          revert GovernanceActionAlreadyConsumed(digest);
      }
      governanceActionsConsumed[digest] = true;
  }
  /**
   * @dev Parses a VAA payload into a ConfigPayload
   * Adheres to the Wormhole governance packet standard:
   *   https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0002_governance_messaging.md
   *   As specified in the standard, chain ID is 0 for non-specific actions such as guardian set changes
   */
  function _parseConfigMessage(bytes memory _vaaPayload) private pure returns (bytes memory) {
    uint256 offset = 0;
    bytes32 module = _vaaPayload.toBytes32(offset);
    offset += 32;
    if(module != MODULE) {
      revert InvalidModule(module);
    }

    uint8 action = _vaaPayload.toUint8(offset);
    offset += 1;
    if(action != uint8(Action.SET_CONFIG)) {
      revert InvalidAction(action);
    }

    uint16 chain = _vaaPayload.toUint16(offset);
    offset += 2;
    if(chain != 0) {
      revert InvalidChainId(chain);
    }

    bytes memory payload = _vaaPayload.slice(offset, _vaaPayload.length - offset);

    return payload;
  }
}
