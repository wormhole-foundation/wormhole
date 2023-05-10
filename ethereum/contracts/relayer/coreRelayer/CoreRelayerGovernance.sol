// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import {IWormhole} from "../../interfaces/IWormhole.sol";
import {InvalidPayloadLength} from "../../interfaces/relayer/IWormholeRelayer.sol";
import {fromWormholeFormat} from "./Utils.sol";
import {BytesParsing} from "./BytesParsing.sol";
import {
  getGovernanceState,
  getRegisteredCoreRelayersState,
  getDefaultRelayProviderState
} from "./CoreRelayerStorage.sol";
import {CoreRelayerBase} from "./CoreRelayerBase.sol";

error GovernanceActionAlreadyConsumed(bytes32 hash);
error InvalidGovernanceVM(string reason);
error InvalidGovernanceChainId(uint16 parsed, uint16 expected);
error InvalidGovernanceContract(bytes32 parsed, bytes32 expected);

error InvalidPayloadChainId(uint16 parsed, uint16 expected);
error InvalidPayloadAction(uint8 parsed, uint8 expected);
error InvalidPayloadModule(bytes32 parsed, bytes32 expected);
error InvalidFork();
error ContractUpgradeFailed(bytes failure);

abstract contract CoreRelayerGovernance is CoreRelayerBase, ERC1967Upgrade {
//TODO AMO: the bytes32 payloads here are suspect - for EVM we only need 20 bytes
//           for other chains, might it not take more than just a single address?
//           Wouldn't it be safer to just allow arbitrary bytes at the end of the payload?

//governance payload structs (packed)
//   struct ContractUpgrade {
//     bytes32 module;
//     uint8 action;
//     uint16 chainId;
//     bytes32 newImplementation; //TODO AMO: why is this not an address?
//   }
//
//   struct RegisterChain {
//     bytes32 module;
//     uint8 action;
//     uint16 chainId; //TODO Why is this on this object? //TODO AMO: why indeed?
//     uint16 emitterChain;
//     bytes32 emitterAddress;
//   }
//
//   //This could potentially be combined with ContractUpgrade
//   struct UpdateDefaultProvider {
//     bytes32 module;
//     uint8 action;
//     uint16 chainId;
//     bytes32 newProvider; //TODO AMO: why is this not an address?
//   }

  //TODO AMO: What's this module business about? (and also the action business for that matter)
  bytes32 constant module = 0x000000000000000000000000000000000000000000436f726552656c61796572; 
  uint16 private constant WORMHOLE_CHAINID_UNSET = 0;

  //TODO AMO: Document that upgrading works by deploying a new contract that implements
  //  an external function executeUpgradeMigration() which enforces that it can only be called
  //  by address(this) (i.e. via codecall)
  //TODO AMO: Discuss upgrade mechanism and whether it actually covers all scenarios we might
  //  ever need.
  //  Currently there's no way to include:
  //  * gas tokens (i.e. the function is not payable, though one can selfdestruct value into the
  //      contract and then use that as part of executeUpgradeMigration if really desperate)
  //  * "dynamic migration data" - i.e. executeUpgradeMigration() takes no parameters (probably
  //       entirely a non-issue since the new contracts need to be deployed on a per-chain basis
  //       anyway, but better to be explicit about it anyway)
  function submitContractUpgrade(bytes memory encodedVm) external {
    address newImplementation = parseAndCheckContractUpgradeVm(encodedVm);

    _upgradeTo(newImplementation);

    //Now that our implementation points to the new contract, we can just call ourselves and have
    //  executeUpgradeMigration() use a simple check that msg.sender == address(this).
    //I.e. the new contract just has to implement:
    //  function executeUpgradeMigration() external view {
    //    assert(msg.sender == address(this));
    //    //migration code goes here
    //  }
    //  so we don't have to mess around with any "intialized" and "reinitialized" storage vars.
    (bool success, bytes memory revertData) =
      address(this).call(abi.encodeWithSignature("executeUpgradeMigration()"));

    if (!success)
      revert ContractUpgradeFailed(revertData);
  }

  function registerCoreRelayerContract(bytes memory encodedVm) external {
    (uint16 emitterChainId, bytes32 emitterAddress) =
      parseAndCheckRegisterCoreRelayerContractVm(encodedVm);
    
    getRegisteredCoreRelayersState().registeredCoreRelayers[emitterChainId] = emitterAddress;
  }

  function setDefaultRelayProvider(bytes memory encodedVm) external {
    address newProvider = parseAndCheckRegisterDefaultRelayProviderVm(encodedVm);
    
    getDefaultRelayProviderState().defaultRelayProvider = newProvider;
  }

  // ------------------------------------------- PRIVATE -------------------------------------------
  using BytesParsing for bytes;

  function parseAndCheckContractUpgradeVm(
    bytes memory encodedVm
  ) private returns (address newImplementation) {
    bytes memory payload = verifyAndConsumeGovernanceVM(encodedVm);
    //TODO AMO: what's action 2?
    uint offset = parseAndCheckPayloadHeader(payload, 2, false);
    
    bytes32 newImplementationWhFmt;
    (newImplementationWhFmt, offset) = payload.asBytes32Unchecked(offset);
    //fromWormholeFormat reverts if first 12 bytes aren't zero (i.e. if it's not an EVM address)
    newImplementation = fromWormholeFormat(newImplementationWhFmt);

    checkLength(payload, offset);
  }

  function parseAndCheckRegisterCoreRelayerContractVm(
    bytes memory encodedVm
  ) private returns (uint16 emitterChainId, bytes32 emitterAddress) {
    bytes memory payload = verifyAndConsumeGovernanceVM(encodedVm);
    //TODO AMO: what's action 1?
    uint offset = parseAndCheckPayloadHeader(payload, 1, true);
    
    (emitterChainId, offset) = payload.asUint16Unchecked(offset);
    (emitterAddress, offset) = payload.asBytes32Unchecked(offset);
    
    checkLength(payload, offset);
  }

  function parseAndCheckRegisterDefaultRelayProviderVm(
    bytes memory encodedVm
  ) private returns (address newProvider) {
    bytes memory payload = verifyAndConsumeGovernanceVM(encodedVm);
    //TODO AMO: what's action 4?
    uint offset = parseAndCheckPayloadHeader(payload, 4, true);

    bytes32 newProviderWhFmt;
    (newProviderWhFmt, offset) = payload.asBytes32Unchecked(offset);
    //fromWormholeFormat reverts if first 12 bytes aren't zero (i.e. if it's not an EVM address)
    newProvider = fromWormholeFormat(newProviderWhFmt);

    checkLength(payload, offset);
  }
  
  function verifyAndConsumeGovernanceVM(
    bytes memory encodedVm
  ) private returns (bytes memory payload) {
    (IWormhole.VM memory vm, bool valid, string memory reason) =
      getWormhole().parseAndVerifyVM(encodedVm);

    if (!valid)
      revert InvalidGovernanceVM(reason);

    //TODO AMO: Check assumption that wormhole core bridge and core relayer share governance!
    uint16 governanceChainId = getWormhole().governanceChainId();
    if (vm.emitterChainId != governanceChainId)
      revert InvalidGovernanceChainId(vm.emitterChainId, governanceChainId);
    
    bytes32 governanceContract = getWormhole().governanceContract();
    if (vm.emitterAddress != governanceContract)
      revert InvalidGovernanceContract(vm.emitterAddress, governanceContract);

    bool consumed = getGovernanceState().consumedGovernanceActions[vm.hash];
    if (consumed)
      revert GovernanceActionAlreadyConsumed(vm.hash);

    getGovernanceState().consumedGovernanceActions[vm.hash] = true;

    return vm.payload;
  }

  function parseAndCheckPayloadHeader(
    bytes memory encodedPayload,
    uint8 expectedAction,
    bool allowUnset
  ) private view returns (uint offset) {
    bytes32 parsedModule;
    (parsedModule, offset) = encodedPayload.asBytes32Unchecked(offset);
    if (parsedModule != module)
      revert InvalidPayloadModule(parsedModule, module);
    
    uint8 parsedAction;
    (parsedAction, offset) = encodedPayload.asUint8Unchecked(offset);
    if (parsedAction != expectedAction)
      revert InvalidPayloadAction(parsedAction, expectedAction);

    uint16 parsedChainId;
    (parsedChainId, offset) = encodedPayload.asUint16Unchecked(offset);
    if (!(parsedChainId == WORMHOLE_CHAINID_UNSET && allowUnset)) {
      if (getWormhole().isFork())
        revert InvalidFork();

      if (parsedChainId != getChainId())
        revert InvalidPayloadChainId(parsedChainId, getChainId());
    }
  }

  function checkLength(bytes memory payload, uint expected) private pure {
    if (payload.length != expected)
      revert InvalidPayloadLength(payload.length, expected);
  }
}
