// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.17;

import "../../contracts/interfaces/IWormhole.sol";
import "../../contracts/libraries/external/BytesLib.sol";

contract MockWormhole is IWormhole {
    using BytesLib for bytes;

    uint256 private constant VM_VERSION_SIZE = 1;
    uint256 private constant VM_GUARDIAN_SET_SIZE = 4;
    uint256 private constant VM_SIGNATURE_COUNT_SIZE = 1;
    uint256 private constant VM_TIMESTAMP_SIZE = 4;
    uint256 private constant VM_NONCE_SIZE = 4;
    uint256 private constant VM_EMITTER_CHAIN_ID_SIZE = 2;
    uint256 private constant VM_EMITTER_ADDRESS_SIZE = 32;
    uint256 private constant VM_SEQUENCE_SIZE = 8;
    uint256 private constant VM_CONSISTENCY_LEVEL_SIZE = 1;
    uint256 private constant VM_SIZE_MINIMUM = VM_VERSION_SIZE + VM_GUARDIAN_SET_SIZE
        + VM_SIGNATURE_COUNT_SIZE + VM_TIMESTAMP_SIZE + VM_NONCE_SIZE + VM_EMITTER_CHAIN_ID_SIZE
        + VM_EMITTER_ADDRESS_SIZE + VM_SEQUENCE_SIZE + VM_CONSISTENCY_LEVEL_SIZE;

    uint256 private constant SIGNATURE_GUARDIAN_INDEX_SIZE = 1;
    uint256 private constant SIGNATURE_R_SIZE = 32;
    uint256 private constant SIGNATURE_S_SIZE = 32;
    uint256 private constant SIGNATURE_V_SIZE = 1;
    uint256 private constant SIGNATURE_SIZE_TOTAL =
        SIGNATURE_GUARDIAN_INDEX_SIZE + SIGNATURE_R_SIZE + SIGNATURE_S_SIZE + SIGNATURE_V_SIZE;

    mapping(address => uint64) public sequences;
    // Dictionary of VMs that must be mocked as invalid.
    mapping(bytes32 => bool) public invalidVMs;

    uint256 currentMsgFee;
    uint16 immutable wormholeChainId;
    uint256 immutable boundEvmChainId;

    constructor(uint16 initChainId, uint256 initEvmChainId) {
        wormholeChainId = initChainId;
        boundEvmChainId = initEvmChainId;
    }

    function invalidateVM(bytes calldata encodedVm) external {
        VM memory vm = _parseVM(encodedVm);
        invalidVMs[vm.hash] = true;
    }

    function publishMessage(
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) external payable returns (uint64 sequence) {
        require(msg.value == currentMsgFee, "invalid fee");
        sequence = sequences[msg.sender]++;
        emit LogMessagePublished(msg.sender, sequence, nonce, payload, consistencyLevel);
    }

    function parseVM(bytes calldata encodedVm) external pure returns (VM memory vm) {
        vm = _parseVM(encodedVm);
    }

    function parseAndVerifyVM(bytes calldata encodedVm)
        external
        view
        returns (VM memory vm, bool valid, string memory reason)
    {
        vm = _parseVM(encodedVm);
        //behold the rigorous checking!
        valid = !invalidVMs[vm.hash];
        reason = "";
    }

    function _parseVM(bytes calldata encodedVm) internal pure returns (VM memory vm) {
        require(encodedVm.length >= VM_SIZE_MINIMUM, "vm too small");

        bytes memory body;

        uint256 offset = 0;
        vm.version = encodedVm.toUint8(offset);
        offset += 1;

        vm.guardianSetIndex = encodedVm.toUint32(offset);
        offset += 4;

        (vm.signatures, offset) = parseSignatures(encodedVm, offset);

        body = encodedVm[offset:];
        vm.timestamp = encodedVm.toUint32(offset);
        offset += 4;

        vm.nonce = encodedVm.toUint32(offset);
        offset += 4;

        vm.emitterChainId = encodedVm.toUint16(offset);
        offset += 2;

        vm.emitterAddress = encodedVm.toBytes32(offset);
        offset += 32;

        vm.sequence = encodedVm.toUint64(offset);
        offset += 8;

        vm.consistencyLevel = encodedVm.toUint8(offset);
        offset += 1;

        vm.payload = encodedVm[offset:];
        vm.hash = keccak256(abi.encodePacked(keccak256(body)));
    }

    function parseSignatures(
        bytes calldata encodedVm,
        uint256 offset
    ) internal pure returns (Signature[] memory signatures, uint256 offsetAfterParse) {
        uint256 sigCount = uint256(encodedVm.toUint8(offset));
        offset += 1;

        require(
            encodedVm.length >= (VM_SIZE_MINIMUM + sigCount * SIGNATURE_SIZE_TOTAL), "vm too small"
        );

        signatures = new Signature[](sigCount);
        for (uint256 i = 0; i < sigCount; ++i) {
            uint8 guardianIndex = encodedVm.toUint8(offset);
            offset += 1;

            bytes32 r = encodedVm.toBytes32(offset);
            offset += 32;

            bytes32 s = encodedVm.toBytes32(offset);
            offset += 32;

            uint8 v = encodedVm.toUint8(offset);
            offset += 1;

            signatures[i] = Signature({
                r: r,
                s: s,
                // The hardcoded 27 comes from the base offset for public key recovery ids, public key type and network
                // used in ECDSA signatures for bitcoin and ethereum.
                // See https://bitcoin.stackexchange.com/a/5089
                v: v + 27,
                guardianIndex: guardianIndex
            });
        }

        return (signatures, offset);
    }

    function initialize() external {}

    function quorum(uint256 /*numGuardians*/ )
        external
        pure
        returns (uint256 /*numSignaturesRequiredForQuorum*/ )
    {
        return 1;
    }

    /**
     * General state and chain observers
     */
    function chainId() external view returns (uint16) {
        return wormholeChainId;
    }

    function evmChainId() external view returns (uint256) {
        return boundEvmChainId;
    }

    function getCurrentGuardianSetIndex() external pure returns (uint32) {
        return 0;
    }

    function getGuardianSet(uint32 /*index*/ ) external pure returns (GuardianSet memory) {
        revert("unsupported getGuardianSet in wormhole mock");
    }

    function getGuardianSetExpiry() external pure returns (uint32) {
        return 0;
    }

    function governanceActionIsConsumed(bytes32 /*hash*/ ) external pure returns (bool) {
        return false;
    }

    function isInitialized(address /*impl*/ ) external pure returns (bool) {
        return true;
    }

    function isFork() external pure returns (bool) {
        return false;
    }

    function governanceChainId() external pure returns (uint16) {
        return 1;
    }

    function governanceContract() external pure returns (bytes32) {
        return bytes32(0x0000000000000000000000000000000000000000000000000000000000000004);
    }

    function messageFee() external view returns (uint256) {
        return currentMsgFee;
    }

    function nextSequence(address emitter) external view returns (uint64) {
        return sequences[emitter];
    }

    function verifyVM(VM memory /*vm*/ )
        external
        pure
        returns (bool, /*valid*/ string memory /*reason*/ )
    {
        revert("unsupported verifyVM in wormhole mock");
    }

    function verifySignatures(
        bytes32, /*hash*/
        Signature[] memory, /*signatures*/
        GuardianSet memory /*guardianSet*/
    ) external pure returns (bool, /*valid*/ string memory /*reason*/ ) {
        revert("unsupported verifySignatures in wormhole mock");
    }

    function parseContractUpgrade(bytes memory /*encodedUpgrade*/ )
        external
        pure
        returns (ContractUpgrade memory /*cu*/ )
    {
        revert("unsupported parseContractUpgrade in wormhole mock");
    }

    function parseGuardianSetUpgrade(bytes memory /*encodedUpgrade*/ )
        external
        pure
        returns (GuardianSetUpgrade memory /*gsu*/ )
    {
        revert("unsupported parseGuardianSetUpgrade in wormhole mock");
    }

    function parseSetMessageFee(bytes memory /*encodedSetMessageFee*/ )
        external
        pure
        returns (SetMessageFee memory /*smf*/ )
    {
        revert("unsupported parseSetMessageFee in wormhole mock");
    }

    function parseTransferFees(bytes memory /*encodedTransferFees*/ )
        external
        pure
        returns (TransferFees memory /*tf*/ )
    {
        revert("unsupported parseTransferFees in wormhole mock");
    }

    function parseRecoverChainId(bytes memory /*encodedRecoverChainId*/ )
        external
        pure
        returns (RecoverChainId memory /*rci*/ )
    {
        revert("unsupported parseRecoverChainId in wormhole mock");
    }

    function submitContractUpgrade(bytes memory /*_vm*/ ) external pure {
        revert("unsupported submitContractUpgrade in wormhole mock");
    }

    function submitSetMessageFee(bytes memory /*_vm*/ ) external pure {
        revert("unsupported submitSetMessageFee in wormhole mock");
    }

    function setMessageFee(uint256 newFee) external {
        currentMsgFee = newFee;
    }

    function submitNewGuardianSet(bytes memory /*_vm*/ ) external pure {
        revert("unsupported submitNewGuardianSet in wormhole mock");
    }

    function submitTransferFees(bytes memory /*_vm*/ ) external pure {
        revert("unsupported submitTransferFees in wormhole mock");
    }

    function submitRecoverChainId(bytes memory /*_vm*/ ) external pure {
        revert("unsupported submitRecoverChainId in wormhole mock");
    }
}
