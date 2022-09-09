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

    // Execute a RegisterChain governance message
    function registerChain(bytes memory encodedVM) public {
        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(encodedVM);
        require(valid, reason);

        setGovernanceActionConsumed(vm.hash);

        BridgeStructs.RegisterChain memory chain = parseRegisterChain(vm.payload);

        require((chain.chainId == chainId() && !isFork()) || chain.chainId == 0, "invalid chain id");
        require(bridgeContracts(chain.emitterChainID) == bytes32(0), "chain already registered");

        setBridgeImplementation(chain.emitterChainID, chain.emitterAddress);
    }

    // Execute a UpgradeContract governance message
    function upgrade(bytes memory encodedVM) public {
        require(!isFork(), "invalid fork");

        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(encodedVM);
        require(valid, reason);

        setGovernanceActionConsumed(vm.hash);

        BridgeStructs.UpgradeContract memory implementation = parseUpgrade(vm.payload);

        require(implementation.chainId == chainId(), "wrong chain id");

        upgradeImplementation(address(uint160(uint256(implementation.newContract))));
    }

    /**
    * @dev Updates the `chainId` and `evmChainId` on a forked chain via Governance VAA/VM
    */
    function submitRecoverChainId(bytes memory encodedVM) public {
        require(isFork(), "not a fork");

        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(encodedVM);
        require(valid, reason);

        setGovernanceActionConsumed(vm.hash);

        BridgeStructs.RecoverChainId memory rci = parseRecoverChainId(vm.payload);

        // Verify the VAA is for this chain
        require(rci.evmChainId == block.chainid, "invalid EVM Chain");

        // Update the chainIds
        setEvmChainId(rci.evmChainId);
        setChainId(rci.newChainId);
    }

    function verifyGovernanceVM(bytes memory encodedVM) internal view returns (IWormhole.VM memory parsedVM, bool isValid, string memory invalidReason){
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

    event ContractUpgraded(address indexed oldContract, address indexed newContract);

    function upgradeImplementation(address newImplementation) internal {
        address currentImplementation = _getImplementation();

        _upgradeTo(newImplementation);

        // Call initialize function of the new implementation
        (bool success, bytes memory reason) = newImplementation.delegatecall(abi.encodeWithSignature("initialize()"));

        require(success, string(reason));

        emit ContractUpgraded(currentImplementation, newImplementation);
    }

    function parseRegisterChain(bytes memory encoded) public pure returns (BridgeStructs.RegisterChain memory chain) {
        uint index = 0;

        // governance header

        chain.module = encoded.toBytes32(index);
        index += 32;
        require(chain.module == module, "wrong module");

        chain.action = encoded.toUint8(index);
        index += 1;
        require(chain.action == 1, "wrong action");

        chain.chainId = encoded.toUint16(index);
        index += 2;

        // payload

        chain.emitterChainID = encoded.toUint16(index);
        index += 2;

        chain.emitterAddress = encoded.toBytes32(index);
        index += 32;

        require(encoded.length == index, "wrong length");
    }

    function parseUpgrade(bytes memory encoded) public pure returns (BridgeStructs.UpgradeContract memory chain) {
        uint index = 0;

        // governance header

        chain.module = encoded.toBytes32(index);
        index += 32;
        require(chain.module == module, "wrong module");

        chain.action = encoded.toUint8(index);
        index += 1;
        require(chain.action == 2, "wrong action");

        chain.chainId = encoded.toUint16(index);
        index += 2;

        // payload

        chain.newContract = encoded.toBytes32(index);
        index += 32;

        require(encoded.length == index, "wrong length");
    }

    /// @dev Parse a recoverChainId (action 3) with minimal validation
    function parseRecoverChainId(bytes memory encodedRecoverChainId) public pure returns (BridgeStructs.RecoverChainId memory rci) {
        uint index = 0;

        rci.module = encodedRecoverChainId.toBytes32(index);
        index += 32;
        require(rci.module == module, "wrong module");

        rci.action = encodedRecoverChainId.toUint8(index);
        index += 1;
        require(rci.action == 3, "wrong action");

        rci.evmChainId = encodedRecoverChainId.toUint256(index);
        index += 32;

        rci.newChainId = encodedRecoverChainId.toUint16(index);
        index += 2;

        require(encodedRecoverChainId.length == index, "wrong length");
    }
}
