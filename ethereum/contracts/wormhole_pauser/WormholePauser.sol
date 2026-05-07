// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import "./interfaces/IWormholePauser.sol";
import "../interfaces/IWormhole.sol";
import "../libraries/external/BytesLib.sol";

string constant WORMHOLE_PAUSER_VERSION = "WormholePauser-0.0.1";

/// @title WormholePauser
/// @author Wormhole Project Contributors.
/// @notice Immutable contract that delegates pause authority to a configurable signer set.
///         The signer set, approval threshold, and proposal expiry are configured via Wormhole
///         governance VAAs from the "DelegatedPauser" module. Designated signers can then propose
///         arbitrary `(target, payload)` calls (typically `pause()` on a downstream protocol),
///         and once the approval threshold is met the call is executed in the same transaction.
///
///         See whitepapers/0018_pauser.md.
contract WormholePauser is IWormholePauser {
    using BytesLib for bytes;

    string public constant VERSION = WORMHOLE_PAUSER_VERSION;
    IWormhole public immutable WORMHOLE;

    // "DelegatedPauser" left-padded
    bytes32 internal constant MODULE = 0x000000000000000000000000000000000044656C656761746564506175736572;
    uint8 internal constant SET_CONFIG_EVM_ACTION = 1;

    /// @notice Whether a given governance VAA hash has been consumed.
    mapping(bytes32 => bool) public consumedGovernanceActions;

    /// @notice Current monotonic configuration index. Starts at 0 (no config set); the first valid
    ///         `SetConfigEvm` carries index 1, the next 2, and so on.
    uint16 public configIndex;

    /// @notice Current approval threshold.
    uint8 public threshold;

    /// @notice Current proposal expiry duration in seconds.
    uint64 public expiryDuration;

    /// @notice Counter used to assign unique proposal IDs.
    uint256 public nextProposalId;

    // Signers are stored under a configIndex-versioned mapping so a SetConfig naturally
    // invalidates the previous signer set without iteration / cleanup.
    mapping(uint16 => mapping(address => bool)) internal _isSignerForConfig;

    mapping(uint256 => Proposal) internal _proposals;
    mapping(uint256 => mapping(address => bool)) internal _hasApproved;

    error InvalidVAA(string reason);
    error InvalidGuardianSet();
    error InvalidGovernanceChain();
    error InvalidGovernanceContract();
    error InvalidModule();
    error InvalidAction();
    error InvalidChain();
    error InvalidIndex();
    error InvalidThreshold();
    error InvalidExpiryDuration();
    error EmptySignerSet();
    error ZeroSigner();
    error DuplicateSigner();
    error AlreadyConsumed();
    error InvalidPayloadLength();

    error NotSigner();
    error ProposalDoesNotExist();
    error ProposalAlreadyExecuted();
    error ProposalExpired();
    error ProposalConfigRotated();
    error AlreadyApproved();
    error NotApproved();
    error ExecutionFailed(bytes returnData);

    constructor(address _wormhole) {
        WORMHOLE = IWormhole(_wormhole);
    }

    // ============================ Governance ============================

    /// @inheritdoc IWormholePauser
    function submitConfig(bytes calldata encodedVm) external override {
        IWormhole.VM memory vm = WORMHOLE.parseVM(encodedVm);
        _verifyGovernanceVm(vm);

        // Header: module (32) + action (1) + chainId (2)
        uint256 idx = 0;

        bytes32 module_ = vm.payload.toBytes32(idx);
        idx += 32;
        if (module_ != MODULE) revert InvalidModule();

        uint8 action = vm.payload.toUint8(idx);
        idx += 1;
        if (action != SET_CONFIG_EVM_ACTION) revert InvalidAction();

        uint16 chain_ = vm.payload.toUint16(idx);
        idx += 2;
        if (chain_ != WORMHOLE.chainId()) revert InvalidChain();

        // Body: index (2) + threshold (1) + expiryDuration (8) + numSigners (1) + signers (20 * numSigners)
        uint16 newIndex = vm.payload.toUint16(idx);
        idx += 2;
        if (newIndex != configIndex + 1) revert InvalidIndex();

        uint8 newThreshold = vm.payload.toUint8(idx);
        idx += 1;
        if (newThreshold == 0) revert InvalidThreshold();

        uint64 newExpiryDuration = vm.payload.toUint64(idx);
        idx += 8;
        if (newExpiryDuration == 0) revert InvalidExpiryDuration();

        uint8 numSigners = vm.payload.toUint8(idx);
        idx += 1;
        if (numSigners == 0) revert EmptySignerSet();
        if (newThreshold > numSigners) revert InvalidThreshold();

        address[] memory signers = new address[](numSigners);
        for (uint8 i = 0; i < numSigners; i++) {
            address s = vm.payload.toAddress(idx);
            idx += 20;
            if (s == address(0)) revert ZeroSigner();
            if (_isSignerForConfig[newIndex][s]) revert DuplicateSigner();
            _isSignerForConfig[newIndex][s] = true;
            signers[i] = s;
        }

        if (idx != vm.payload.length) revert InvalidPayloadLength();

        // Effects
        consumedGovernanceActions[vm.hash] = true;
        configIndex = newIndex;
        threshold = newThreshold;
        expiryDuration = newExpiryDuration;

        emit ConfigSet(newIndex, newThreshold, newExpiryDuration, signers);
    }

    // ============================ Public API ============================

    /// @inheritdoc IWormholePauser
    function isSigner(address who) public view override returns (bool) {
        return _isSignerForConfig[configIndex][who];
    }

    /// @inheritdoc IWormholePauser
    function getProposal(uint256 proposalId) external view override returns (Proposal memory) {
        return _proposals[proposalId];
    }

    /// @inheritdoc IWormholePauser
    function hasApproved(uint256 proposalId, address signer) external view override returns (bool) {
        return _hasApproved[proposalId][signer];
    }

    /// @inheritdoc IWormholePauser
    function propose(address target, bytes calldata payload) external override returns (uint256 proposalId) {
        if (!isSigner(msg.sender)) revert NotSigner();

        proposalId = nextProposalId;
        unchecked {
            nextProposalId = proposalId + 1;
        }

        Proposal storage p = _proposals[proposalId];
        p.exists = true;
        p.configIndex = configIndex;
        p.expiresAt = uint64(block.timestamp) + expiryDuration;
        p.target = target;
        p.payload = payload;

        emit ProposalProposed(proposalId, msg.sender, target, payload, p.configIndex, p.expiresAt);

        // Reuse the same code path as approve; a threshold==1 config therefore executes immediately.
        _approve(proposalId);
    }

    /// @inheritdoc IWormholePauser
    function approve(uint256 proposalId) external override {
        if (!isSigner(msg.sender)) revert NotSigner();
        _approve(proposalId);
    }

    function _approve(uint256 proposalId) internal {
        Proposal storage p = _proposals[proposalId];
        if (!p.exists) revert ProposalDoesNotExist();
        if (p.executed) revert ProposalAlreadyExecuted();
        if (block.timestamp >= p.expiresAt) revert ProposalExpired();
        if (p.configIndex != configIndex) revert ProposalConfigRotated();
        if (_hasApproved[proposalId][msg.sender]) revert AlreadyApproved();

        _hasApproved[proposalId][msg.sender] = true;
        unchecked {
            p.approvalCount += 1;
        }

        emit ProposalApproved(proposalId, msg.sender, p.approvalCount);

        if (p.approvalCount >= threshold) {
            // Effect first, then interaction (CEI); on revert the entire transaction reverts and
            // all state — including the approval increment and `executed` flag — is rolled back.
            p.executed = true;
            (bool ok, bytes memory returnData) = p.target.call(p.payload);
            if (!ok) revert ExecutionFailed(returnData);
            emit ProposalExecuted(proposalId);
        }
    }

    /// @inheritdoc IWormholePauser
    function cancelApproval(uint256 proposalId) external override {
        if (!isSigner(msg.sender)) revert NotSigner();
        Proposal storage p = _proposals[proposalId];
        if (!p.exists) revert ProposalDoesNotExist();
        if (p.executed) revert ProposalAlreadyExecuted();
        if (block.timestamp >= p.expiresAt) revert ProposalExpired();
        if (p.configIndex != configIndex) revert ProposalConfigRotated();
        if (!_hasApproved[proposalId][msg.sender]) revert NotApproved();

        _hasApproved[proposalId][msg.sender] = false;
        unchecked {
            p.approvalCount -= 1;
        }

        emit ProposalApprovalCancelled(proposalId, msg.sender, p.approvalCount);
    }

    // ============================ Internal ============================

    function _verifyGovernanceVm(IWormhole.VM memory vm) internal view {
        (bool isValid, string memory reason) = WORMHOLE.verifyVM(vm);
        if (!isValid) revert InvalidVAA(reason);

        if (vm.guardianSetIndex != WORMHOLE.getCurrentGuardianSetIndex()) revert InvalidGuardianSet();

        if (uint16(vm.emitterChainId) != WORMHOLE.governanceChainId()) revert InvalidGovernanceChain();

        if (vm.emitterAddress != WORMHOLE.governanceContract()) revert InvalidGovernanceContract();

        if (consumedGovernanceActions[vm.hash]) revert AlreadyConsumed();
    }
}
