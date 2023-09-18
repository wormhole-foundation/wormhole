// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "forge-std/Test.sol";
import "forge-test/rv-helpers/KEVMCheats.sol";

uint32 constant MAX_UINT8 = 0xff;
uint32 constant MAX_UINT16 = 0xffff;
uint32 constant MAX_UINT32 = 0xffffffff;

bytes32 constant CHAINID_STORAGE_INDEX = bytes32(uint256(0));
bytes32 constant GOVERNANCECONTRACT_STORAGE_INDEX = bytes32(uint256(1));
bytes32 constant GUARDIANSETS_STORAGE_INDEX = bytes32(uint256(2));
bytes32 constant GUARDIANSETINDEX_STORAGE_INDEX = bytes32(uint256(3));
bytes32 constant SEQUENCES_STORAGE_INDEX = bytes32(uint256(4));
bytes32 constant CONSUMEDGOVACTIONS_STORAGE_INDEX = bytes32(uint256(5));
bytes32 constant INITIALIZEDIMPLEMENTATIONS_STORAGE_INDEX = bytes32(uint256(6));
bytes32 constant MESSAGEFEE_STORAGE_INDEX = bytes32(uint256(7));
bytes32 constant EVMCHAINID_STORAGE_INDEX = bytes32(uint256(8));

uint256 constant SECP256K1_CURVE_ORDER =
    115792089237316195423570985008687907852837564279074904382605163141518161494337;

contract TestUtils is Test, KEVMCheats {

    // Returns the index hash of the storage slot of a map at location `index` and the key `_key`.
    function hashedLocation(address _key, bytes32 _index) public pure returns(bytes32) {
        // returns `keccak(#buf(32,_key) +Bytes #buf(32, index))
        return keccak256(abi.encode(_key, _index));
    }

    // Returns the index hash of the storage slot of a map at location `index` and the key `_key`.
    function hashedLocation(bytes32 _key, bytes32 _index) public pure returns(bytes32) {
        // returns `keccak(#buf(32,_key) +Bytes #buf(32, index))
        return keccak256(abi.encode(_key, _index));
    }

    // Returns the index hash of the storage slot of a map at location `index` and the key `_key`.
    function hashedLocationOffset(uint32 _key, bytes32 _index, uint256 offset) public pure returns(bytes32) {
        // returns `keccak(#buf(32,_key) +Bytes #buf(32, index))
        return bytes32(uint256(keccak256(abi.encode(_key, _index))) + offset);
    }

    // Updates an address's storage slot with the given content, using a bitmask.
    // The bitmask should set to 0 those bits that will be updated with the new content.
    // It is assumed that the new content fits in the 0 region of the bitmask.
    function storeWithMask(address contractAddress, bytes32 storageSlot, bytes32 content, bytes32 mask) public returns (bytes32) {
        bytes32 originalStorage = vm.load(contractAddress, storageSlot);
        bytes32 updatedStorage = (mask & originalStorage) | content;
        vm.store(contractAddress, storageSlot, updatedStorage);
        return updatedStorage;
    }

    // Uses KEVM cheatcodes to make the gas and contract storage symbolic.
    modifier symbolic(address contractAddress){
        kevm.infiniteGas();
        kevm.symbolicStorage(contractAddress);
        _;
    }

    // Asserts that the given storage slot doesn't change in the given contract.
    modifier unchangedStorage(address contractAddress, bytes32 storageSlot) {
        bytes32 initialStorage = vm.load(contractAddress, storageSlot);
        _;
        bytes32 finalStorage = vm.load(contractAddress, storageSlot);
        assertEq(initialStorage, finalStorage);
    }

    function validVmHeader(uint32 guardianSetIndex) internal pure returns (bytes memory vmH) {
        uint8 version = 1;
        uint8 signersLen = 1;
        uint8 guardianIndex = 0;

        vmH = abi.encodePacked(
                version,
                guardianSetIndex,
                signersLen,
                guardianIndex
            );
    }

    function payloadSubmitContract(bytes32 module, uint16 chainId, address newImpl) internal pure returns (bytes memory payload) {
        uint8 action = 1;
        bytes32 newContract = bytes32(uint256(uint160(newImpl)));

        payload = abi.encodePacked(
            module,
            action,
            chainId,
            newContract
        );
    }

    function payloadSubmitMessageFee(bytes32 module, uint16 chainId, uint256 newMessageFee) internal pure returns (bytes memory payload) {
        uint8 action = 3;

        payload = abi.encodePacked(
            module,
            action,
            chainId,
            newMessageFee
        );
    }

    function payloadSubmitNewGuardianSet(bytes32 module, uint16 chainId, uint32 newGuardianSetIndex, address[] memory keys) internal pure returns (bytes memory payload) {
        uint8 action = 2;
        uint8 keysLength = uint8(keys.length);

        payload = abi.encodePacked(
            module,
            action,
            chainId,
            newGuardianSetIndex,
            keysLength
        );

        for(uint8 i = 0; i < keysLength; i++)
            payload = abi.encodePacked(payload, keys[i]);
    }

    function payloadSubmitTransferFees(bytes32 module, uint16 chainId, uint256 amount, bytes32 recipient) internal pure returns (bytes memory payload) {
        uint8 action = 4;

        payload = abi.encodePacked(
            module,
            action,
            chainId,
            amount,
            recipient
        );
    }

    function payloadSubmitRecoverChainId(bytes32 module, uint256 evmChainId, uint16 newChainId) internal pure returns (bytes memory payload) {
        uint8 action = 5;

        payload = abi.encodePacked(
            module,
            action,
            evmChainId,
            newChainId
        );
    }
    

    function validVm(
        uint32 guardianSetIndex,
        uint32 timestamp,
        uint32 nonce,
        uint16 emitterChainId,
        bytes32 emitterAddress,
        uint64 sequence,
        uint8 consistencyLevel,
        bytes memory payload,
        uint256 pk)
        internal pure
            returns (bytes memory _vm, bytes32 hash)
    {
        bytes memory header = validVmHeader(guardianSetIndex);

        bytes memory body = abi.encodePacked(
                timestamp,
                nonce,
                emitterChainId,
                emitterAddress,
                sequence,
                consistencyLevel,
                payload
            );
        
        hash = keccak256(abi.encodePacked(keccak256(body)));

        bytes memory signature = validSignature(pk, hash);

        _vm = bytes.concat(header, signature, body);
    }

    function validSignature(uint256 pk, bytes32 hash) public pure returns (bytes memory signature) {
        uint8 v;
        bytes32 r;
        bytes32 s;
        (v, r, s) = vm.sign(pk, hash);

        signature = abi.encodePacked(r, s,(v - 27));
    }

    // @dev compute the storage slot of an array based on its key and offset
    // @dev `keyHash` is generally from `hashedLocationOffset()`
    function arrayElementLocation(bytes32 keyHash, uint8 arrayOffset) public pure returns (bytes32) {
        return bytes32(uint256(keccak256(abi.encodePacked(keyHash))) + arrayOffset);
    }
}
