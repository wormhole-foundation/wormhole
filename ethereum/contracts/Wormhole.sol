// contracts/Wormhole.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.6.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/math/SafeMath.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "./BytesLib.sol";

contract Wormhole is ReentrancyGuard {
    using BytesLib for bytes;

    // Chain ID of Ethereum
    uint8 public CHAIN_ID = 2;

    struct GuardianSet {
        address[] keys;
        uint32 expiration_time;
    }

    event LogGuardianSetChanged(
        uint32 oldGuardianIndex,
        uint32 newGuardianIndex
    );

    event LogMessagePublished(
        address emitter_address,
        uint32 nonce,
        bytes payload
    );

    struct ParsedVAA {
        uint8 version;
        bytes32 hash;
        uint32 guardian_set_index;
        uint32 timestamp;
        uint8 action;
        bytes payload;
    }

    // Mapping of guardian_set_index => guardian set
    mapping(uint32 => GuardianSet) public guardian_sets;
    // Current active guardian set
    uint32 public guardian_set_index;

    // Period for which a guardian set stays active after it has been replaced
    uint32 public guardian_set_expirity;

    // Mapping of already consumedVAAs
    mapping(bytes32 => bool) public consumedVAAs;

    constructor(GuardianSet memory initial_guardian_set, uint32 _guardian_set_expirity) public {
        guardian_sets[0] = initial_guardian_set;
        // Explicitly set for doc purposes
        guardian_set_index = 0;
        guardian_set_expirity = _guardian_set_expirity;
    }

    function getGuardianSet(uint32 idx) view public returns (GuardianSet memory gs) {
        return guardian_sets[idx];
    }

    // Publish a message to be attested by the Wormhole network
    function publishMessage(
        uint32 nonce,
        bytes memory payload
    ) public {
        emit LogMessagePublished(msg.sender, nonce, payload);
    }

    // Enact a governance VAA
    function executeGovernanceVAA(
        bytes calldata vaa
    ) public nonReentrant {
        ParsedVAA memory parsed_vaa = parseAndVerifyVAA(vaa);
        // Process VAA
        if (parsed_vaa.action == 0x01) {
            require(parsed_vaa.guardian_set_index == guardian_set_index, "only the current guardian set can change the guardian set");
            vaaUpdateGuardianSet(parsed_vaa.payload);
        } else {
            revert("invalid VAA action");
        }

        // Set the VAA as consumed
        consumedVAAs[parsed_vaa.hash] = true;
    }

    // parseAndVerifyVAA parses raw VAA data into a struct and verifies whether it contains sufficient signatures of an
    // active guardian set i.e. is valid according to Wormhole consensus rules.
    function parseAndVerifyVAA(bytes calldata vaa) public view returns (ParsedVAA memory parsed_vaa) {
        parsed_vaa.version = vaa.toUint8(0);
        require(parsed_vaa.version == 1, "VAA version incompatible");

        // Load 4 bytes starting from index 1
        parsed_vaa.guardian_set_index = vaa.toUint32(1);

        uint256 len_signers = vaa.toUint8(5);
        uint offset = 6 + 66 * len_signers;

        // Load 4 bytes timestamp
        parsed_vaa.timestamp = vaa.toUint32(offset);

        // Hash the body
        parsed_vaa.hash = keccak256(vaa.slice(offset, vaa.length - offset));
        require(!consumedVAAs[parsed_vaa.hash], "VAA was already executed");

        GuardianSet memory guardian_set = guardian_sets[parsed_vaa.guardian_set_index];
        require(guardian_set.keys.length > 0, "invalid guardian set");
        require(guardian_set.expiration_time == 0 || guardian_set.expiration_time > block.timestamp, "guardian set has expired");
        // We're using a fixed point number transformation with 1 decimal to deal with rounding.
        require(((guardian_set.keys.length * 10 / 3) * 2) / 10 + 1 <= len_signers, "no quorum");

        int16 last_index = - 1;
        for (uint i = 0; i < len_signers; i++) {
            uint8 index = vaa.toUint8(6 + i * 66);
            require(index > last_index, "signature indices must be ascending");
            last_index = int16(index);

            bytes32 r = vaa.toBytes32(7 + i * 66);
            bytes32 s = vaa.toBytes32(39 + i * 66);
            uint8 v = vaa.toUint8(71 + i * 66);
            v += 27;
            require(ecrecover(parsed_vaa.hash, v, r, s) == guardian_set.keys[index], "VAA signature invalid");
        }

        parsed_vaa.payload = vaa.slice(offset + 4, vaa.length - (offset + 4));
    }

    function vaaUpdateGuardianSet(bytes memory data) private {
        uint32 new_guardian_set_index = data.toUint32(0);
        require(new_guardian_set_index == guardian_set_index + 1, "index must increase in steps of 1");
        uint8 len = data.toUint8(4);

        address[] memory new_guardians = new address[](len);
        for (uint i = 0; i < len; i++) {
            address addr = data.toAddress(5 + i * 20);
            new_guardians[i] = addr;
        }

        uint32 old_guardian_set_index = guardian_set_index;
        guardian_set_index = new_guardian_set_index;

        GuardianSet memory new_guardian_set = GuardianSet(new_guardians, 0);
        guardian_sets[guardian_set_index] = new_guardian_set;
        guardian_sets[old_guardian_set_index].expiration_time = uint32(block.timestamp) + guardian_set_expirity;

        emit LogGuardianSetChanged(old_guardian_set_index, guardian_set_index);
    }

    fallback() external payable {revert("unsupported");}

    receive() external payable {revert("the Wormhole core does not accept assets");}
}
