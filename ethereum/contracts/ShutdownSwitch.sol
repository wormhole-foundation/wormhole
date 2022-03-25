// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

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
 * NOTE: The number of votes to enable is irrelevant. As long as there are enough votes to disable, trabsfers are blocked.
 *
 * An important design goal of this implementation is to minimize the impact (gas) to users, so the check to determine
 * if transfers are enabled is a simple boolean check.
 *
 * The shutdown status is only updated when a vote is cast. When that happens, this contract gets the current guardian set.
 * It validates that the voter (message sender) is an active guardian. If so, it updates their vote, counts the number of
 * votes to disable, and updates the shutdown status.
 *
 * NOTE: The status is updated when any active guardian casts a vote, whether it changes anything or not. This means that,
 * after a guardian set update, the shutdown status can be updated by casting an enable vote when you are already enabled.
 *
 * This contract provides public getters to query the shutdown status. It also emits events when valid votes are cast,
 * or the shutdown status changes.
 */

import "./interfaces/IWormhole.sol";

abstract contract ShutdownSwitch {
    /// @dev A map of all guardians that have active votes to disable. If a guardian is removed from the guardian set while they
    /// have an active vote to disable, they would get left in the map. The current assumption is that this will have minimal impact.
    mapping(address => bool) private disabledVotes;

    bool private enabled = true;

    /// @dev Returns the number of votes required to disable transfers.
    function requiredVotesToDisable() public view returns (uint16) {
        return computeRequiredVotesToDisable(getCurrentGuardianSet().keys.length);
    }

    /// @dev Returns the current number of votes to disable transfers.
    function numVotesToDisable() public view returns (uint16) {
        return computeNumVotesDisabled(getCurrentGuardianSet());
    }

    /// @dev returns the current shutdown status, where true means transfers are enabled.
    function enabledFlag() public view returns (bool) {
        return enabled;
    }

    /// @dev Event published whenenver a valid guardian votes.
    event ShutdownVoteCast(address indexed voter, bool votedToEnable, uint16 numVotesToDisable, bool enableFlag);

    /// @dev Event published whenever the shutdown status changes from enabled to disabled, or vice versa.
    event ShutdownStatusChanged(bool enabledFlag, uint16 numVotesToDisable);

    /// @dev Function that must be implemented by contracts inheriting from this one.
    function getCurrentGuardianSet() public virtual view returns (Structs.GuardianSet memory);

    /// @dev The number of disable votes required to disable transfers (assuming there are at least that many guardians).
    uint16 constant REQUIRED_NO_VOTES = 3;

    /// @dev Function that should be called from setup() on new contract deployments, or initialize() on the initial upgrade to deploy this feature.
    function setUpShutdownSwitch() internal {
        enabled = true;
    }

    /// @dev modifier used to block transfers when shutdown.
    modifier isEnabled {
        require(enabledFlag(), "transfers are temporarily disabled");
        _;
    }

    /// @dev This is the function that allows guardians to vote, and determines the resulting shutdown status.
    function castShutdownVote(bool _enabled) public {
        // We always use the current guardian set.
        Structs.GuardianSet memory gs = getCurrentGuardianSet();

        // Only currently active guardians are allowed to vote.
        require(isRegisteredVoter(msg.sender, gs), "you are not a registered voter");

        // Only votes to disable are maintained in the map.
        if (_enabled) {
            delete disabledVotes[msg.sender];
        } else {
            disabledVotes[msg.sender] = true;
        }

        // Update the number of votes to disable.
        uint16 votesToDisable = computeNumVotesDisabled(gs);

        // Determine the new shutdown status and generate the appropriate events.
        bool newEnabledFlag = (votesToDisable < computeRequiredVotesToDisable(gs.keys.length));
        emit ShutdownSwitch.ShutdownVoteCast(msg.sender, _enabled, votesToDisable, newEnabledFlag);

        if (newEnabledFlag != enabled) {
            enabled = newEnabledFlag;
            emit ShutdownSwitch.ShutdownStatusChanged(newEnabledFlag, votesToDisable);
        }
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
    function computeRequiredVotesToDisable(uint numGuardians) private pure returns(uint16) {
        // If the number of active guardians is less than the pre-configured threshold, use that.
        return uint16(numGuardians >= REQUIRED_NO_VOTES ? REQUIRED_NO_VOTES : numGuardians);
    }

    /// @dev Counts up the number of current guardians that have active votes to disable.
    function computeNumVotesDisabled(Structs.GuardianSet memory gs) private view returns (uint16) {
        uint16 numDisabled = 0;
        for (uint idx = 0; (idx < gs.keys.length); idx++) {
            if (disabledVotes[gs.keys[idx]]) {
                numDisabled++;
            }
        }

        return numDisabled;
    }
}