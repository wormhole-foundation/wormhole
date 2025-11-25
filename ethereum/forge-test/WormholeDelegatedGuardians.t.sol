pragma solidity ^0.8.0;

import "../contracts/Implementation.sol";
import "../contracts/Setup.sol";
import "../contracts/Wormhole.sol";
import "../contracts/delegated_guardians/WormholeDelegatedGuardians.sol";
import "forge-std/Test.sol";
import "forge-test/rv-helpers/TestUtils.sol";

contract TestWormholeDelegatedGuardians is TestUtils {
  using BytesLib for bytes;

  uint16  constant CHAINID = 2;
  uint256 constant EVMCHAINID = 1;
  bytes32 constant MODULE = 0x000000000000000000000000000044656C656761746564477561726469616E73;
  uint16 constant ACTION_CHAINID = 0;
  uint16 constant GOVERNANCE_CHAIN_ID = 1;
  bytes32 constant GOVERNANCE_CONTRACT = 0x0000000000000000000000000000000000000000000000000000000000000004;
  uint256 constant TEST_GUARDIAN_PK = 93941733246223705020089879371323733820373732307041878556247502674739205313440;

  bytes32 constant BAD_MODULE = 0x000000000000000000000000000044656C656761746564477561726469616E74;
  uint8 constant BAD_ACTION = 1;
  uint16 constant BAD_ACTION_CHAINID = 1;
  bytes32 constant BAD_GOVERNANCE_CONTRACT = 0x0000000000000000000000000000000000000000000000000000000000000005;
  uint16 constant BAD_GOVERNANCE_CHAINID = 9;
  uint256 constant BAD_GUARDIAN_PK = 93941733246223705020089879371323733820373732307041878556247502674739205313441;

  Wormhole proxy;
  Implementation impl;
  Setup setup;
  Setup proxiedSetup;
  WormholeDelegatedGuardians delegatedGuardians;

  address guardian1 = makeAddr("guardian1");
  address guardian2 = makeAddr("guardian2");
  address guardian3 = makeAddr("guardian3");
  address guardian4 = makeAddr("guardian4");
  address guardian5 = makeAddr("guardian5");
  address guardian6 = makeAddr("guardian6");
  address guardian7 = makeAddr("guardian7");
  address guardian8 = makeAddr("guardian8");
  address guardian9 = makeAddr("guardian9");

  uint16 CHAIN_ID_2 = 2;

  event ChainConfigSet(uint256 configIndex, uint16 chainId, uint8 threshold, address[] keys);


  function setUp() public {
    setup = new Setup();
    impl = new Implementation();
    proxy = new Wormhole(address(setup), bytes(""));

    address[] memory keys = new address[](1);
    keys[0] = 0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe; // vm.addr(testGuardian)

    proxiedSetup = Setup(address(proxy));

    vm.chainId(1);
    proxiedSetup.setup({
        implementation: address(impl),
        initialGuardians: keys,
        chainId: CHAINID,
        governanceChainId: 1,
        governanceContract: GOVERNANCE_CONTRACT,
        evmChainId: EVMCHAINID
    });

    
    delegatedGuardians = new WormholeDelegatedGuardians(address(proxy));
  }

  function testSubmitOneConfig(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = _buildSimpleConfig();
    bytes memory encodedPayload = _buildPayload(0, configs);

    (bytes memory _vm, bytes32 hash) = validVm(
      0,
      timestamp,
      nonce,
      GOVERNANCE_CHAIN_ID,
      GOVERNANCE_CONTRACT,
      sequence,
      consistencyLevel,
      encodedPayload,
      TEST_GUARDIAN_PK
    );

    bool consumedBefore = delegatedGuardians.governanceActionsConsumed(hash);

    vm.expectEmit(address(delegatedGuardians));
    emit ChainConfigSet(
      0,
      CHAIN_ID_2,
      2,
      configs[0].keys
    );
    
    delegatedGuardians.submitConfig(_vm);

    WormholeDelegatedGuardians.DelegatedGuardianSet[] memory storedConfigs = delegatedGuardians.getConfig();
    WormholeDelegatedGuardians.DelegatedGuardianSet[] memory historicalConfigs = delegatedGuardians.getHistoricalConfig(CHAIN_ID_2);
    WormholeDelegatedGuardians.DelegatedGuardianSet memory singleConfig = delegatedGuardians.getConfig(CHAIN_ID_2);
    uint256 historicalConfigLength = delegatedGuardians.getHistoricalConfigLength(CHAIN_ID_2);
    uint256 chainIdsLength = delegatedGuardians.chainIdsLength();

    assertEq(storedConfigs.length, 1);
    assertEq(storedConfigs[0].chainId, CHAIN_ID_2);
    assertEq(storedConfigs[0].threshold, 2);
    assertEq(storedConfigs[0].keys.length, 9);
    assertEq(storedConfigs[0].keys[0], guardian1);
    assertEq(storedConfigs[0].keys[1], guardian2);
    assertEq(singleConfig.chainId, CHAIN_ID_2);
    assertEq(singleConfig.threshold, 2);
    assertEq(singleConfig.keys.length, 9);
    assertEq(singleConfig.keys[0], guardian1);
    assertEq(singleConfig.keys[1], guardian2);
    assertEq(historicalConfigs.length, 1);
    assertEq(historicalConfigs[0].chainId, CHAIN_ID_2);
    assertEq(historicalConfigs[0].threshold, 2);
    assertEq(historicalConfigs[0].keys.length, 9);
    assertEq(historicalConfigs[0].keys[0], guardian1);
    assertEq(historicalConfigs[0].keys[1], guardian2);
    assertEq(consumedBefore, false);
    assertEq(delegatedGuardians.governanceActionsConsumed(hash), true);
    assertEq(historicalConfigLength, 1);
    assertEq(chainIdsLength, 1);
  }

  function testSubmitAndOverwriteToZero(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = _buildSimpleConfig();
    bytes memory encodedPayload = _buildPayload(0, configs);

    {
      (bytes memory _vm,) = validVm(
        0,
        timestamp,
        nonce,
        GOVERNANCE_CHAIN_ID,
        GOVERNANCE_CONTRACT,
        sequence,
        consistencyLevel,
        encodedPayload,
        TEST_GUARDIAN_PK
      );

      delegatedGuardians.submitConfig(_vm);
    }

    address[] memory keys2 = new address[](0);
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs2 = new WormholeDelegatedGuardians.DelegatedGuardianPayload[](1);
    configs2[0] = WormholeDelegatedGuardians.DelegatedGuardianPayload({
      chainId: CHAIN_ID_2,
      threshold: 0,
      keys: keys2
    });

    bytes memory encodedPayload2 = _buildPayload(1, configs2);
    {
      (bytes memory _vm2,) = validVm(
        0,
        timestamp,
        nonce,
        GOVERNANCE_CHAIN_ID,
        GOVERNANCE_CONTRACT,
        sequence,
        consistencyLevel,
        encodedPayload2,
        TEST_GUARDIAN_PK
      );

      delegatedGuardians.submitConfig(_vm2);
    }

    WormholeDelegatedGuardians.DelegatedGuardianSet[] memory storedConfigs = delegatedGuardians.getConfig();
    WormholeDelegatedGuardians.DelegatedGuardianSet[] memory historicalConfigs = delegatedGuardians.getHistoricalConfig(CHAIN_ID_2);
    uint256 historicalConfigLength = delegatedGuardians.getHistoricalConfigLength(CHAIN_ID_2);
    WormholeDelegatedGuardians.DelegatedGuardianSet memory historicalConfig_0 = delegatedGuardians.getHistoricalConfig(CHAIN_ID_2, 0);
    WormholeDelegatedGuardians.DelegatedGuardianSet memory historicalConfig_1 = delegatedGuardians.getHistoricalConfig(CHAIN_ID_2, 1);
    uint256 chainIdsLength = delegatedGuardians.chainIdsLength();

    assertEq(storedConfigs.length, 1);
    assertEq(storedConfigs[0].chainId, CHAIN_ID_2);
    assertEq(storedConfigs[0].threshold, 0);
    assertEq(storedConfigs[0].keys.length, 0);
    assertEq(historicalConfigs.length, 2);
    assertEq(historicalConfigs[0].chainId, CHAIN_ID_2);
    assertEq(historicalConfigs[0].threshold, 2);
    assertEq(historicalConfigs[0].keys.length, 9);
    assertEq(historicalConfigs[0].keys[0], guardian1);
    assertEq(historicalConfigs[0].keys[1], guardian2);
    assertEq(historicalConfigs[0].keys[2], guardian3);
    assertEq(historicalConfigs[0].keys[3], guardian4);
    assertEq(historicalConfigs[0].keys[4], guardian5);
    assertEq(historicalConfigs[0].keys[5], guardian6);
    assertEq(historicalConfigs[0].keys[6], guardian7);
    assertEq(historicalConfigs[0].keys[7], guardian8);
    assertEq(historicalConfigs[0].keys[8], guardian9);
    assertEq(historicalConfigs[1].chainId, CHAIN_ID_2);
    assertEq(historicalConfigs[1].threshold, 0);
    assertEq(historicalConfigs[1].keys.length, 0);
    assertEq(historicalConfig_0.chainId, CHAIN_ID_2);
    assertEq(historicalConfig_0.threshold, 2);
    assertEq(historicalConfig_0.keys.length, 9);
    assertEq(historicalConfig_0.keys[0], guardian1);
    assertEq(historicalConfig_0.keys[1], guardian2);
    assertEq(historicalConfig_0.keys[2], guardian3);
    assertEq(historicalConfig_0.keys[3], guardian4);
    assertEq(historicalConfig_0.keys[4], guardian5);
    assertEq(historicalConfig_0.keys[5], guardian6);
    assertEq(historicalConfig_0.keys[6], guardian7);
    assertEq(historicalConfig_0.keys[7], guardian8);
    assertEq(historicalConfig_0.keys[8], guardian9);
    assertEq(historicalConfig_1.chainId, CHAIN_ID_2);
    assertEq(historicalConfig_1.threshold, 0);
    assertEq(historicalConfig_1.keys.length, 0);
    assertEq(historicalConfigLength, 2);
    assertEq(chainIdsLength, 1);
  }

  function testSubmit_50_Configs(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = new WormholeDelegatedGuardians.DelegatedGuardianPayload[](50);
    for(uint16 i = 0; i < 50; i++) {
      configs[i] = _buildSimpleConfig(i)[0];
    }

    bytes memory encodedPayload = _buildPayload(0, configs);

    (bytes memory _vm, bytes32 hash) = validVm(
      0,
      timestamp,
      nonce,
      GOVERNANCE_CHAIN_ID,
      GOVERNANCE_CONTRACT,
      sequence,
      consistencyLevel,
      encodedPayload,
      TEST_GUARDIAN_PK
    );

    for(uint16 i = 0; i < 50; i++) {
      vm.expectEmit(address(delegatedGuardians));
      emit ChainConfigSet(
        0,
        configs[i].chainId,
        configs[i].threshold,
        configs[i].keys
      );
    }
    delegatedGuardians.submitConfig(_vm);

    WormholeDelegatedGuardians.DelegatedGuardianSet[] memory storedConfigs = delegatedGuardians.getConfig();
    uint256 historicalConfigLength = delegatedGuardians.getHistoricalConfigLength(CHAIN_ID_2);
    uint256 chainIdsLength = delegatedGuardians.chainIdsLength();
    uint16[] memory chainIds = delegatedGuardians.getChainIds();

    assertEq(storedConfigs.length, 50);
    assertEq(historicalConfigLength, 1);
    assertEq(chainIdsLength, 50);
    for(uint16 i = 0; i < 50; i++) {
      assertEq(storedConfigs[i].chainId, configs[i].chainId);
      assertEq(storedConfigs[i].threshold, configs[i].threshold);
      assertEq(storedConfigs[i].keys.length, configs[i].keys.length);
      for(uint16 j = 0; j < configs[i].keys.length; j++) {
        assertEq(storedConfigs[i].keys[j], configs[i].keys[j]);
      }
      assertEq(chainIds[i], configs[i].chainId);
      uint16 chainId = delegatedGuardians.getChainId(i);
      assertEq(chainId, configs[i].chainId);
    }
  }

  function testSubmitReplayProtection(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = _buildSimpleConfig();
    bytes memory encodedPayload = _buildPayload(0, configs);

    (bytes memory _vm, bytes32 hash) = validVm(
      0,
      timestamp,
      nonce,
      GOVERNANCE_CHAIN_ID,
      GOVERNANCE_CONTRACT,
      sequence,
      consistencyLevel,
      encodedPayload,
      TEST_GUARDIAN_PK
    );

    delegatedGuardians.submitConfig(_vm);

    vm.expectRevert(
      abi.encodeWithSelector(WormholeDelegatedGuardians.GovernanceActionAlreadyConsumed.selector, hash)
    );
    delegatedGuardians.submitConfig(_vm);
  }

  function testSubmitBadModule(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = _buildSimpleConfig();
    bytes memory encodedPayload = _buildPayload(
      0,
      configs,
      BAD_MODULE, // invalid module
      uint8(WormholeDelegatedGuardians.Action.SET_CONFIG),
      ACTION_CHAINID
    );

    (bytes memory _vm,) = validVm(
      0,
      timestamp,
      nonce,
      GOVERNANCE_CHAIN_ID,
      GOVERNANCE_CONTRACT,
      sequence,
      consistencyLevel,
      encodedPayload,
      TEST_GUARDIAN_PK
    );
    
    vm.expectRevert(
      abi.encodeWithSelector(WormholeDelegatedGuardians.InvalidModule.selector, BAD_MODULE)
    );
    delegatedGuardians.submitConfig(_vm);
  }

  function testSubmitBadAction(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = _buildSimpleConfig();
    bytes memory encodedPayload = _buildPayload(
      0,
      configs,
      MODULE,
      BAD_ACTION, // invalid action
      ACTION_CHAINID
    );

    (bytes memory _vm,) = validVm(
      0,
      timestamp,
      nonce,
      GOVERNANCE_CHAIN_ID,
      GOVERNANCE_CONTRACT,
      sequence,
      consistencyLevel,
      encodedPayload,
      TEST_GUARDIAN_PK
    );
    
    vm.expectRevert(
      abi.encodeWithSelector(WormholeDelegatedGuardians.InvalidAction.selector, BAD_ACTION)
    );
    delegatedGuardians.submitConfig(_vm);
  }

  function testSubmitBadChainId(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = _buildSimpleConfig();
    bytes memory encodedPayload = _buildPayload(
      0,
      configs,
      MODULE,
      uint8(WormholeDelegatedGuardians.Action.SET_CONFIG),
      BAD_ACTION_CHAINID // invalid chain id
    );

    (bytes memory _vm,) = validVm(
      0,
      timestamp,
      nonce,
      GOVERNANCE_CHAIN_ID,
      GOVERNANCE_CONTRACT,
      sequence,
      consistencyLevel,
      encodedPayload,
      TEST_GUARDIAN_PK
    );
    
    vm.expectRevert(
      abi.encodeWithSelector(WormholeDelegatedGuardians.InvalidChainId.selector, BAD_ACTION_CHAINID)
    );
    delegatedGuardians.submitConfig(_vm);
  }

  function testInvalidGovernanceChainId(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = _buildSimpleConfig();
    bytes memory encodedPayload = _buildPayload(0, configs);

    (bytes memory _vm,) = validVm(
      0,
      timestamp,
      nonce,
      BAD_GOVERNANCE_CHAINID, // invalid governance chain id
      GOVERNANCE_CONTRACT,
      sequence,
      consistencyLevel,
      encodedPayload,
      TEST_GUARDIAN_PK
    );
    
    vm.expectRevert(
      abi.encodeWithSelector(WormholeDelegatedGuardians.InvalidGovernanceChainId.selector, BAD_GOVERNANCE_CHAINID)
    );
    delegatedGuardians.submitConfig(_vm);
  }

  function testInvalidGovernanceContract(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = _buildSimpleConfig();
    bytes memory encodedPayload = _buildPayload(0, configs);

    (bytes memory _vm,) = validVm(
      0,
      timestamp,
      nonce,
      GOVERNANCE_CHAIN_ID,
      BAD_GOVERNANCE_CONTRACT, // invalid contract
      sequence,
      consistencyLevel,
      encodedPayload,
      TEST_GUARDIAN_PK
    );
    
    vm.expectRevert(
      abi.encodeWithSelector(WormholeDelegatedGuardians.InvalidGovernanceContract.selector, BAD_GOVERNANCE_CONTRACT)
    );
    delegatedGuardians.submitConfig(_vm);
  }

  function testSubmitBadGuardianSignature(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = _buildSimpleConfig();
    bytes memory encodedPayload = _buildPayload(0, configs);

    (bytes memory _vm,) = validVm(
      0,
      timestamp,
      nonce,
      GOVERNANCE_CHAIN_ID,
      GOVERNANCE_CONTRACT,
      sequence,
      consistencyLevel,
      encodedPayload,
      BAD_GUARDIAN_PK
    );

    vm.expectRevert("VM signature invalid");
    delegatedGuardians.submitConfig(_vm);
  }

  function testSubmitBadIndex(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = _buildSimpleConfig();
    bytes memory encodedPayload = _buildPayload(1, configs);

    (bytes memory _vm,) = validVm(
      0,
      timestamp,
      nonce,
      GOVERNANCE_CHAIN_ID,
      GOVERNANCE_CONTRACT,
      sequence,
      consistencyLevel,
      encodedPayload,
      TEST_GUARDIAN_PK
    );
    
    vm.expectRevert(
      abi.encodeWithSelector(WormholeDelegatedGuardians.InvalidNextConfigIndex.selector, 1)
    );
    delegatedGuardians.submitConfig(_vm);
  }

  function testSubmitBadConfigurationEmptyKeysNonEmptyThreshold(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    address[] memory keys = new address[](0);
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = new WormholeDelegatedGuardians.DelegatedGuardianPayload[](1);
    configs[0] = WormholeDelegatedGuardians.DelegatedGuardianPayload({
      chainId: CHAIN_ID_2,
      threshold: 2,
      keys: keys
    });
    bytes memory encodedPayload = _buildPayload(0, configs);

    (bytes memory _vm,) = validVm(
      0,
      timestamp,
      nonce,
      GOVERNANCE_CHAIN_ID,
      GOVERNANCE_CONTRACT,
      sequence,
      consistencyLevel,
      encodedPayload,
      TEST_GUARDIAN_PK
    );
    
    vm.expectRevert(
      abi.encodeWithSelector(WormholeDelegatedGuardians.InvalidConfig.selector, CHAIN_ID_2)
    );
    delegatedGuardians.submitConfig(_vm);
  }

  function testSubmitBadConfigurationEmptyThresholdNonEmptyKeys(
    uint32 timestamp,
    uint32 nonce,
    uint64 sequence,
    uint8 consistencyLevel
  ) public {
    address[] memory keys = new address[](2);
    keys[0] = guardian1;
    keys[1] = guardian2;
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = new WormholeDelegatedGuardians.DelegatedGuardianPayload[](1);
    configs[0] = WormholeDelegatedGuardians.DelegatedGuardianPayload({
      chainId: CHAIN_ID_2,
      threshold: 0,
      keys: keys
    });
    bytes memory encodedPayload = _buildPayload(0, configs);

    (bytes memory _vm,) = validVm(
      0,
      timestamp,
      nonce,
      GOVERNANCE_CHAIN_ID,
      GOVERNANCE_CONTRACT,
      sequence,
      consistencyLevel,
      encodedPayload,
      TEST_GUARDIAN_PK
    );
    
    vm.expectRevert(
      abi.encodeWithSelector(WormholeDelegatedGuardians.InvalidConfig.selector, CHAIN_ID_2)
    );
    delegatedGuardians.submitConfig(_vm);
  }

  function _buildSimpleConfig() private view returns (WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory) {
    address[] memory keys = new address[](9);
    keys[0] = guardian1;
    keys[1] = guardian2;
    keys[2] = guardian3;
    keys[3] = guardian4;
    keys[4] = guardian5;
    keys[5] = guardian6;
    keys[6] = guardian7;
    keys[7] = guardian8;
    keys[8] = guardian9;

    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = new WormholeDelegatedGuardians.DelegatedGuardianPayload[](1);
    configs[0] = WormholeDelegatedGuardians.DelegatedGuardianPayload({
      chainId: CHAIN_ID_2,
      threshold: 2,
      keys: keys
    });
    return configs;
  }

  function _buildSimpleConfig(uint16 chainId) private view returns (WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory) {
    address[] memory keys = new address[](9);
    keys[0] = guardian1;
    keys[1] = guardian2;
    keys[2] = guardian3;
    keys[3] = guardian4;
    keys[4] = guardian5;
    keys[5] = guardian6;
    keys[6] = guardian7;
    keys[7] = guardian8;
    keys[8] = guardian9;

    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs = new WormholeDelegatedGuardians.DelegatedGuardianPayload[](1);
    configs[0] = WormholeDelegatedGuardians.DelegatedGuardianPayload({
      chainId: chainId,
      threshold: 2,
      keys: keys
    });
    return configs;
  }

  function _buildPayload(
    uint256 configIndex,
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs
  ) private pure returns (bytes memory) {
    return _buildPayload(
      configIndex,
      configs,
      MODULE,
      uint8(WormholeDelegatedGuardians.Action.SET_CONFIG),
      ACTION_CHAINID
    );
  }

  function _buildPayload(
    uint256 configIndex,
    WormholeDelegatedGuardians.DelegatedGuardianPayload[] memory configs,
    bytes32 module,
    uint8 action,
    uint16 actionChainId
  ) private pure returns (bytes memory) {
      bytes memory encodedPayload = abi.encodePacked(
      module,
      action,
      actionChainId,
      configIndex,
      uint8(configs.length)
    );
    for(uint256 i = 0; i < configs.length; i++) {
      bytes memory encodedKeys = abi.encodePacked(uint8(configs[i].keys.length));

      for(uint256 j = 0; j < configs[i].keys.length; j++) {
        encodedKeys = encodedKeys.concat(abi.encodePacked(configs[i].keys[j]));
      }

      bytes memory encodedConfig = abi.encodePacked(
        uint16(configs[i].chainId),
        uint8(configs[i].threshold),
        encodedKeys
      );

      encodedPayload = encodedPayload.concat(encodedConfig);
    }
    return encodedPayload;
  }
}
