// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "./Governance.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

/// @title Implementation
/// @notice The top-level implementation contract for the Wormhole core bridge.
/// @dev This contract is deployed behind a proxy (Wormhole.sol) and upgraded via governance.
///      It extends Governance (which extends Messages, Getters, etc.) and adds the message
///      publishing entrypoint and the one-time initialization logic.
contract Implementation is Governance {
    event LogMessagePublished(
        address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel
    );

    /// @notice Publishes a message to the Wormhole network, emitting a `LogMessagePublished` event.
    /// @dev Guardian nodes observe this event and produce a signed VAA (Verified Action Approval)
    ///      attesting to its existence. The VAA can then be submitted to any other Wormhole-connected
    ///      chain to deliver the message.
    ///      Callers must pay exactly `messageFee()` wei or the call will revert.
    ///      The sequence number is monotonically increasing per sender — together with the emitter
    ///      chain ID and address, it uniquely identifies this message and is used for replay protection
    ///      on the destination chain.
    /// @param nonce An arbitrary nonce chosen by the caller. Not enforced by the protocol —
    ///        can be used by integrators for message deduplication or batch grouping.
    /// @param payload The application-level message payload. No size enforcement by the core bridge
    ///        (though guardian node limits may apply).
    /// @param consistencyLevel The finality level the guardian network should wait for before signing.
    ///        Interpretation is chain-specific. On Ethereum: 200 = instant (unsafe), 1 = safe, 0 = finalized.
    ///        See https://docs.wormhole.com/wormhole/reference/glossary#consistency-level
    /// @return sequence The sequence number assigned to this message for this emitter.
    function publishMessage(
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) public payable returns (uint64 sequence) {
        // check fee
        require(msg.value == messageFee(), "invalid fee");

        sequence = useSequence(msg.sender);
        // emit log
        emit LogMessagePublished(msg.sender, sequence, nonce, payload, consistencyLevel);
    }

    /// @dev Reads and increments the sequence number for the given emitter atomically.
    function useSequence(
        address emitter
    ) internal returns (uint64 sequence) {
        sequence = nextSequence(emitter);
        setNextSequence(emitter, sequence + 1);
    }

    /// @notice Initializes the implementation contract after it is deployed and upgraded to.
    /// @dev Must be called exactly once per implementation address via `upgradeImplementation`.
    ///      Sets the EVM chain ID (`evmChainId`) from a hardcoded mapping of Wormhole chain IDs
    ///      to EVM chain IDs. This mapping is needed because the Wormhole chain ID (set during
    ///      proxy setup) is not the same as the EVM chain ID, and `block.chainid` is only
    ///      available at runtime, not at proxy initialization time.
    ///      Reverts with "Unknown chain id." if the Wormhole chain ID is not in the mapping.
    function initialize() public virtual initializer {
        // this function needs to be exposed for an upgrade to pass
        if (evmChainId() == 0) {
            uint256 evmChainId;
            uint16 chain = chainId();

            // Wormhole chain ids explicitly enumerated
            if (chain == 2) {
                evmChainId = 1; // ethereum
            } else if (chain == 4) {
                evmChainId = 56; // bsc
            } else if (chain == 5) {
                evmChainId = 137; // polygon
            } else if (chain == 6) {
                evmChainId = 43114; // avalanche
            } else if (chain == 7) {
                evmChainId = 42262; // oasis
            } else if (chain == 9) {
                evmChainId = 1313161554; // aurora
            } else if (chain == 10) {
                evmChainId = 250; // fantom
            } else if (chain == 11) {
                evmChainId = 686; // karura
            } else if (chain == 12) {
                evmChainId = 787; // acala
            } else if (chain == 13) {
                evmChainId = 8217; // klaytn
            } else if (chain == 14) {
                evmChainId = 42220; // celo
            } else if (chain == 16) {
                evmChainId = 1284; // moonbeam
            } else if (chain == 17) {
                evmChainId = 245022934; // neon
            } else if (chain == 23) {
                evmChainId = 42161; // arbitrum
            } else if (chain == 24) {
                evmChainId = 10; // optimism
            } else if (chain == 25) {
                evmChainId = 100; // gnosis
            } else {
                revert("Unknown chain id.");
            }

            setEvmChainId(evmChainId);
        }
    }

    /// @dev Guards `initialize()` to ensure it can only be called once per implementation address.
    ///      Marks the current implementation as initialized in contract state.
    modifier initializer() {
        address implementation = ERC1967Upgrade._getImplementation();

        require(!isInitialized(implementation), "already initialized");

        setInitialized(implementation);

        _;
    }

    /// @dev Rejects all calls to undefined function selectors. Wormhole does not support arbitrary fallback calls.
    fallback() external payable {
        revert("unsupported");
    }

    /// @dev Rejects direct ETH transfers. The Wormhole contract only accepts ETH via `publishMessage` with the correct fee.
    receive() external payable {
        revert("the Wormhole contract does not accept assets");
    }
}
