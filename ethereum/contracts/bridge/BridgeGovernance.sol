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
    /// @notice Reverts when a `SetPauserAddresses` payload encodes a pauser/freezer/unpauser length
    ///         that is neither 0 (unassigned) nor 20 (the EVM native address size).
    error InvalidAddressLength();

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

    /// @notice Emitted when the pauser/freezer/unpauser addresses are updated via governance.
    /// @param pauser The address authorized to call pause().
    /// @param freezer The address authorized to call freeze().
    /// @param unpauser The address authorized to call unpause().
    event PauserAddressesSet(
        address indexed pauser,
        address indexed freezer,
        address indexed unpauser
    );

    /// @notice Set the pauser, freezer, and unpauser addresses via a `SetPauserAddresses`
    ///         (action 4) governance VAA.
    /// @dev Payload layout:
    ///        module(32) | action(1)=4 | chainId(2)
    ///      | pauserLen(1) | pauser[pauserLen]
    ///      | freezerLen(1) | freezer[freezerLen]
    ///      | unpauserLen(1) | unpauser[unpauserLen]
    ///
    ///      Each length must be either 20 (the EVM native address size) or 0 (role left
    ///      unassigned); any other length is rejected. An all-zero 20-byte address is treated as
    ///      equivalent to a zero-length address (also unassigned). Parsed inline (no separate
    ///      `parseSetPauserAddresses` / struct) to keep `BridgeImplementation` under the
    ///      24,576-byte EIP-170 limit. See the "Pausing" section of whitepapers/0003_token_bridge.md.
    function submitSetPauserAddresses(bytes memory encodedVM) public {
        IWormhole.VM memory vm = verifyGovernanceVM(encodedVM);

        setGovernanceActionConsumed(vm.hash);

        bytes memory payload = vm.payload;
        if (payload.toBytes32(0) != module) revert WrongModule();
        if (payload.toUint8(32) != 4) revert WrongAction();
        if (payload.toUint16(33) != chainId()) revert WrongChainId();

        uint index = 35;
        address newPauser;
        address newFreezer;
        address newUnpauser;
        (newPauser, index) = _parsePauserAddress(payload, index);
        (newFreezer, index) = _parsePauserAddress(payload, index);
        (newUnpauser, index) = _parsePauserAddress(payload, index);

        if (payload.length != index) revert WrongLength();

        setPauser(newPauser);
        setFreezer(newFreezer);
        setUnpauser(newUnpauser);

        emit PauserAddressesSet(newPauser, newFreezer, newUnpauser);
    }

    /// @dev Parse one length-prefixed address from a `SetPauserAddresses` payload at `index`.
    ///      Length must be 20 (EVM address) or 0 (unassigned → address(0)); any other length
    ///      reverts. Returns the parsed address and the advanced index. Shared by the three role
    ///      fields to keep `BridgeImplementation` under the EIP-170 limit.
    function _parsePauserAddress(bytes memory payload, uint index)
        internal
        pure
        returns (address addr, uint newIndex)
    {
        uint8 len = payload.toUint8(index);
        index += 1;
        if (len == 20) {
            addr = payload.toAddress(index);
            index += 20;
        } else if (len != 0) {
            revert InvalidAddressLength();
        }
        return (addr, index);
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

    /// @dev Verify a governance VM. Reverts on any failure path; returns the parsed VM on success.
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
