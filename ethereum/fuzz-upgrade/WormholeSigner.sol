// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import {IWormhole} from "../contracts/interfaces/IWormhole.sol";
import "../contracts/libraries/external/BytesLib.sol";

interface IHevm {
    function warp(uint256 newTimestamp) external;
    function roll(uint256 newNumber) external;
    function load(address where, bytes32 slot) external returns (bytes32);
    function store(address where, bytes32 slot, bytes32 value) external;
    function sign(uint256 privateKey, bytes32 digest) external returns (uint8 r, bytes32 v, bytes32 s);
    function addr(uint256 privateKey) external returns (address add);
    function ffi(string[] calldata inputs) external returns (bytes memory result);
    function prank(address newSender) external;
    function createFork(string calldata urlOrAlias) external returns (uint256 forkId);
    function selectFork(uint256 forkId) external;
    function deal(address usr, uint amt) external;
}

/**
 * @notice These are the common parts for the signing and the non signing wormhole simulators.
 * @dev This contract is meant to be used when testing against a mainnet fork.
 */
abstract contract WormholeSigner {
    using BytesLib for bytes;

    bytes32 constant MODULE = 0x00000000000000000000000000000000000000000000000000000000436f7265;
    uint16  constant CHAINID = 2;
    uint16  constant GOVERNANCE_CHAIN_ID = 1;
    bytes32 constant governanceContract = 0x0000000000000000000000000000000000000000000000000000000000000004;
    uint256 constant testGuardianKey = 93941733246223705020089879371323733820373732307041878556247502674739205313440;
   
   IHevm hevm = IHevm(0x7109709ECfa91a80626fF3989D68f67F5b1DD12D);

   uint256[] guardianSetKeys = [testGuardianKey];
   uint256[] pendingGuardianSetKeys;
   address[] pendingGuardianSetAddresses;

    function doubleKeccak256(bytes memory body) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(keccak256(body)));
    }

    /**
     * @notice Encodes Wormhole message body into bytes
     * @param vm_ Wormhole VM struct
     * @return encodedObservation Wormhole message body encoded into bytes
     */
    function encodeObservation(IWormhole.VM memory vm_)
        internal
        pure
        returns (bytes memory encodedObservation)
    {
        encodedObservation = abi.encodePacked(
            vm_.timestamp,
            vm_.nonce,
            vm_.emitterChainId,
            vm_.emitterAddress,
            vm_.sequence,
            vm_.consistencyLevel,
            vm_.payload
        );
    }

    function overrideToTestGuardian(IWormhole wormhole, address testGuardianAddress) internal {
        {
            // Get slot for Guardian Set at the current index
            uint32 guardianSetIndex = wormhole.getCurrentGuardianSetIndex();
            bytes32 guardianSetSlot = keccak256(abi.encode(guardianSetIndex, 2));

            // Overwrite all but first guardian set to zero address. This isn't
            // necessary, but just in case we inadvertently access these slots
            // for any reason.
            uint256 numGuardians = uint256(hevm.load(address(wormhole), guardianSetSlot));
            for (uint256 i = 1; i < numGuardians;) {
                hevm.store(
                    address(wormhole),
                    bytes32(uint256(keccak256(abi.encodePacked(guardianSetSlot))) + i),
                    bytes32(0)
                );
                unchecked {
                    i += 1;
                }
            }

            // Now overwrite the first guardian key with the devnet key specified
            // in the function argument.
            hevm.store(
                address(wormhole),
                bytes32(uint256(keccak256(abi.encodePacked(guardianSetSlot))) + 0), // just explicit w/ index 0
                bytes32(uint256(uint160(testGuardianAddress)))
            );

            // Change the length to 1 guardian
            hevm.store(
                address(wormhole),
                guardianSetSlot,
                bytes32(uint256(1)) // length == 1
            );

            // Confirm guardian set override
            address[] memory guardians = wormhole.getGuardianSet(guardianSetIndex).keys;
            assert(guardians.length == 1);
            assert(guardians[0] == testGuardianAddress);
        }
    }

    function encodeAndSignGovernanceMessage(bytes memory message, IWormhole wormhole)
        internal
        returns (bytes memory signedMessage, bytes32 messageHash)
    {     
        IWormhole.VM memory vm_ = IWormhole.VM({
            version: 1,
            timestamp: uint32(block.timestamp),
            nonce: 0,
            emitterChainId: GOVERNANCE_CHAIN_ID,
            emitterAddress: governanceContract,
            sequence: 0,
            consistencyLevel: 1,
            payload: message,
            guardianSetIndex: 0,
            signatures: new IWormhole.Signature[](0),
            hash: bytes32("")
        });

        uint32 currentGuardianSetIndex = wormhole.getCurrentGuardianSetIndex();
        uint256 guardianSetLength = guardianSetKeys.length;

        // Compute the hash of the body
        bytes memory body = encodeObservation(vm_);
        vm_.hash = doubleKeccak256(body);
        messageHash = vm_.hash;

        signedMessage = abi.encodePacked(
            vm_.version,
            currentGuardianSetIndex,
            uint8(guardianSetLength)
        );

        // Sign the hash with the guardian private keys
        IWormhole.Signature[] memory sigs = new IWormhole.Signature[](guardianSetLength);
        for (uint256 i = 0; i < guardianSetLength; i++) {
            (sigs[i].v, sigs[i].r, sigs[i].s) = hevm.sign(guardianSetKeys[i], vm_.hash);
            sigs[i].guardianIndex = uint8(i);

            signedMessage = abi.encodePacked(
                signedMessage,
                sigs[i].guardianIndex,
                sigs[i].r,
                sigs[i].s,
                sigs[i].v - 27
            );
        }

        signedMessage = abi.encodePacked(
            signedMessage,
            body
        );
    }

    function encodeAndSignMessage(bytes memory message, uint16 emitterChainId, bytes32 emitterAddress, IWormhole wormhole) internal returns (bytes memory signedMessage) {
        IWormhole.VM memory vm_ = IWormhole.VM({
            version: 1,
            timestamp: uint32(block.timestamp),
            nonce: 0,
            emitterChainId: emitterChainId,
            emitterAddress: emitterAddress,
            sequence: 0,
            consistencyLevel: 1,
            payload: message,
            guardianSetIndex: 0,
            signatures: new IWormhole.Signature[](0),
            hash: bytes32("")
        });

        uint32 currentGuardianSetIndex = wormhole.getCurrentGuardianSetIndex();
        uint256 guardianSetLength = guardianSetKeys.length;

        // Compute the hash of the body
        bytes memory body = encodeObservation(vm_);
        vm_.hash = doubleKeccak256(body);

        signedMessage = abi.encodePacked(
            vm_.version,
            currentGuardianSetIndex,
            uint8(guardianSetLength)
        );

        // Sign the hash with the guardian private keys
        IWormhole.Signature[] memory sigs = new IWormhole.Signature[](guardianSetLength);
        for (uint256 i = 0; i < guardianSetLength; i++) {
            (sigs[i].v, sigs[i].r, sigs[i].s) = hevm.sign(guardianSetKeys[i], vm_.hash);
            sigs[i].guardianIndex = uint8(i);

            signedMessage = abi.encodePacked(
                signedMessage,
                sigs[i].guardianIndex,
                sigs[i].r,
                sigs[i].s,
                sigs[i].v - 27
            );
        }

        signedMessage = abi.encodePacked(
            signedMessage,
            body
        );
    }

    function rollNewGuardianSet(IWormhole wormhole, bytes32 seed) internal returns (uint32 newIndex){
        // We can roll more than 256 guardian sets, but the signature encoding would fail at that point
        // since we force it to a uint8
        newIndex = wormhole.getCurrentGuardianSetIndex() + 1;
        if (newIndex > 256) {
            return 0;
        }
        
        for (uint256 i = 0; i < 19; i++) {
            uint256 privateKey = uint256(keccak256(abi.encodePacked(seed, i)));
            pendingGuardianSetKeys.push(privateKey);
            pendingGuardianSetAddresses.push(hevm.addr(privateKey));
        }
    }

    function commitNewGuardianSet() internal {
        guardianSetKeys = pendingGuardianSetKeys;
        pendingGuardianSetKeys = new uint256[](0);
        pendingGuardianSetAddresses = new address[](0);
    }
}
