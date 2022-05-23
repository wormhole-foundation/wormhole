// contracts/Implementation.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;
pragma experimental ABIEncoderV2;

import "./Governance.sol";

import "@openzeppelin/contracts/proxy/ERC1967/ERC1967Upgrade.sol";

contract Implementation is Governance {
    event LogMessagePublished(
        address indexed sender,
        uint64 sequence,
        uint32 nonce,
        bytes payload,
        uint8 consistencyLevel
    );

    // Publish a message to be attested by the Wormhole network
    function publishMessage(
        uint32 nonce,
        bytes memory payload,
        uint8 consistencyLevel
    ) public payable returns (uint64 sequence) {
        // check fee
        require(msg.value == messageFee(), "invalid fee");

        sequence = useSequence(msg.sender);
        // emit log
        emit LogMessagePublished(
            msg.sender,
            sequence,
            nonce,
            payload,
            consistencyLevel
        );
    }

    function useSequence(address emitter) internal returns (uint64 sequence) {
        sequence = nextSequence(emitter);
        setNextSequence(emitter, sequence + 1);
    }

    modifier initializer() {
        address implementation = ERC1967Upgrade._getImplementation();

        require(!isInitialized(implementation), "already initialized");

        setInitialized(implementation);

        _;
    }

    fallback() external payable {
        revert("unsupported");
    }

    receive() external payable {
        revert("the Wormhole contract does not accept assets");
    }
}
