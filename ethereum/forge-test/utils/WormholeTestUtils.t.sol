// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../contracts/Implementation.sol";
import "../../contracts/Setup.sol";
import "../../contracts/Wormhole.sol";
import "../../contracts/interfaces/IWormhole.sol";

import "forge-std/Test.sol";

abstract contract WormholeTestUtils is Test {
    function setUpWormhole(uint8 numGuardians) public returns (address) {
        Implementation wormholeImpl = new Implementation();
        Setup wormholeSetup = new Setup();

        Wormhole wormhole = new Wormhole(address(wormholeSetup), new bytes(0));

        address[] memory initSigners = new address[](numGuardians);

        for (uint256 i = 0; i < numGuardians; ++i) {
            initSigners[i] = vm.addr(i + 1); // i+1 is the private key for the i-th signer.
        }

        // These values are the default values used in our tilt test environment
        // and are not important.
        Setup(address(wormhole)).setup(
            address(wormholeImpl),
            initSigners,
            2, // Ethereum chain ID
            1, // Governance source chain ID (1 = solana)
            0x0000000000000000000000000000000000000000000000000000000000000004, // Governance source address
            block.chainid // EVM chain Id must be same as block.chainid
        );

        return address(wormhole);
    }

    function generateVaa(
        uint32 timestamp,
        uint16 emitterChainId,
        bytes32 emitterAddress,
        uint64 sequence,
        bytes memory payload,
        uint8 numSigners
    ) public returns (bytes memory vaa) {
        bytes memory body = abi.encodePacked(
            timestamp,
            uint32(0), // Nonce. It is zero for single VAAs.
            emitterChainId,
            emitterAddress,
            sequence,
            uint8(0), // Consistency level (sometimes no. confirmation block). Not important here.
            payload
        );

        bytes32 hash = keccak256(abi.encodePacked(keccak256(body)));

        bytes memory signatures = new bytes(0);

        for (uint256 i = 0; i < numSigners; ++i) {
            (uint8 v, bytes32 r, bytes32 s) = vm.sign(i + 1, hash);
            // encodePacked uses padding for arrays and we don't want it, so we manually concat them.
            signatures = abi.encodePacked(
                signatures,
                uint8(i), // Guardian index of the signature
                r,
                s,
                v - 27 // v is either 27 or 28. 27 is added to v in Eth (following BTC) but Wormhole doesn't use it.
            );
        }

        vaa = abi.encodePacked(
            uint8(1), // Version
            uint32(0), // Guardian set index. it is initialized by 0
            numSigners,
            signatures,
            body
        );
    }
}

contract WormholeTestUtilsTest is Test, WormholeTestUtils {
    function testGenerateVaaWorks() public {
        IWormhole wormhole = IWormhole(setUpWormhole(5));

        bytes memory vaa = generateVaa(
            112,
            7,
            0x0000000000000000000000000000000000000000000000000000000000000bad,
            10,
            hex"deadbeaf",
            4
        );

        (IWormhole.VM memory vm, bool valid, ) = wormhole.parseAndVerifyVM(vaa);
        assertTrue(valid);

        assertEq(vm.timestamp, 112);
        assertEq(vm.emitterChainId, 7);
        assertEq(
            vm.emitterAddress,
            0x0000000000000000000000000000000000000000000000000000000000000bad
        );
        assertEq(vm.payload, hex"deadbeaf");
        assertEq(vm.signatures.length, 4);
    }
}
