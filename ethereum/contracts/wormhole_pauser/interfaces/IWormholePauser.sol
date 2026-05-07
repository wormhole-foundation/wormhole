// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

interface IWormholePauser {
    /// @notice Emitted when the delegated pauser configuration is updated via governance.
    /// @param configIndex The new monotonic config index.
    /// @param threshold Approval threshold for the new config.
    /// @param expiryDuration Proposal expiry duration in seconds for the new config.
    /// @param signers The signer set authorized under the new config.
    event ConfigSet(uint16 indexed configIndex, uint8 threshold, uint64 expiryDuration, address[] signers);

    /// @notice Emitted when a new pause proposal is created.
    /// @param proposalId Monotonic proposal identifier.
    /// @param proposer Signer that proposed this call. Their auto-approval is emitted in a separate `ProposalApproved`.
    /// @param target Contract the proposal will call when the threshold is met.
    /// @param payload Calldata the proposal will pass to `target`.
    /// @param configIndex Config index in effect when this proposal was created.
    /// @param expiresAt UNIX timestamp at which this proposal expires.
    event ProposalProposed(
        uint256 indexed proposalId,
        address indexed proposer,
        address indexed target,
        bytes payload,
        uint16 configIndex,
        uint64 expiresAt
    );

    /// @notice Emitted when a signer approves a proposal.
    event ProposalApproved(uint256 indexed proposalId, address indexed signer, uint8 approvalCount);

    /// @notice Emitted when a signer cancels their previous approval of a proposal.
    event ProposalApprovalCancelled(uint256 indexed proposalId, address indexed signer, uint8 approvalCount);

    /// @notice Emitted when a proposal is executed (i.e. the threshold-meeting approval call into `target` succeeded).
    event ProposalExecuted(uint256 indexed proposalId);

    struct Proposal {
        bool exists;
        bool executed;
        uint8 approvalCount;
        uint16 configIndex;
        uint64 expiresAt;
        address target;
        bytes payload;
    }

    /// @notice Apply a `SetConfigEvm` governance VAA, updating the signer set, threshold, expiry duration,
    ///         and config index. The config index in the message must be `currentIndex + 1`.
    function submitConfig(bytes calldata encodedVm) external;

    /// @notice Create a new pause proposal. Caller must be in the current signing set. The caller's
    ///         approval is recorded automatically; a `threshold == 1` configuration therefore executes
    ///         the call in the same transaction.
    function propose(address target, bytes calldata payload) external returns (uint256 proposalId);

    /// @notice Approve a pending proposal. If this approval meets the threshold, the proposal's call
    ///         is executed in the same transaction. If the call reverts, the entire transaction
    ///         reverts (including the approval increment), so any signer may retry.
    function approve(uint256 proposalId) external;

    /// @notice Cancel the caller's approval of a pending proposal.
    function cancelApproval(uint256 proposalId) external;

    /// @notice Whether `who` is in the current signing set.
    function isSigner(address who) external view returns (bool);

    /// @notice Get full state for `proposalId`. Returns a zero-valued Proposal if it does not exist.
    function getProposal(uint256 proposalId) external view returns (Proposal memory);

    /// @notice Whether `signer` has approved `proposalId` (and not subsequently cancelled).
    function hasApproved(uint256 proposalId, address signer) external view returns (bool);
}
