// SPDX-License-Identifier: Apache 2
pragma solidity ^0.8.0;

import {IWormhole} from "../../contracts/interfaces/IWormhole.sol";
import {MockWormhole} from "./MockWormhole.sol";
import "../../contracts/libraries/external/BytesLib.sol";

import "forge-std/Vm.sol";
import "forge-std/console.sol";

/**
 * @notice These are the common parts for the signing and the non signing wormhole simulators.
 * @dev This contract is meant to be used when testing against a mainnet fork.
 */
abstract contract WormholeSimulator {
    using BytesLib for bytes;

    function doubleKeccak256(bytes memory body) internal pure returns (bytes32) {
        return keccak256(abi.encodePacked(keccak256(body)));
    }

    function parseVMFromLogs(Vm.Log memory log) public pure returns (IWormhole.VM memory vm_) {
        uint256 index = 0;

        // emitterAddress
        vm_.emitterAddress = bytes32(log.topics[1]);

        // sequence
        vm_.sequence = log.data.toUint64(index + 32 - 8);
        index += 32;

        // nonce
        vm_.nonce = log.data.toUint32(index + 32 - 4);
        index += 32;

        // skip random bytes
        index += 32;

        // consistency level
        vm_.consistencyLevel = log.data.toUint8(index + 32 - 1);
        index += 32;

        // length of payload
        uint256 payloadLen = log.data.toUint256(index);
        index += 32;

        vm_.payload = log.data.slice(index, payloadLen);
        index += payloadLen;

        // trailing bytes (due to 32 byte slot overlap)
        index += log.data.length - index;

        require(index == log.data.length, "failed to parse wormhole message");
    }

    /**
     * @notice Finds published Wormhole events in forge logs
     * @param logs The forge Vm.log captured when recording events during test execution
     */
    function fetchWormholeMessageFromLog(Vm.Log[] memory logs)
        public
        pure
        returns (Vm.Log[] memory)
    {
        uint256 count = 0;
        for (uint256 i = 0; i < logs.length; i++) {
            if (
                logs[i].topics[0]
                    == keccak256("LogMessagePublished(address,uint64,uint32,bytes,uint8)")
            ) {
                count += 1;
            }
        }

        // create log array to save published messages
        Vm.Log[] memory published = new Vm.Log[](count);

        uint256 publishedIndex = 0;
        for (uint256 i = 0; i < logs.length; i++) {
            if (
                logs[i].topics[0]
                    == keccak256("LogMessagePublished(address,uint64,uint32,bytes,uint8)")
            ) {
                published[publishedIndex] = logs[i];
                publishedIndex += 1;
            }
        }

        return published;
    }

    /**
     * @notice Encodes Wormhole message body into bytes
     * @param vm_ Wormhole VM struct
     * @return encodedObservation Wormhole message body encoded into bytes
     */
    function encodeObservation(IWormhole.VM memory vm_)
        public
        pure
        returns (bytes memory encodedObservation)
    {
        encodedObservation = abi.encodePacked(
            vm_.timestamp,
            vm_.nonce,
            vm_.emitterChainId,
            vm_.emitterAddress,
            vm_.sequence,
            vm_.consistencyLevel,
            vm_.payload
        );
    }

    /**
     * @notice Formats and signs a simulated Wormhole message using the emitted log from calling `publishMessage`
     * @param log The forge Vm.log captured when recording events during test execution
     * @return signedMessage Formatted and signed Wormhole message
     */
    function fetchSignedMessageFromLogs(
        Vm.Log memory log,
        uint16 emitterChainId,
        address emitterAddress
    ) public returns (bytes memory signedMessage) {
        // Parse wormhole message from ethereum logs
        IWormhole.VM memory vm_ = parseVMFromLogs(log);

        // Set empty body values before computing the hash
        vm_.version = uint8(1);
        vm_.timestamp = uint32(block.timestamp);
        vm_.emitterChainId = emitterChainId;
        vm_.emitterAddress = bytes32(uint256(uint160(emitterAddress)));

        return encodeAndSignMessage(vm_);
    }

    /**
     * Functions that must be implemented by concrete wormhole simulators.
     */

    /**
     * @notice Sets the message fee for a wormhole message.
     */
    function setMessageFee(uint256 newFee) public virtual;

    /**
     * @notice Invalidates a VM. It must be executed before it is parsed and verified by the Wormhole instance to work.
     */
    function invalidateVM(bytes memory message) public virtual;

    /**
     * @notice Signs and preformatted simulated Wormhole message
     * @param vm_ The preformatted Wormhole message
     * @return signedMessage Formatted and signed Wormhole message
     */
    function encodeAndSignMessage(IWormhole.VM memory vm_)
        public
        virtual
        returns (bytes memory signedMessage);
}

/**
 * @title A Wormhole Guardian Simulator
 * @notice This contract simulates signing Wormhole messages emitted in a forge test.
 * This particular version doesn't sign any message but just exists to keep a standard interface for tests.
 * @dev This contract is meant to be used with the MockWormhole contract that validates any VM as long
 *   as its hash wasn't banned.
 */
contract FakeWormholeSimulator is WormholeSimulator {
    // Allow access to Wormhole
    MockWormhole public wormhole;

    /**
     * @param initWormhole address of the Wormhole core contract for the mainnet chain being forked
     */
    constructor(MockWormhole initWormhole) {
        wormhole = initWormhole;
    }

    function setMessageFee(uint256 newFee) public override {
        wormhole.setMessageFee(newFee);
    }

    function invalidateVM(bytes memory message) public override {
        wormhole.invalidateVM(message);
    }

    /**
     * @notice Signs and preformatted simulated Wormhole message
     * @param vm_ The preformatted Wormhole message
     * @return signedMessage Formatted and signed Wormhole message
     */
    function encodeAndSignMessage(IWormhole.VM memory vm_)
        public
        view
        override
        returns (bytes memory signedMessage)
    {
        // Compute the hash of the body
        bytes memory body = encodeObservation(vm_);
        vm_.hash = doubleKeccak256(body);

        signedMessage = abi.encodePacked(
            vm_.version,
            wormhole.getCurrentGuardianSetIndex(),
            // length of signature array
            uint8(1),
            // guardian index
            uint8(0),
            // r sig argument
            bytes32(uint256(0)),
            // s sig argument
            bytes32(uint256(0)),
            // v sig argument (encodes public key recovery id, public key type and network of the signature)
            uint8(0),
            body
        );
    }
}

/**
 * @title A Wormhole Guardian Simulator
 * @notice This contract simulates signing Wormhole messages emitted in a forge test.
 * It overrides the Wormhole guardian set to allow for signing messages with a single
 * private key on any EVM where Wormhole core contracts are deployed.
 * @dev This contract is meant to be used when testing against a mainnet fork.
 */
contract SigningWormholeSimulator is WormholeSimulator {
    // Taken from forge-std/Script.sol
    address private constant VM_ADDRESS =
        address(bytes20(uint160(uint256(keccak256("hevm cheat code")))));
    Vm public constant vm = Vm(VM_ADDRESS);

    // Allow access to Wormhole
    IWormhole public wormhole;

    // Save the guardian PK to sign messages with
    uint256 private devnetGuardianPK;

    /**
     * @param wormhole_ address of the Wormhole core contract for the mainnet chain being forked
     * @param devnetGuardian private key of the devnet Guardian
     */
    constructor(IWormhole wormhole_, uint256 devnetGuardian) {
        wormhole = wormhole_;
        devnetGuardianPK = devnetGuardian;
        overrideToDevnetGuardian(vm.addr(devnetGuardian));
    }

    function overrideToDevnetGuardian(address devnetGuardian) internal {
        {
            // Get slot for Guardian Set at the current index
            uint32 guardianSetIndex = wormhole.getCurrentGuardianSetIndex();
            bytes32 guardianSetSlot = keccak256(abi.encode(guardianSetIndex, 2));

            // Overwrite all but first guardian set to zero address. This isn't
            // necessary, but just in case we inadvertently access these slots
            // for any reason.
            uint256 numGuardians = uint256(vm.load(address(wormhole), guardianSetSlot));
            for (uint256 i = 1; i < numGuardians;) {
                vm.store(
                    address(wormhole),
                    bytes32(uint256(keccak256(abi.encodePacked(guardianSetSlot))) + i),
                    bytes32(0)
                );
                unchecked {
                    i += 1;
                }
            }

            // Now overwrite the first guardian key with the devnet key specified
            // in the function argument.
            vm.store(
                address(wormhole),
                bytes32(uint256(keccak256(abi.encodePacked(guardianSetSlot))) + 0), // just explicit w/ index 0
                bytes32(uint256(uint160(devnetGuardian)))
            );

            // Change the length to 1 guardian
            vm.store(
                address(wormhole),
                guardianSetSlot,
                bytes32(uint256(1)) // length == 1
            );

            // Confirm guardian set override
            address[] memory guardians = wormhole.getGuardianSet(guardianSetIndex).keys;
            require(guardians.length == 1, "guardians.length != 1");
            require(guardians[0] == devnetGuardian, "incorrect guardian set override");
        }
    }

    function setMessageFee(uint256 newFee) public override {
        bytes32 coreModule = 0x00000000000000000000000000000000000000000000000000000000436f7265;
        bytes memory message =
            abi.encodePacked(coreModule, uint8(3), uint16(wormhole.chainId()), newFee);
        IWormhole.VM memory preSignedMessage = IWormhole.VM({
            version: 1,
            timestamp: uint32(block.timestamp),
            nonce: 0,
            emitterChainId: wormhole.governanceChainId(),
            emitterAddress: wormhole.governanceContract(),
            sequence: 0,
            consistencyLevel: 200,
            payload: message,
            guardianSetIndex: 0,
            signatures: new IWormhole.Signature[](0),
            hash: bytes32("")
        });

        bytes memory signed = encodeAndSignMessage(preSignedMessage);
        wormhole.submitSetMessageFee(signed);
    }

    function invalidateVM(bytes memory message) public pure override {
        // Don't do anything. Signatures are easily invalidated modifying the payload.
        // If it becomes necessary to prevent producing a good signature for this message, that can be done here.
    }

    /**
     * @notice Signs and preformatted simulated Wormhole message
     * @param vm_ The preformatted Wormhole message
     * @return signedMessage Formatted and signed Wormhole message
     */
    function encodeAndSignMessage(IWormhole.VM memory vm_)
        public
        view
        override
        returns (bytes memory signedMessage)
    {
        // Compute the hash of the body
        bytes memory body = encodeObservation(vm_);
        vm_.hash = doubleKeccak256(body);

        // Sign the hash with the devnet guardian private key
        IWormhole.Signature[] memory sigs = new IWormhole.Signature[](1);
        (sigs[0].v, sigs[0].r, sigs[0].s) = vm.sign(devnetGuardianPK, vm_.hash);
        sigs[0].guardianIndex = 0;

        signedMessage = abi.encodePacked(
            vm_.version,
            wormhole.getCurrentGuardianSetIndex(),
            uint8(sigs.length),
            sigs[0].guardianIndex,
            sigs[0].r,
            sigs[0].s,
            sigs[0].v - 27,
            body
        );
    }
}
