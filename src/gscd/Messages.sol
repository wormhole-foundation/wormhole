// contracts/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "./Getters.sol";
import "wormhole-sdk/libraries/BytesParsing.sol";

contract Messages is Getters {
    using BytesParsing for bytes;

    uint8 private constant ADDRESS_SIZE = 20; // in bytes
    uint8 private constant EXPIRATION_TIME_SIZE = 4; // in bytes

    function parseAndVerifyVMOptimized(
        bytes calldata encodedVM,
        bytes calldata guardianSet,
        uint32 guardianSetIndex
    ) public view returns (IWormhole.VM memory vm, bool valid, string memory reason) {
        // Verify that the specified guardian set is a valid.
        bytes32 guardianSetHash = getGuardianSetHash(guardianSetIndex);
        require(
            guardianSetHash == keccak256(guardianSet) && guardianSetHash != bytes32(0),
            "invalid guardian set"
        );

        vm = parseVM(encodedVM);

        // Verify that the VM is signed with the same guardian set that was specified.
        require(vm.guardianSetIndex == guardianSetIndex, "mismatched guardian set index");

        (valid, reason) = verifyVMInternal(vm, parseGuardianSet(guardianSet), false);
    }

    function parseGuardianSet(bytes calldata guardianSetData) public pure returns (IWormhole.GuardianSet memory guardianSet) {
        uint256 guardianSetDataLength = guardianSetData.length;
        uint256 guardianCount = (guardianSetDataLength - EXPIRATION_TIME_SIZE) / ADDRESS_SIZE;

        guardianSet = IWormhole.GuardianSet({
            keys : new address[](guardianCount),
            expirationTime : 0
        });

        uint256 offset = 0;
        for(uint256 i = 0; i < guardianCount;) {
            (guardianSet.keys[i], offset) = guardianSetData.asAddressUnchecked(offset);
            unchecked {
                ++i;
            }
        }

        (guardianSet.expirationTime, offset) = guardianSetData.asUint32Unchecked(offset);

        require(guardianSetDataLength == offset, "invalid guardian set data length");
    }

    /// @dev parseAndVerifyVM serves to parse an encodedVM and wholy validate it for consumption
    function parseAndVerifyVM(bytes calldata encodedVM) public view returns (IWormhole.VM memory vm, bool valid, string memory reason) {
        vm = parseVM(encodedVM);
        /// setting checkHash to false as we can trust the hash field in this case given that parseVM computes and then sets the hash field above
        (valid, reason) = verifyVMInternal(vm, getGuardianSet(vm.guardianSetIndex), false);
    }

   /**
    * @dev `verifyVM` serves to validate an arbitrary vm against a valid Guardian set
    *  - it aims to make sure the VM is for a known guardianSet
    *  - it aims to ensure the guardianSet is not expired
    *  - it aims to ensure the VM has reached quorum
    *  - it aims to verify the signatures provided against the guardianSet
    *  - it aims to verify the hash field provided against the contents of the vm
    */
    function verifyVM(IWormhole.VM memory vm) public view returns (bool valid, string memory reason) {
        (valid, reason) = verifyVMInternal(vm, getGuardianSet(vm.guardianSetIndex), true);
    }

    /**
    * @dev `verifyVMInternal` serves to validate an arbitrary vm against a valid Guardian set
    * if checkHash is set then the hash field of the vm is verified against the hash of its contents
    * in the case that the vm is securely parsed and the hash field can be trusted, checkHash can be set to false
    * as the check would be redundant
    */
    function verifyVMInternal(IWormhole.VM memory vm, IWormhole.GuardianSet memory guardianSet, bool checkHash) internal view returns (bool valid, string memory reason) {
        /**
         * Verify that the hash field in the vm matches with the hash of the contents of the vm if checkHash is set
         * WARNING: This hash check is critical to ensure that the vm.hash provided matches with the hash of the body.
         * Without this check, it would not be safe to call verifyVM on it's own as vm.hash can be a valid signed hash
         * but the body of the vm could be completely different from what was actually signed by the guardians
         */
        if(checkHash){
            bytes memory body = abi.encodePacked(
                vm.timestamp,
                vm.nonce,
                vm.emitterChainId,
                vm.emitterAddress,
                vm.sequence,
                vm.consistencyLevel,
                vm.payload
            );

            bytes32 vmHash = keccak256(abi.encodePacked(keccak256(body)));

            if(vmHash != vm.hash){
                return (false, "vm.hash doesn't match body");
            }
        }

        uint256 guardianCount = guardianSet.keys.length;

       /**
        * @dev Checks whether the guardianSet has zero keys
        * WARNING: This keys check is critical to ensure the guardianSet has keys present AND to ensure
        * that guardianSet key size doesn't fall to zero and negatively impact quorum assessment.  If guardianSet
        * key length is 0 and vm.signatures length is 0, this could compromise the integrity of both vm and
        * signature verification.
        */
        if(guardianCount == 0){
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
        if (vm.signatures.length < quorum(guardianCount)){
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
     * @dev verifySignatures serves to validate arbitrary sigatures against an arbitrary guardianSet
     *  - it intentionally does not solve for expectations within guardianSet (you should use verifyVM if you need these protections)
     *  - it intentioanlly does not solve for quorum (you should use verifyVM if you need these protections)
     *  - it intentionally returns true when signatures is an empty set (you should use verifyVM if you need these protections)
     */
    function verifySignatures(bytes32 hash, IWormhole.Signature[] memory signatures, IWormhole.GuardianSet memory guardianSet) public pure returns (bool valid, string memory reason) {
        uint8 lastIndex = 0;
        uint256 sigCount = signatures.length;
        uint256 guardianCount = guardianSet.keys.length;
        for (uint i = 0; i < sigCount;) {
            IWormhole.Signature memory sig = signatures[i];
            address signatory = ecrecover(hash, sig.v, sig.r, sig.s);
            // ecrecover returns 0 for invalid signatures. We explicitly require valid signatures to avoid unexpected
            // behaviour due to the default storage slot value also being 0.
            require(signatory != address(0), "ecrecover failed with signature");

            /// Ensure that provided signature indices are ascending only
            require(i == 0 || sig.guardianIndex > lastIndex, "signature indices must be ascending");
            lastIndex = sig.guardianIndex;

            /// @dev Ensure that the provided signature index is within the
            /// bounds of the guardianSet. This is implicitly checked by the array
            /// index operation below, so this check is technically redundant.
            /// However, reverting explicitly here ensures that a bug is not
            /// introduced accidentally later due to the nontrivial storage
            /// semantics of solidity.
            require(sig.guardianIndex < guardianCount, "guardian index out of bounds");

            /// Check to see if the signer of the signature does not match a specific Guardian key at the provided index
            if(signatory != guardianSet.keys[sig.guardianIndex]){
                return (false, "VM signature invalid");
            }

            unchecked { ++i; }
        }

        /// If we are here, we've validated that the provided signatures are valid for the provided guardianSet
        return (true, "");
    }

    /**
     * @dev parseVM serves to parse an encodedVM into a vm struct
     *  - it intentionally performs no validation functions, it simply parses raw into a struct
     */
    function parseVM(bytes memory encodedVM) public view virtual returns (IWormhole.VM memory vm) {
        uint256 offset = 0;

        // SECURITY: Note that currently the VM.version is not part of the hash
        // and for reasons described below it cannot be made part of the hash.
        // This means that this field's integrity is not protected and cannot be trusted.
        // This is not a problem today since there is only one accepted version, but it
        // could be a problem if we wanted to allow other versions in the future.
        (vm.version, offset) = encodedVM.asUint8Unchecked(offset);
        require(vm.version == 1, "VM version incompatible");

        // Guardian set index.
        (vm.guardianSetIndex, offset) = encodedVM.asUint32Unchecked(offset);

        // Parse sigs.
        uint256 signersLen;
        (signersLen, offset) = encodedVM.asUint8Unchecked(offset);

        vm.signatures = new IWormhole.Signature[](signersLen);
        for (uint i = 0; i < signersLen;) {
            (vm.signatures[i].guardianIndex, offset) = encodedVM.asUint8Unchecked(offset);
            (vm.signatures[i].r, offset) = encodedVM.asBytes32Unchecked(offset);
            (vm.signatures[i].s, offset) = encodedVM.asBytes32Unchecked(offset);
            (vm.signatures[i].v, offset) = encodedVM.asUint8Unchecked(offset);

            unchecked {
                vm.signatures[i].v += 27;
                ++i;
            }
        }

        /*
        Hash the body

        SECURITY: Do not change the way the hash of a VM is computed!
        Changing it could result into two different hashes for the same observation.
        But xDapps rely on the hash of an observation for replay protection.
        */
        bytes memory body;
        (body, ) = encodedVM.sliceUnchecked(offset, encodedVM.length - offset);
        vm.hash = keccak256(abi.encodePacked(keccak256(body)));

        // Parse the body
        (vm.timestamp, offset) = encodedVM.asUint32Unchecked(offset);
        (vm.nonce, offset) = encodedVM.asUint32Unchecked(offset);
        (vm.emitterChainId, offset) = encodedVM.asUint16Unchecked(offset);
        (vm.emitterAddress, offset) = encodedVM.asBytes32Unchecked(offset);
        (vm.sequence, offset) = encodedVM.asUint64Unchecked(offset);
        (vm.consistencyLevel, offset) = encodedVM.asUint8Unchecked(offset);
        (vm.payload, offset) = encodedVM.sliceUnchecked(offset, encodedVM.length - offset);

        require(encodedVM.length == offset, "invalid payload length");
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
