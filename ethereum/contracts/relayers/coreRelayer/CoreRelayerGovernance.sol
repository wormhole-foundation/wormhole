// contracts/Relayer.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "../libraries/external/BytesLib.sol";

import "./CoreRelayerGetters.sol";
import "./CoreRelayerSetters.sol";
import "./CoreRelayerStructs.sol";
import "./CoreRelayerMessages.sol";

import "../interfaces/IWormhole.sol";
import "./CoreRelayerLibrary.sol";

abstract contract CoreRelayerGovernance is
    CoreRelayerGetters,
    CoreRelayerSetters,
    CoreRelayerMessages,
    ERC1967Upgrade
{
    using BytesLib for bytes;

    error InvalidFork();
    error InvalidGovernanceVM(string reason);
    error WrongChainId(uint16 chainId);
    error InvalidChainId(uint16 chainId);
    error FailedToInitializeImplementation(string reason);

    event ContractUpgraded(address indexed oldContract, address indexed newContract);

    // "CoreRelayer" (left padded)
    bytes32 constant module = 0x000000000000000000000000000000000000000000436f726552656c61796572;

    function submitContractUpgrade(bytes memory _vm) public {
        if (isFork()) {
            revert InvalidFork();
        }

        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(_vm);
        if (!valid) {
            revert InvalidGovernanceVM(string(reason));
        }

        setConsumedGovernanceAction(vm.hash);

        CoreRelayerLibrary.ContractUpgrade memory contractUpgrade = CoreRelayerLibrary.parseUpgrade(vm.payload, module);
        if (contractUpgrade.chain != chainId()) {
            revert WrongChainId(contractUpgrade.chain);
        }

        upgradeImplementation(contractUpgrade.newContract);
    }

    function registerCoreRelayerContract(bytes memory vaa) public {
        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(vaa);
        if (!valid) {
            revert InvalidGovernanceVM(string(reason));
        }

        setConsumedGovernanceAction(vm.hash);

        CoreRelayerLibrary.RegisterChain memory rc = CoreRelayerLibrary.parseRegisterChain(vm.payload, module);

        if ((rc.chain != chainId() || isFork()) && rc.chain != 0) {
            revert InvalidChainId(rc.chain);
        }

        setRegisteredCoreRelayerContract(rc.emitterChain, rc.emitterAddress);
    }

    function setDefaultRelayProvider(bytes memory vaa) public {
        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(vaa);
        if (!valid) {
            revert InvalidGovernanceVM(string(reason));
        }

        setConsumedGovernanceAction(vm.hash);

        CoreRelayerLibrary.UpdateDefaultProvider memory provider =
            CoreRelayerLibrary.parseUpdateDefaultProvider(vm.payload, module);

        if ((provider.chain != chainId() || isFork()) && provider.chain != 0) {
            revert InvalidChainId(provider.chain);
        }

        setRelayProvider(provider.newProvider);
    }

    function upgradeImplementation(address newImplementation) internal {
        address currentImplementation = _getImplementation();

        _upgradeTo(newImplementation);

        // Call initialize function of the new implementation
        (bool success, bytes memory reason) = newImplementation.delegatecall(abi.encodeWithSignature("initialize()"));

        if (!success) {
            revert FailedToInitializeImplementation(string(reason));
        }

        emit ContractUpgraded(currentImplementation, newImplementation);
    }

    function verifyGovernanceVM(bytes memory encodedVM)
        internal
        view
        returns (IWormhole.VM memory parsedVM, bool isValid, string memory invalidReason)
    {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVM);

        if (!valid) {
            return (vm, valid, reason);
        }

        if (vm.emitterChainId != governanceChainId()) {
            return (vm, false, "wrong governance chain");
        }
        if (vm.emitterAddress != governanceContract()) {
            return (vm, false, "wrong governance contract");
        }

        if (governanceActionIsConsumed(vm.hash)) {
            return (vm, false, "governance action already consumed");
        }

        return (vm, true, "");
    }
}
