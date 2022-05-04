// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "@openzeppelin/contracts/utils/StorageSlot.sol";

/**
 * @dev Abstract contract module that provides a bridge contract with the ability to temporarily suspend transfers.
 *
 * Inheriting from `ShutdownSwitch` will allow a bridge contract to use the isEnabled modifier to prevent transactions
 * when the shutdown switch is engaged via a guardian voting mechanism.
 *
 * This contract provides a public function that allows the guardians to vote to disable / enable transfers.
 * When a predetermined number of guardians vote to disable, then transfers are blocked until the number of
 * disable votes is reduced below that threshold.
 *
 * NOTE: The number of votes to enable is irrelevant. As long as there are enough votes to disable, transfers are blocked.
 *
 * An important design goal of this implementation is to minimize the impact (gas) to users, so the check to determine
 * if transfers are enabled is a simple boolean check.
 *
 * The shutdown status is only updated when a vote is cast. When that happens, this contract gets the current guardian set.
 * It validates that the vote is for an active guardian. If so, it updates their vote, counts the number of
 * votes to disable, and updates the shutdown status.
 *
 * Votes are cast using an authorization proof that is unique to each guardian and the wallet public key used to generate
 * the proof. That wallet should be different from the guardian's private key. When a guardian casts a vote, it must be sent
 * from that wallet.
 *
 * The authorization proof is generated using the wallet public key and the guardian private key. When a vote is cast,
 * the code uses the authorization proof and the message sender (wallet public key) to recover the guardian public key.
 * That is compared to the current list of active guardians, and if a match is found, then the vote is valid.
 *
 * NOTE: The status is updated when any active guardian casts a vote, whether it changes anything or not. This means that,
 * after a guardian set update, the shutdown status can be updated by casting an enable vote when you are already enabled.
 *
 * This contract provides public getters to query the shutdown status. It also emits events when valid votes are cast,
 * or the shutdown status changes.
 */

import "./libraries/external/BytesLib.sol";
import "./interfaces/IWormhole.sol";

abstract contract ShutdownSwitch {
    using BytesLib for bytes;

    /// @dev Returns the threshold of the number of votes required to disable transfers.
    function requiredVotesToShutdown() public view returns (uint16) {
        return computeRequiredVotesToShutdown(getCurrentGuardianSet().keys.length);
    }

    /// @dev Returns the current number of votes to disable transfers. When this value reaches requiredVotesToShutdown(), then transfers will be disabled.
    function numVotesToShutdown() public view returns (uint16) {
        return computeNumVotesShutdown(getCurrentGuardianSet());
    }

    /// @dev returns the current shutdown status, where true means transfers are enabled.
    function enabledFlag() public view returns (bool) {
        return _getState().enabled;
    }

    /// @dev Returns the current number of votes to disable transfers.
    function currentVotesToShutdown() public view returns (address[] memory) {
        Structs.GuardianSet memory gs = getCurrentGuardianSet();
        address[] memory ret = new address[](computeNumVotesShutdown(gs));
        uint retIdx = 0;
        for (uint idx = 0; (idx < gs.keys.length); idx++) {
            if (_getShutdownVote(gs.keys[idx])) {
                ret[retIdx++] = gs.keys[idx];
            }
        }

        return ret;
    }

    /// @dev Event published whenenver a valid guardian votes.
    event ShutdownVoteCast(address indexed voter, bool votedToEnable, uint16 numVotesToShutdown, bool enableFlag);

    /// @dev Event published whenever the shutdown status changes from enabled to disabled, or vice versa.
    event ShutdownStatusChanged(bool enabledFlag, uint16 numVotesToShutdown);

    /// @dev Function that must be implemented by contracts inheriting from this one.
    function getCurrentGuardianSet() public virtual view returns (Structs.GuardianSet memory);

    /// @dev The number of disable votes required to disable transfers (assuming there are at least that many guardians).
    uint16 constant REQUIRED_NO_VOTES = 3;
   
    constructor() {
        setUpShutdownSwitch();
    }

    /// @dev Function that should be called from setup() on new contract deployments, or initialize() on the initial upgrade to deploy this feature.
    function setUpShutdownSwitch() internal {
        _setEnabled(true);
    }

    /// @dev modifier used to block transfers when shutdown.
    modifier isEnabled {
        require(enabledFlag(), "transfers are temporarily disabled");
        _;
    }

    function castShutdownVote(bytes memory authProof) public {
        _castVote(authProof, false);
    }
    
    function castStartupVote(bytes memory authProof) public {
        _castVote(authProof, true);
    }

    /// @dev This is the function that allows guardians to vote, and determines the resulting shutdown status.
    function _castVote(bytes memory authProof, bool _enabled) private {
        // Extract the guardian public key from the authProof.
        address voter = decodeVoter(msg.sender, authProof);

        // We always use the current guardian set.
        Structs.GuardianSet memory gs = getCurrentGuardianSet();

        // Only currently active guardians are allowed to vote.
        require(isRegisteredVoter(voter, gs), "you are not a registered voter");

        // Update the status of this voter.
        _updateShutdownVote(voter, _enabled);

        // Update the number of votes to disable.
        uint16 votesToShutdown = computeNumVotesShutdown(gs);

        // Determine the new shutdown status and generate the appropriate events.
        bool newEnabledFlag = (votesToShutdown < computeRequiredVotesToShutdown(gs.keys.length));
        emit ShutdownSwitch.ShutdownVoteCast(voter, _enabled, votesToShutdown, newEnabledFlag);

        if (newEnabledFlag != enabledFlag()) {
            _setEnabled(newEnabledFlag);
            emit ShutdownSwitch.ShutdownStatusChanged(newEnabledFlag, votesToShutdown);
        }
    }
    
    /// @dev Extract the guardian key from the authProof.
    function decodeVoter(address sender, bytes memory authProof) public pure returns (address) {
        // The authProof is made up as follows:
        //    r: bytes32
        //    s: bytes32
        //    v: uint8

        require((authProof.length == 65), "invalid auth proof");

        bytes32 r = authProof.toBytes32(0);
        bytes32 s = authProof.toBytes32(32);
        uint8 v = authProof.toUint8(64) + 27; // Adding 27 is required, see here for details: https://github.com/ethereum/go-ethereum/issues/19751#issuecomment-504900739

        bytes32 digest = keccak256(abi.encodePacked(sender));
        return ecrecover(digest, v, r, s);
    }

    /// @dev Determines if a voter is a current guardian.
    function isRegisteredVoter(address voter, Structs.GuardianSet memory gs) private pure returns(bool) {
        for (uint idx = 0; (idx < gs.keys.length); idx++) {
            if (voter == gs.keys[idx]) {
                return true;
            }
        }

        return false;
    }

    /// @dev Returns the number of disable votes required to go disabled.
    function computeRequiredVotesToShutdown(uint numGuardians) private pure returns(uint16) {
        // If the number of active guardians is less than the pre-configured threshold, use that.
        return uint16(numGuardians >= REQUIRED_NO_VOTES ? REQUIRED_NO_VOTES : numGuardians);
    }

    /// @dev Counts up the number of current guardians that have active votes to disable.
    function computeNumVotesShutdown(Structs.GuardianSet memory gs) private view returns (uint16) {
        uint16 votesToShutdown = 0;
        for (uint idx = 0; (idx < gs.keys.length); idx++) {
            if (_getShutdownVote(gs.keys[idx])) {
                votesToShutdown++;
            }
        }

        return votesToShutdown;
    }
    
    /// @dev Since other contracts will inherit from this one, it cannot have any of its own local state.
    /// This is because that would affect the layout of local state in the inheriting contracts. To prevent this,
    /// we use the storage slot library to put our state in deterministic storage.

    bytes32 private constant _STATE_SLOT = keccak256('ShutdownSwitch.state');
    struct ShutdownSwitchState {
        /// @dev A map of all guardians that have active votes to disable. If a guardian is removed from the guardian set while they
        /// have an active vote to disable, they would get left in the map. The current assumption is that this will have minimal impact,
        /// because they would never be referenced again anyway.
        mapping(address => bool) shutdownVotes;

        /// @dev the current shutdown state, where true means transfers are enabled, and false means they are blocked.
        bool enabled;
    }

    /// @dev Accesses our deterministic storage.
    function _getState() private pure returns (ShutdownSwitchState storage r) {
        bytes32 slot = _STATE_SLOT;
        assembly {
            r.slot := slot
        }
    }
    
    /// @dev Updates the enabled flag in deterministic storage.
    function _setEnabled(bool val) private {
        _getState().enabled = val;
    }

    /// @dev Gets the shutdown state for a given address from our deterministic storage. 
    function _getShutdownVote(address voter) private view returns (bool) {
        return _getState().shutdownVotes[voter];
    }

    /// @dev Updates the voter map in deterministic storage.
    function _updateShutdownVote(address voter, bool enabled) private {
        // Only votes to disable are maintained in the map.
        if (enabled) {
            delete _getState().shutdownVotes[voter];
        } else {
            _getState().shutdownVotes[voter] = true;
        }
    }
}
