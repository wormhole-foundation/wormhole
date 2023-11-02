// SPDX-License-Identifier: MIT
pragma solidity >=0.4.22 <0.9.0;

/// @title Migrations Contract
/// @notice This contract is designed to facilitate and record migrations of contract states during upgrades.
contract Migrations {
     /// @notice The owner of the contract, set upon deployment to the address deploying the contract.
    address public owner = msg.sender;
    uint public last_completed_migration;

    /// @dev Restricts function calls to only the owner of the contract.
    modifier restricted() {
        require(
            msg.sender == owner,
            "This function is restricted to the contract's owner"
        );
        _;
    }

    /// @notice Sets a given migration as complete.
    /// @dev Callable only by the current owner due to the `restricted` modifier.
    /// @param completed The identifier of the migration that has been completed.
    function setCompleted(uint completed) public restricted {
        last_completed_migration = completed;
    }
}