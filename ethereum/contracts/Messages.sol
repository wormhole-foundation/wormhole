// contracts/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "./Getters.sol";
import "./Setters.sol";
import "./Structs.sol";
import "./libraries/external/BytesLib.sol";


contract Messages is Getters, Setters {
    using BytesLib for bytes;

    /// @dev parseAndVerifyVM serves to parse an encodedVM and wholy validate it for consumption
    function parseAndVerifyVM(bytes calldata encodedVM) public view returns (Structs.VM memory vm, bool valid, string memory reason) {
        vm = parseVM(encodedVM);
        (valid, reason) = verifyVM(vm);
    }

    /**
     * @dev parseAndVerifyBatchVM serves to parse an encodedVM2 that includes a batch of observations
     * and wholy validate the batch for consumption
     * it saves the hash of each observation in a cache when `cache` is set to true
     */
    function parseAndVerifyBatchVM(bytes calldata encodedVM2, bool cache) public returns (Structs.VM2 memory vm, bool valid, string memory reason) {
        vm = parseBatchVM(encodedVM2);
        (valid, reason) = verifyBatchVM(vm, cache);
    }

    /**
     * @dev clearBatchCache serves to reduce gas costs by clearing VM hashes from storage
     * it must be called in the same transaction as parseAndVerifyBatchVM to reduce gas usage
     * it only removes hashes from storage that are provided in the hashesToClear argument
     */
    function clearBatchCache(bytes32[] memory hashesToClear) public {
        uint256 hashesLen = hashesToClear.length;
        for (uint i = 0; i < hashesLen;) {
            updateVerifiedCacheStatus(hashesToClear[i], false);
            unchecked { i += 1; }
        }
    }

    /**
     * @dev verifySignatures serves to validate arbitrary signatures against an arbitrary guardianSet
     *  - it intentionally does not solve for expectations within guardianSet (you should use verifyVM if you need these protections)
     *  - it intentioanlly does not solve for quorum (you should use verifyVM if you need these protections)
     *  - it intentionally returns true when signatures is an empty set (you should use verifyVM if you need these protections)
     */
    function verifySignatures(bytes32 hash, Structs.Signature[] memory signatures, Structs.GuardianSet memory guardianSet) public pure returns (bool valid, string memory reason) {
        uint8 lastIndex = 0;
        uint256 guardianCount = guardianSet.keys.length;
        uint256 signaturesLen = signatures.length;
        for (uint i = 0; i < signaturesLen;) {
            Structs.Signature memory sig = signatures[i];

            // Ensure that provided signature indices are ascending only
            require(i == 0 || sig.guardianIndex > lastIndex, "signature indices must be ascending");
            lastIndex = sig.guardianIndex;

            // @dev Ensure that the provided signature index is within the
            // bounds of the guardianSet. This is implicitly checked by the array
            // index operation below, so this check is technically redundant.
            // However, reverting explicitly here ensures that a bug is not
            // introduced accidentally later due to the nontrivial storage
            // semantics of solidity.
            require(sig.guardianIndex < guardianCount, "guardian index out of bounds");

            // Check to see if the signer of the signature does not match a specific Guardian key at the provided index
            if(ecrecover(hash, sig.v, sig.r, sig.s) != guardianSet.keys[sig.guardianIndex]){
                return (false, "VM signature invalid");
            }
            unchecked { i += 1; }
        }

        // If we are here, we've validated that the provided signatures are valid for the provided guardianSet
        return (true, "");
    }

    function verifyHeader(Structs.Header memory vm) internal view returns (bool valid, string memory reason) {
        /// @dev Obtain the current guardianSet for the guardianSetIndex provided
        Structs.GuardianSet memory guardianSet = getGuardianSet(vm.guardianSetIndex);

       /**
        * @dev Checks whether the guardianSet has zero keys
        * WARNING: This keys check is critical to ensure the guardianSet has keys present AND to ensure
        * that guardianSet key size doesn't fall to zero and negatively impact quorum assessment.  If guardianSet
        * key length is 0 and vm.signatures length is 0, this could compromise the integrity of both vm and
        * signature verification.
        */
        if(guardianSet.keys.length == 0){
            return (false, "invalid guardian set");
        }

        /// @dev Checks if VM guardian set index matches the current index (unless the current set is expired).
        if(vm.guardianSetIndex != getCurrentGuardianSetIndex() && guardianSet.expirationTime < block.timestamp){
            return (false, "guardian set has expired");
        }

       /**
        * @dev We're using a fixed point number transformation with 1 decimal to deal with rounding.
        *   WARNING: This quorum check is critical to assessing whether we have enough Guardian signatures to validate a VM
        *   if making any changes to this, obtain additional peer review. If guardianSet key length is 0 and
        *   vm.signatures length is 0, this could compromise the integrity of both vm and signature verification.
        */
        if (vm.signatures.length < quorum(guardianSet.keys.length)){
            return (false, "no quorum");
        }

        /// @dev Verify the proposed vm.signatures against the guardianSet
        (bool signaturesValid, string memory invalidReason) = verifySignatures(vm.hash, vm.signatures, guardianSet);
        if(!signaturesValid){
            return (false, invalidReason);
        }

        /// If we are here, we've validated the VM is a valid multi-sig that matches the guardianSet.
        return (true, "");
    }

    /**
    * @dev verifyVM serves to validate an arbitrary VM against a valid Guardian set
    *  - it aims to make sure the VM is for a known guardianSet
    *  - it aims to ensure the guardianSet is not expired
    *  - it aims to ensure the VM has reached quorum
    *  - it aims to verify the signatures provided against the guardianSet
    *  - it aims to verify VM3s (Headless VMs) by checking the hash against
    *    an array of previously verified message hashes
    */
    function verifyVM(Structs.VM memory vm) public view returns (bool valid, string memory reason) {
        if (vm.version == 1) {
            return verifyVM1(vm);
        } else if (vm.version == 3) {
            return verifyVM3(vm);
        } else {
            return (false, "Invalid version");
        }
    }

    function verifyVM1(Structs.VM memory vm) internal view returns (bool valid, string memory reason) {
        Structs.Header memory header;
        header.guardianSetIndex = vm.guardianSetIndex;
        header.signatures = vm.signatures;
        header.hash = vm.hash;

        return verifyHeader(header);
    }

    function verifyVM3(Structs.VM memory vm) internal view returns (bool valid, string memory reason) {
        // Check to see if the hash has been cached
        if (verifiedHashCached(vm.hash)) {
            return (true, "");
        } else {
            return (false, "Could not find hash in cache");
        }
    }

    /**
     * @dev verifyBatchVM serves to validate an arbitrary batch of VMs against a valid Guardian set
     * - it aims to ensure the VM2 is for a known guardianSet
     * - it aims to ensure the guardianSet is not expired
     * - it aims to ensure the VM2 has reached quorum
     * - it aims to verify the signatures provided against the guardianSet
     * - it aims to verify that the guardians have signed the hash of all hashes included in the batch
     * - it aims to verify each observation's hash
     * - it aims to cache each observation's hash if the `cache` argument is true
     */
    function verifyBatchVM(Structs.VM2 memory vm2, bool cache) public returns (bool valid, string memory reason) {
        Structs.Header memory header;
        header.guardianSetIndex = vm2.guardianSetIndex;
        header.signatures = vm2.signatures;
        header.hash = vm2.hash;

        // Verify the header
        (valid, reason) = verifyHeader(header);

        if (valid) {
            uint256 observationsLen = vm2.observations.length;
            bytes memory batchHashBody;
            for (uint i = 0; i < observationsLen;) {
                // Verify the hash of the observation against parsed array of hashes. This confirms
                // that the observation wasn't tampered with and that order was preserved. The version
                // (uint8(3)) occupies the first byte of each observation, and should not be included in the
                // body when computing the hash.

                // SECURITY: This is a necessary security check and should not be removed.
                require(
                    vm2.hashes[i] == doubleKeccak256(vm2.observations[i].slice(1, vm2.observations[i].length - 1)),
                    "invalid observation"
                );

                // Cache the hash of the observation if `cache` is set to true
                if (cache) {
                    updateVerifiedCacheStatus(vm2.hashes[i], true);
                }

                // Reconstruct the batch hash
                batchHashBody = abi.encodePacked(batchHashBody, vm2.hashes[i]);

                unchecked { i += 1; }
            }

            // Independently verify the batch hash to confirm that none of the
            // observations were modified.
            require(
                vm2.hash == doubleKeccak256(abi.encodePacked(vm2.version, keccak256(batchHashBody))),
                "invalid hash"
            );
        }
    }

    function parseObservation(
        uint256 start,
        uint256 length,
        bytes memory encodedObservation,
        Structs.VM memory vm
    ) internal pure returns (Structs.VM memory) {
        uint256 index = start;

        vm.timestamp = encodedObservation.toUint32(index);
        index += 4;

        vm.nonce = encodedObservation.toUint32(index);
        index += 4;

        vm.emitterChainId = encodedObservation.toUint16(index);
        index += 2;

        vm.emitterAddress = encodedObservation.toBytes32(index);
        index += 32;

        vm.sequence = encodedObservation.toUint64(index);
        index += 8;

        vm.consistencyLevel = encodedObservation.toUint8(index);
        index += 1;

        uint256 consumed = index - start;
        require(length >= consumed, "Insufficient observation length");

        vm.payload = encodedObservation.slice(index, length - consumed);

        return vm;
    }

    function parseSignatures(
        uint256 start,
        uint256 signersLen,
        bytes memory data
    ) internal pure returns (Structs.Signature[] memory signatures) {
        uint256 index = start;
        signatures = new Structs.Signature[](signersLen);
        for (uint i = 0; i < signersLen; i++) {
            signatures[i].guardianIndex = data.toUint8(index);
            index += 1;

            signatures[i].r = data.toBytes32(index);
            index += 32;
            signatures[i].s = data.toBytes32(index);
            index += 32;
            signatures[i].v = data.toUint8(index) + 27;
            index += 1;
        }
    }

    /**
     * @dev parseVM serves to parse an encodedVM into a VM struct
     *  - it determines the version of the VM by checking the first byte
     *  - it intentionally performs no validation functions, it simply parses VMs raw into a struct
     *  - it does not set the signatures or guardianSetIndex for VM3s (Headless VMs)
     */
    function parseVM(bytes memory encodedVM) public pure virtual returns (Structs.VM memory vm) {
        uint8 version = encodedVM.toUint8(0);
        if (version == 1) {
            vm = parseVM1(encodedVM);
        } else if (version == 3) {
            vm = parseVM3(encodedVM);
        } else {
            revert("Invalid version");
        }
    }

    function parseVM1(bytes memory encodedVM) internal pure returns (Structs.VM memory vm) {
        uint256 index = 0;

        vm.version = encodedVM.toUint8(index);
        index += 1;
        // SECURITY: Note that currently the VM.version is not part of the hash
        // and for reasons described below it cannot be made part of the hash.
        // This means that this field's integrity is not protected and cannot be trusted.
        // This is not a problem today since there is only one accepted version, but it
        // could be a problem if we wanted to allow other versions in the future.
        require(vm.version == 1, "VM version incompatible");

        vm.guardianSetIndex = encodedVM.toUint32(index);
        index += 4;

        // Parse Signatures
        uint256 signersLen = encodedVM.toUint8(index);
        index += 1;

        vm.signatures = parseSignatures(index, signersLen, encodedVM);
        index += 66*signersLen;

        /*
        Hash the body

        SECURITY: Do not change the way the hash of a VM is computed!
        Changing it could result into two different hashes for the same observation.
        But xDapps rely on the hash of an observation for replay protection.
        */
        bytes memory body = encodedVM.slice(index, encodedVM.length - index);
        vm.hash = doubleKeccak256(body);

        // Parse the observation
        return parseObservation(index, encodedVM.length - index, encodedVM, vm);
    }

    function parseVM3(bytes memory encodedVM) internal pure returns (Structs.VM memory vm) {
        uint256 index = 0;

        vm.version = encodedVM.toUint8(index);
        index += 1;
        require(vm.version == 3, "VM version incompatible");

        // There are no signatures to parse for version 3 VMs

        /*
        Hash the body

        SECURITY: Do not change the way the hash of a VM is computed!
        Changing it could result into two different hashes for the same observation.
        But xDapps rely on the hash of an observation for replay protection.
        */
        bytes memory body = encodedVM.slice(index, encodedVM.length - index);
        vm.hash = doubleKeccak256(body);

        // Parse the observation
        return parseObservation(index, encodedVM.length - index, encodedVM, vm);
    }

    /**
     * @dev parseVM2 serves to parse an encodedVM into a VM2 struct
     *  - it intentionally performs no validation functions, it simply parses raw into a struct
     */
    function parseBatchVM(bytes memory encodedVM2) public pure virtual returns (Structs.VM2 memory vm2) {
        uint256 index = 0;

        // Parse the header
        vm2.version = encodedVM2.toUint8(index);
        index += 1;
        require(vm2.version == 2, "VM version incompatible");

        vm2.guardianSetIndex = encodedVM2.toUint32(index);
        index += 4;

        // Parse signatures
        uint256 signersLen = encodedVM2.toUint8(index);
        index += 1;

        vm2.signatures = parseSignatures(index, signersLen, encodedVM2);
        index += 66*signersLen;

        // Number of hashes in the batch
        uint256 hashesLen = encodedVM2.toUint8(index);
        index += 1;

        // Compute the batch hash
        bytes memory body = encodedVM2.slice(index, hashesLen * 32);
        vm2.hash = doubleKeccak256(abi.encodePacked(vm2.version, keccak256(body)));

        // Parse hashes
        vm2.hashes = new bytes32[](hashesLen);
        for (uint8 i = 0; i < hashesLen;) {
            vm2.hashes[i] = encodedVM2.toBytes32(index);
            index += 32;
            unchecked { i += 1; }
        }

        // The number of observations in the batch
        uint8 observationsLen = encodedVM2.toUint8(index);
        index += 1;

        // The batch should have the same number of observations and hashes
        require(observationsLen == hashesLen && observationsLen > 0, "invalid number of observations");

        // Save each observation in a byte array
        vm2.observations = new bytes[](observationsLen);
        uint32 observationLen;
        for (uint8 i = 0; i < observationsLen;) {
            // Skip the first byte of the observation. The first byte is the index of the
            // observation in the batch and could eventually be used to support partial batches.
            index += 1;

            observationLen = encodedVM2.toUint32(index);
            index += 4;

            // Prepend uint8(3) to the observation bytes to signal that the
            // bytes are considered a "Headless" VM3 payload.
            vm2.observations[i] = abi.encodePacked(uint8(3), encodedVM2.slice(index, observationLen));

            index += observationLen;
            unchecked { i += 1; }
        }

        // This is necessary to confirm that the observationsLen byte was set correctly
        // for partial batches.
        require(encodedVM2.length == index, "invalid VM2");
    }

    function doubleKeccak256(bytes memory body) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(keccak256(body)));
    }

    /**
     * @dev quorum serves solely to determine the number of signatures required to acheive quorum
     */
    function quorum(uint numGuardians) public pure virtual returns (uint numSignaturesRequiredForQuorum) {
        // The max number of guardians is 255
        require(numGuardians < 256, "too many guardians");
        return ((numGuardians * 2) / 3) + 1;
    }
}
