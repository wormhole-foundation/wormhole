// contracts/Governance.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "./Structs.sol";
import "./GovernanceStructs.sol";
import "./Messages.sol";
import "./Setters.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

abstract contract Governance is GovernanceStructs, Messages, Setters, ERC1967Upgrade {
    event ContractUpgraded(address indexed oldContract, address indexed newContract);
    event OwnershipTransfered(address indexed oldOwner, address indexed newOwner);

    // "Core" (left padded)
    bytes32 constant module = 0x00000000000000000000000000000000000000000000000000000000436f7265;

    function submitNewGuardianSet(bytes memory _vm) public {
        Structs.VM memory vm = parseVM(_vm);

        (bool isValid, string memory reason) = verifyGovernanceVM(vm);
        require(isValid, reason);

        GovernanceStructs.GuardianSetUpgrade memory upgrade = parseGuardianSetUpgrade(vm.payload);

        require(upgrade.module == module, "invalid Module");

        require(upgrade.newGuardianSet.keys.length > 0, "new guardian set is empty");
        require(upgrade.newGuardianSetIndex == getCurrentGuardianSetIndex() + 1, "index must increase in steps of 1");

        setGovernanceActionConsumed(vm.hash);

        expireGuardianSet(getCurrentGuardianSetIndex());
        storeGuardianSet(upgrade.newGuardianSet, upgrade.newGuardianSetIndex);
        updateGuardianSetIndex(upgrade.newGuardianSetIndex);
    }

    function upgradeImplementation(address newImplementation) public onlyOwner {
        address currentImplementation = _getImplementation();

        _upgradeTo(newImplementation);

        // Call initialize function of the new implementation
        (bool success, bytes memory reason) = newImplementation.delegatecall(abi.encodeWithSignature("initialize()"));

        require(success, string(reason));

        emit ContractUpgraded(currentImplementation, newImplementation);
    }

    function verifyGovernanceVM(Structs.VM memory vm) internal view returns (bool, string memory){
        // validate vm
        (bool isValid, string memory reason) = verifyVM(vm);
        if (!isValid){
            return (false, reason);
        }

        // only current guardianset can sign governance packets
        if (vm.guardianSetIndex != getCurrentGuardianSetIndex()) {
            return (false, "not signed by current guardian set");
        }

        // verify source
        if (uint16(vm.emitterChainId) != governanceChainId()) {
            return (false, "wrong governance chain");
        }
        if (vm.emitterAddress != governanceContract()) {
            return (false, "wrong governance contract");
        }

        // prevent re-entry
        if (governanceActionIsConsumed(vm.hash)){
            return (false, "governance action already consumed");
        }

        return (true, "");
    }

    function transferOwnership(address newOwner) public onlyOwner {
        require(newOwner != address(0), "new owner cannot be the zero address");

        address currentOwner = owner();
        
        setOwner(newOwner);

        emit OwnershipTransfered(currentOwner, newOwner);
    }

    modifier onlyOwner() {
        require(owner() == msg.sender, "caller is not the owner");
        _;
    }
}
