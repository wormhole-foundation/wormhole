// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

import "../../libraries/external/BytesLib.sol";
import "../../interfaces/relayer/IForwardWrapper.sol";
import "../../interfaces/IWormhole.sol";
import "./CoreRelayerState.sol";
import "../../interfaces/relayer/IForwardInstructionViewer.sol";

contract CoreRelayerLibrary is CoreRelayerState, ERC1967Upgrade {
    using BytesLib for bytes;

    //structs, consts, errors, events
    struct ContractUpgrade {
        bytes32 module;
        uint8 action;
        uint16 chain;
        address newContract;
    }

    struct RegisterChain {
        bytes32 module;
        uint8 action;
        uint16 chain;
        uint16 emitterChain;
        bytes32 emitterAddress;
    }

    struct RecoverChainId {
        bytes32 module;
        uint8 action;
        uint256 evmChainId;
        uint16 newChainId;
    }

    struct UpdateDefaultProvider {
        bytes32 module;
        uint8 action;
        uint16 chain;
        address newProvider;
    }

    bytes32 constant module = 0x000000000000000000000000000000000000000000436f726552656c61796572;
    IForwardInstructionViewer public immutable forwardInstructionViewer;
    IWormhole immutable wormhole;

    error InvalidFork();
    error NotAFork();
    error InvalidGovernanceVM(string reason);
    error WrongChainId(uint16 chainId);
    error InvalidChainId(uint16 chainId);
    error FailedToInitializeImplementation(string reason);
    error WrongModule(bytes32 module);
    error InvalidContractUpgradeAction(uint8 action);
    error InvalidContractUpgradeLength(uint256 length);
    error InvalidRegisterChainAction(uint8);
    error InvalidRegisterChainLength(uint256);
    error InvalidDefaultProviderAction(uint8);
    error InvalidDefaultProviderLength(uint256);
    error InvalidRecoverChainAction(uint8);
    error InvalidRecoverChainLength(uint256);
    error RequesterNotCoreRelayer();
    error InvalidEvmChainId();

    event ContractUpgraded(address indexed oldContract, address indexed newContract);

    //This modifier is used to ensure that only the wormhole relayer can call the functions in this contract via delegate call
    modifier onlyWormholeRelayer() {
        if (address(this) != address(forwardInstructionViewer)) {
            revert RequesterNotCoreRelayer();
        }
        _;
    }

    constructor(address _wormholeRelayer, address _wormhole) {
        forwardInstructionViewer = IForwardInstructionViewer(_wormholeRelayer);
        wormhole = IWormhole(_wormhole);
    }

    //external functions
    function submitContractUpgrade(bytes memory _vm) external onlyWormholeRelayer {
        if (isFork()) {
            revert InvalidFork();
        }

        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(_vm);
        if (!valid) {
            revert InvalidGovernanceVM(string(reason));
        }

        setConsumedGovernanceAction(vm.hash);

        ContractUpgrade memory contractUpgrade = parseUpgrade(vm.payload);
        if (contractUpgrade.chain != chainId()) {
            revert WrongChainId(contractUpgrade.chain);
        }

        upgradeImplementation(contractUpgrade.newContract);
    }

    function registerCoreRelayerContract(bytes memory vaa) external onlyWormholeRelayer {
        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(vaa);
        if (!valid) {
            revert InvalidGovernanceVM(string(reason));
        }

        setConsumedGovernanceAction(vm.hash);

        RegisterChain memory rc = parseRegisterChain(vm.payload);

        if ((rc.chain != chainId() || isFork()) && rc.chain != 0) {
            revert InvalidChainId(rc.chain);
        }

        setRegisteredCoreRelayerContract(rc.emitterChain, rc.emitterAddress);
    }

    function submitRecoverChainId(bytes memory encodedVM) public {
        if (!isFork()) {
            revert NotAFork();
        }

        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(encodedVM);
        require(valid, reason);

        setConsumedGovernanceAction(vm.hash);

        RecoverChainId memory rci = parseRecoverChainId(vm.payload);

        // Update the chainIds
        setEvmChainId(rci.evmChainId);
        setChainId(rci.newChainId);
    }

    function setDefaultRelayProvider(bytes memory vaa) external onlyWormholeRelayer {
        (IWormhole.VM memory vm, bool valid, string memory reason) = verifyGovernanceVM(vaa);
        if (!valid) {
            revert InvalidGovernanceVM(string(reason));
        }

        setConsumedGovernanceAction(vm.hash);

        UpdateDefaultProvider memory provider = parseUpdateDefaultProvider(vm.payload);

        if ((provider.chain != chainId() || isFork()) && provider.chain != 0) {
            revert InvalidChainId(provider.chain);
        }

        setRelayProvider(provider.newProvider);
    }

    //parser functions
    function parseUpgrade(bytes memory encodedUpgrade)
        internal
        pure
        returns (ContractUpgrade memory cu)
    {
        uint256 index = 0;

        cu.module = encodedUpgrade.toBytes32(index);
        index += 32;

        if (cu.module != module) {
            revert WrongModule(cu.module);
        }

        cu.action = encodedUpgrade.toUint8(index);
        index += 1;

        if (cu.action != 2) {
            revert InvalidContractUpgradeAction(cu.action);
        }

        cu.chain = encodedUpgrade.toUint16(index);
        index += 2;

        cu.newContract = address(uint160(uint256(encodedUpgrade.toBytes32(index))));
        index += 32;

        if (encodedUpgrade.length != index) {
            revert InvalidContractUpgradeLength(encodedUpgrade.length);
        }
    }

    function parseRegisterChain(bytes memory encodedRegistration)
        internal
        pure
        returns (RegisterChain memory registerChain)
    {
        uint256 index = 0;

        registerChain.module = encodedRegistration.toBytes32(index);
        index += 32;

        if (registerChain.module != module) {
            revert WrongModule(registerChain.module);
        }

        registerChain.action = encodedRegistration.toUint8(index);
        index += 1;

        registerChain.chain = encodedRegistration.toUint16(index);
        index += 2;

        if (registerChain.action != 1) {
            revert InvalidRegisterChainAction(registerChain.action);
        }

        registerChain.emitterChain = encodedRegistration.toUint16(index);
        index += 2;

        registerChain.emitterAddress = encodedRegistration.toBytes32(index);
        index += 32;

        if (encodedRegistration.length != index) {
            revert InvalidRegisterChainLength(encodedRegistration.length);
        }
    }

    function parseUpdateDefaultProvider(bytes memory encodedDefaultProvider)
        internal
        pure
        returns (UpdateDefaultProvider memory defaultProvider)
    {
        uint256 index = 0;

        defaultProvider.module = encodedDefaultProvider.toBytes32(index);
        index += 32;

        if (defaultProvider.module != module) {
            revert WrongModule(defaultProvider.module);
        }

        defaultProvider.action = encodedDefaultProvider.toUint8(index);
        index += 1;

        if (defaultProvider.action != 4) {
            revert InvalidDefaultProviderAction(defaultProvider.action);
        }

        defaultProvider.chain = encodedDefaultProvider.toUint16(index);
        index += 2;

        defaultProvider.newProvider =
            address(uint160(uint256(encodedDefaultProvider.toBytes32(index))));
        index += 32;

        if (encodedDefaultProvider.length != index) {
            revert InvalidDefaultProviderLength(encodedDefaultProvider.length);
        }
    }

    function parseRecoverChainId(bytes memory encodedRecoverChainId)
        internal
        pure
        returns (RecoverChainId memory rci)
    {
        uint256 index = 0;

        rci.module = encodedRecoverChainId.toBytes32(index);
        index += 32;
        if (rci.module != module) {
            revert WrongModule(rci.module);
        }

        rci.action = encodedRecoverChainId.toUint8(index);
        index += 1;
        if (rci.action != 3) {
            revert InvalidRegisterChainAction(rci.action);
        }

        rci.evmChainId = encodedRecoverChainId.toUint256(index);
        index += 32;

        rci.newChainId = encodedRecoverChainId.toUint16(index);
        index += 2;

        if (encodedRecoverChainId.length != index) {
            revert InvalidRecoverChainLength(encodedRecoverChainId.length);
        }
    }

    //helper functions
    function upgradeImplementation(address newImplementation) internal {
        address currentImplementation = _getImplementation();

        _upgradeTo(newImplementation);

        // Call initialize function of the new implementation
        (bool success, bytes memory reason) =
            newImplementation.delegatecall(abi.encodeWithSignature("initialize()"));

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
        (IWormhole.VM memory vm, bool valid, string memory reason) =
            getWormholeState().parseAndVerifyVM(encodedVM);

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

    function truncateReturnData(bytes memory returnData)
        internal
        pure
        returns (bytes memory returnDataTruncated)
    {
        if (returnData.length <= 124) {
            returnDataTruncated = returnData;
        } else {
            returnDataTruncated = returnData.slice(0, 124);
        }
    }

    //setters
    function setConsumedGovernanceAction(bytes32 hash) internal {
        _state.consumedGovernanceActions[hash] = true;
    }

    function setRelayProvider(address defaultRelayProvider) internal {
        _state.defaultRelayProvider = defaultRelayProvider;
    }

    function setRegisteredCoreRelayerContract(
        uint16 targetChain,
        bytes32 relayerAddress
    ) internal {
        _state.registeredCoreRelayerContract[targetChain] = relayerAddress;
    }

    function setChainId(uint16 _chainId) internal {
        _state.provider.chainId = _chainId;
    }

    function setEvmChainId(uint256 _evmChainId) internal {
        if (_evmChainId != block.chainid) {
            revert InvalidEvmChainId();
        }
        _state.evmChainId = _evmChainId;
    }

    //getters
    function getWormholeState() internal view returns (IWormhole) {
        return IWormhole(_state.provider.wormhole);
    }

    function chainId() internal view returns (uint16) {
        return _state.provider.chainId;
    }

    function evmChainId() internal view returns (uint256) {
        return _state.evmChainId;
    }

    function isFork() internal view returns (bool) {
        return evmChainId() != block.chainid;
    }

    function governanceActionIsConsumed(bytes32 hash) internal view returns (bool) {
        return _state.consumedGovernanceActions[hash];
    }

    function governanceChainId() internal view returns (uint16) {
        return _state.provider.governanceChainId;
    }

    function governanceContract() internal view returns (bytes32) {
        return _state.provider.governanceContract;
    }
}
