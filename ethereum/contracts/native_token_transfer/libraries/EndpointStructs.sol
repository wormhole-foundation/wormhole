// SPDX-License-Identifier: Apache 2
pragma solidity >=0.6.12 <0.9.0;

struct EndpointManagerMessage {
    /// @notice chainId that message originates from
    uint16 chainId;
    /// @notice unique sequence number
    uint64 sequence;
    /// @notice type of the message, which determines how the payload should be decoded.
    uint8 msgType;
    /// @notice payload that corresponds to the type.
    bytes payload;
}

/// Token Transfer payload corresponding to type == 1
struct NativeTokenTransfer {
    /// @notice Amount being transferred (big-endian uint256)
    uint256 amount;
    /// @notice Address of the token. Left-zero-padded if shorter than 32 bytes
    bytes32 tokenAddress;
    /// @notice Address of the recipient. Left-zero-padded if shorter than 32 bytes
    bytes32 to;
    /// @notice Chain ID of the recipient
    uint16 toChain;
}

struct EndpointMessage {
    /// @notice Magic string (constant value set by messaging provider) that idenfies the payload as an endpoint-emitted payload.
    ///         Note that this is not a security critical field. It's meant to be used by messaging providers to identify which messages are Endpoint-related.
    bytes32 endpointId;
    /// @notice Payload provided to the Endpoint contract by the EndpointManager contract.
    bytes managerPayload;
    /// @notice Custom payload which messaging providers can use to pass bridge-specific information, if needed.
    bytes endpointPayload;
}
