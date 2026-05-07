// contracts/Bridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "../libraries/external/BytesLib.sol";

import "./BridgeGetters.sol";
import "./BridgeSetters.sol";
import "./BridgeStructs.sol";

import "./token/Token.sol";
import "./token/TokenImplementation.sol";

import "../interfaces/IWormhole.sol";

contract BridgeGovernance is BridgeGetters, BridgeSetters, ERC1967Upgrade {
    using BytesLib for bytes;

    // "TokenBridge" (left padded)
    bytes32 constant module = 0x000000000000000000000000000000000000000000546f6b656e427269646765;

    /// @dev Custom errors are used in place of revert strings to keep `BridgeImplementation` under
    ///      the 24,576-byte EIP-170 limit.
    error InvalidChainId();
    error ChainAlreadyRegistered();
    error WrongChainId();
    error WrongLength();
    error WrongModule();
    error WrongAction();
    error InvalidFork();
    error NotAFork();
    error InvalidEVMChain();
    error WrongGovernanceChain();
    error WrongGovernanceContract();
    error GovernanceActionConsumed();
    error InitializeFailed(bytes reason);

    // Execute a RegisterChain governance message
    function registerChain(bytes memory encodedVM) public {
        IWormhole.VM memory vm = verifyGovernanceVM(encodedVM);

        setGovernanceActionConsumed(vm.hash);

        BridgeStructs.RegisterChain memory chain = parseRegisterChain(vm.payload);

        if (!((chain.chainId == chainId() && !isFork()) || chain.chainId == 0)) revert InvalidChainId();
        if (bridgeContracts(chain.emitterChainID) != bytes32(0)) revert ChainAlreadyRegistered();

        setBridgeImplementation(chain.emitterChainID, chain.emitterAddress);
    }

    // Execute a UpgradeContract governance message
    function upgrade(bytes memory encodedVM) public {
        if (isFork()) revert InvalidFork();

        IWormhole.VM memory vm = verifyGovernanceVM(encodedVM);

        setGovernanceActionConsumed(vm.hash);

        BridgeStructs.UpgradeContract memory implementation = parseUpgrade(vm.payload);

        if (implementation.chainId != chainId()) revert WrongChainId();

        upgradeImplementation(address(uint160(uint256(implementation.newContract))));
    }

    /// @notice Emitted when the pauser/unpauser addresses are updated via governance.
    /// @param pauser The address authorized to call pause().
    /// @param unpauser The address authorized to call unpause().
    event PauserAddressesSet(address indexed pauser, address indexed unpauser);

    /// @notice Set the pauser and unpauser addresses via a SetPauserAddressesEvm (action 4) governance VAA.
    /// @dev Payload layout: module(32) | action(1)=4 | chainId(2) | pauser(20) | unpauser(20). Parsed
    ///      inline (no separate `parseSetPauserAddresses` / struct) to keep `BridgeImplementation` under
    ///      the 24,576-byte EIP-170 limit. See whitepapers/0018_pauser.md.
    function submitSetPauserAddresses(bytes memory encodedVM) public {
        IWormhole.VM memory vm = verifyGovernanceVM(encodedVM);

        setGovernanceActionConsumed(vm.hash);

        bytes memory payload = vm.payload;
        if (payload.length != 75) revert WrongLength();
        if (payload.toBytes32(0) != module) revert WrongModule();
        if (payload.toUint8(32) != 4) revert WrongAction();
        if (payload.toUint16(33) != chainId()) revert WrongChainId();

        address newPauser = payload.toAddress(35);
        address newUnpauser = payload.toAddress(55);

        setPauser(newPauser);
        setUnpauser(newUnpauser);

        emit PauserAddressesSet(newPauser, newUnpauser);
    }

    /**
    * @dev Updates the `chainId` and `evmChainId` on a forked chain via Governance VAA/VM
    */
    function submitRecoverChainId(bytes memory encodedVM) public {
        if (!isFork()) revert NotAFork();

        IWormhole.VM memory vm = verifyGovernanceVM(encodedVM);

        setGovernanceActionConsumed(vm.hash);

        BridgeStructs.RecoverChainId memory rci = parseRecoverChainId(vm.payload);

        // Verify the VAA is for this chain
        if (rci.evmChainId != block.chainid) revert InvalidEVMChain();

        // Update the chainIds
        setEvmChainId(rci.evmChainId);
        setChainId(rci.newChainId);
    }

    /// @dev Reverts directly on any failure path. Returns the parsed VM on success. Callers no longer
    ///      need to unpack a `(vm, valid, reason)` tuple — saves bytecode at every call site.
    function verifyGovernanceVM(bytes memory encodedVM) internal view returns (IWormhole.VM memory) {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVM);
        // Forwards the dynamic Wormhole core reason; cheaper than encoding a parameterized error.
        require(valid, reason);

        if (vm.emitterChainId != governanceChainId()) revert WrongGovernanceChain();
        if (vm.emitterAddress != governanceContract()) revert WrongGovernanceContract();
        if (governanceActionIsConsumed(vm.hash)) revert GovernanceActionConsumed();

        return vm;
    }

    event ContractUpgraded(address indexed oldContract, address indexed newContract);

    function upgradeImplementation(address newImplementation) internal {
        address currentImplementation = _getImplementation();

        _upgradeTo(newImplementation);

        // Call initialize function of the new implementation
        (bool success, bytes memory reason) = newImplementation.delegatecall(abi.encodeWithSignature("initialize()"));

        if (!success) revert InitializeFailed(reason);

        emit ContractUpgraded(currentImplementation, newImplementation);
    }

    function parseRegisterChain(bytes memory encoded) public pure returns (BridgeStructs.RegisterChain memory chain) {
        uint index = 0;

        // governance header

        chain.module = encoded.toBytes32(index);
        index += 32;
        if (chain.module != module) revert WrongModule();

        chain.action = encoded.toUint8(index);
        index += 1;
        if (chain.action != 1) revert WrongAction();

        chain.chainId = encoded.toUint16(index);
        index += 2;

        // payload

        chain.emitterChainID = encoded.toUint16(index);
        index += 2;

        chain.emitterAddress = encoded.toBytes32(index);
        index += 32;

        if (encoded.length != index) revert WrongLength();
    }

    function parseUpgrade(bytes memory encoded) public pure returns (BridgeStructs.UpgradeContract memory chain) {
        uint index = 0;

        // governance header

        chain.module = encoded.toBytes32(index);
        index += 32;
        if (chain.module != module) revert WrongModule();

        chain.action = encoded.toUint8(index);
        index += 1;
        if (chain.action != 2) revert WrongAction();

        chain.chainId = encoded.toUint16(index);
        index += 2;

        // payload

        chain.newContract = encoded.toBytes32(index);
        index += 32;

        if (encoded.length != index) revert WrongLength();
    }

/// @dev Parse a recoverChainId (action 3) with minimal validation
    function parseRecoverChainId(bytes memory encodedRecoverChainId) public pure returns (BridgeStructs.RecoverChainId memory rci) {
        uint index = 0;

        rci.module = encodedRecoverChainId.toBytes32(index);
        index += 32;
        if (rci.module != module) revert WrongModule();

        rci.action = encodedRecoverChainId.toUint8(index);
        index += 1;
        if (rci.action != 3) revert WrongAction();

        rci.evmChainId = encodedRecoverChainId.toUint256(index);
        index += 32;

        rci.newChainId = encodedRecoverChainId.toUint16(index);
        index += 2;

        if (encodedRecoverChainId.length != index) revert WrongLength();
    }
}
