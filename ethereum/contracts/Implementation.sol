// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "./Governance.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

/// @title Wormhole Governance Implementation Contract upgrades
/// @notice This contract is an implementation of Wormhole's Governance contract.
contract Implementation is Governance {

    /// @notice Emitted when a message is published to the Wormhole network
    event LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel);

    /// @notice Publish a message to be attested by the Wormhole network
    /// @param nonce The nonce to ensure message uniqueness
    /// @param payload The content of the emitted message to be published, an arbitrary byte array
    /// @param consistencyLevel The level of finality to reach before the guardians will observe and attest the emitted event
    /// @return sequence A number that is unique and increments for every message for a given emitter (and implicitly chain)
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

    /// @dev Update and fetch the current sequence for a given emitter
    /// @param emitter The address that emits a message
    /// @return sequence The next sequence for a given emitter
    function useSequence(address emitter) internal returns (uint64 sequence) {
        sequence = nextSequence(emitter);
        setNextSequence(emitter, sequence + 1);
    }

    /// @notice Initializes the contract with the EVM chain ID mapping upon deployment or upgrade
    function initialize() initializer public virtual {
        // this function needs to be exposed for an upgrade to pass
        if (evmChainId() == 0) {
            uint256 evmChainId;
            uint16 chain = chainId();

            // Wormhole chain ids explicitly enumerated
            if        (chain == 2)  { evmChainId = 1;          // ethereum
            } else if (chain == 4)  { evmChainId = 56;         // bsc
            } else if (chain == 5)  { evmChainId = 137;        // polygon
            } else if (chain == 6)  { evmChainId = 43114;      // avalanche
            } else if (chain == 7)  { evmChainId = 42262;      // oasis
            } else if (chain == 9)  { evmChainId = 1313161554; // aurora
            } else if (chain == 10) { evmChainId = 250;        // fantom
            } else if (chain == 11) { evmChainId = 686;        // karura
            } else if (chain == 12) { evmChainId = 787;        // acala
            } else if (chain == 13) { evmChainId = 8217;       // klaytn
            } else if (chain == 14) { evmChainId = 42220;      // celo
            } else if (chain == 16) { evmChainId = 1284;       // moonbeam
            } else if (chain == 17) { evmChainId = 245022934;  // neon
            } else if (chain == 23) { evmChainId = 42161;      // arbitrum
            } else if (chain == 24) { evmChainId = 10;         // optimism
            } else if (chain == 25) { evmChainId = 100;        // gnosis
            } else {
                revert("Unknown chain id.");
            }

            setEvmChainId(evmChainId);
        }
    }

    /// @dev Modifier to ensure the contract is only initialized once
    modifier initializer() {
        address implementation = ERC1967Upgrade._getImplementation();

        require(
            !isInitialized(implementation),
            "already initialized"
        );

        setInitialized(implementation);

        _;
    }

    /// @dev Fallback function that reverts all calls
    fallback() external payable {revert("unsupported");}

    /// @dev Receive function that reverts all transactions with assets
    receive() external payable {revert("the Wormhole contract does not accept assets");}
}
