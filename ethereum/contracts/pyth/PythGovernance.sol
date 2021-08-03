// contracts/Bridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "../libraries/external/BytesLib.sol";

import "./PythGetters.sol";
import "./PythSetters.sol";
import "./PythStructs.sol";

import "../interfaces/IWormhole.sol";

contract PythGovernance is PythGetters, PythSetters, ERC1967Upgrade {
    using BytesLib for bytes;

    bytes32 constant module = 0x0000000000000000000000000000000000000000000000000000000050797468;

    // Execute a UpgradeContract governance message
    function upgrade(bytes memory encodedVM) public {
        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(encodedVM);
        require(valid, reason);

        setGovernanceActionConsumed(vm.hash);

        PythStructs.UpgradeContract memory implementation = parseContractUpgrade(vm.payload);

        require(implementation.module == module, "wrong module");
        require(implementation.chain == chainId(), "wrong chain id");

        upgradeImplementation(implementation.newContract);
    }

    function verifyGovernanceVM(bytes memory encodedVM) internal view returns (IWormhole.VM memory parsedVM, bool isValid, string memory invalidReason){
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVM);

        if(!valid){
            return (vm, valid, reason);
        }

        if (vm.emitterChainId != governanceChainId()) {
            return (vm, false, "wrong governance chain");
        }
        if (vm.emitterAddress != governanceContract()) {
            return (vm, false, "wrong governance contract");
        }

        if(governanceActionIsConsumed(vm.hash)){
            return (vm, false, "governance action already consumed");
        }

        return (vm, true, "");
    }

    event ContractUpgraded(address indexed oldContract, address indexed newContract);
    function upgradeImplementation(address newImplementation) internal {
        address currentImplementation = _getImplementation();

        _upgradeTo(newImplementation);

        // Call initialize function of the new implementation
        (bool success, bytes memory reason) = newImplementation.delegatecall(abi.encodeWithSignature("initialize()"));

        require(success, string(reason));

        emit ContractUpgraded(currentImplementation, newImplementation);
    }

    function parseContractUpgrade(bytes memory encodedUpgrade) public pure returns (PythStructs.UpgradeContract memory cu) {
        uint index = 0;

        cu.module = encodedUpgrade.toBytes32(index);
        index += 32;

        cu.action = encodedUpgrade.toUint8(index);
        index += 1;

        require(cu.action == 1, "invalid ContractUpgrade 1");

        cu.chain = encodedUpgrade.toUint16(index);
        index += 2;

        cu.newContract = address(uint160(uint256(encodedUpgrade.toBytes32(index))));
        index += 32;

        require(encodedUpgrade.length == index, "invalid ContractUpgrade 2");
    }
}
