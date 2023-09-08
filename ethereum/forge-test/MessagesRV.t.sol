// test/Messages.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/Messages.sol";
import "../contracts/Setters.sol";
import "../contracts/Structs.sol";
import "forge-test/rv-helpers/TestUtils.sol";

contract TestMessagesRV is TestUtils {
    using BytesLib for bytes;

    Messages messages;

    struct GuardianSetParams {
        uint256[] privateKeys;
        uint8 guardianCount;
        uint32 expirationTime;
    }

    function setUp() public {
        messages = new Messages();
    }

    function paramsAreWellFormed(GuardianSetParams memory params)
        internal
        pure
        returns (bool)
    {
        return params.guardianCount <= 19 &&
               params.guardianCount <= params.privateKeys.length;
    }

    function generateGuardianSet(GuardianSetParams memory params)
        internal pure
        returns (Structs.GuardianSet memory)
    {
        for (uint8 i = 0; i < params.guardianCount; ++i)
            vm.assume(0 < params.privateKeys[i] &&
                          params.privateKeys[i] < SECP256K1_CURVE_ORDER);

        address[] memory guardians = new address[](params.guardianCount);

        for (uint8 i = 0; i < params.guardianCount; ++i) {
            guardians[i] = vm.addr(params.privateKeys[i]);
        }

        return Structs.GuardianSet(guardians, params.expirationTime);
    }

    function generateSignature(
        uint8 index,
        uint256 privateKey,
        address guardian,
        bytes32 message
    )
        internal
        returns (Structs.Signature memory)
    {
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(privateKey, message);
        assertEq(ecrecover(message, v, r, s), guardian);

        return Structs.Signature(r, s, v, index);
    }

    function generateSignatures(
        uint256[] memory privateKeys,
        address[] memory guardians,
        bytes32 message
    )
        internal
        returns (Structs.Signature[] memory)
    {
        Structs.Signature[] memory sigs =
            new Structs.Signature[](guardians.length);

        for (uint8 i = 0; i < guardians.length; ++i) {
            sigs[i] = generateSignature(
                i,
                privateKeys[i],
                guardians[i],
                message
            );
        }

        return sigs;
    }

    function isProperSignature(Structs.Signature memory sig, bytes32 message)
        internal
        pure
        returns (bool)
    {
        address signer = ecrecover(message, sig.v, sig.r, sig.s);

        return signer != address(0);
    }

    function testCannotVerifySignaturesWithOutOfBoundsSignature(
        bytes memory encoded,
        GuardianSetParams memory params,
        uint8 outOfBoundsGuardian,
        uint8 outOfBoundsAmount
    ) public {
        vm.assume(encoded.length > 0);
        vm.assume(paramsAreWellFormed(params));
        vm.assume(params.guardianCount > 0);
        outOfBoundsGuardian = uint8(bound(outOfBoundsGuardian, 0, params.guardianCount - 1));
        outOfBoundsAmount = uint8(bound(outOfBoundsAmount, 0, MAX_UINT8 - params.guardianCount));

        bytes32 message = keccak256(encoded);
        Structs.GuardianSet memory guardianSet = generateGuardianSet(params);
        Structs.Signature[] memory sigs = generateSignatures(
            params.privateKeys,
            guardianSet.keys,
            keccak256(encoded)
        );

        sigs[outOfBoundsGuardian].guardianIndex =
            params.guardianCount + outOfBoundsAmount;

        vm.expectRevert("guardian index out of bounds");
        messages.verifySignatures(message, sigs, guardianSet);
    }

    function testCannotVerifySignaturesWithInvalidSignature1(
        bytes memory encoded,
        GuardianSetParams memory params,
        Structs.Signature memory fakeSignature
    ) public {
        vm.assume(encoded.length > 0);
        vm.assume(paramsAreWellFormed(params));
        vm.assume(fakeSignature.guardianIndex < params.guardianCount);

        bytes32 message = keccak256(encoded);
        Structs.GuardianSet memory guardianSet = generateGuardianSet(params);
        Structs.Signature[] memory sigs = generateSignatures(
            params.privateKeys,
            guardianSet.keys,
            message
        );

        sigs[fakeSignature.guardianIndex] = fakeSignature;

        // It is very unlikely that the arbitrary fakeSignature will be the
        // correct signature for the guardian at that index, so the below
        // should be the only reasonable outcomes
        if (isProperSignature(fakeSignature, message)) {
            (bool valid, string memory reason) =
                messages.verifySignatures(message, sigs, guardianSet);

            assertEq(valid, false);
            assertEq(reason, "VM signature invalid");
        } else {
            vm.expectRevert("ecrecover failed with signature");
            messages.verifySignatures(message, sigs, guardianSet);
        }
    }

    function testCannotVerifySignaturesWithInvalidSignature2(
        bytes memory encoded,
        GuardianSetParams memory params,
        uint8 fakeGuardianIndex,
        uint256 fakeGuardianPrivateKey
    ) public {
        vm.assume(encoded.length > 0);
        vm.assume(paramsAreWellFormed(params));
        vm.assume(fakeGuardianIndex < params.guardianCount);
        vm.assume(0 < fakeGuardianPrivateKey &&
                      fakeGuardianPrivateKey < SECP256K1_CURVE_ORDER);
        vm.assume(fakeGuardianPrivateKey != params.privateKeys[fakeGuardianIndex]);

        bytes32 message = keccak256(encoded);
        Structs.GuardianSet memory guardianSet = generateGuardianSet(params);
        Structs.Signature[] memory sigs = generateSignatures(
            params.privateKeys,
            guardianSet.keys,
            message
        );

        address fakeGuardian = vm.addr(fakeGuardianPrivateKey);
        sigs[fakeGuardianIndex] = generateSignature(
            fakeGuardianIndex,
            fakeGuardianPrivateKey,
            fakeGuardian,
            message
        );

        (bool valid, string memory reason) = messages.verifySignatures(message, sigs, guardianSet);
        assertEq(valid, false);
        assertEq(reason, "VM signature invalid");
    }

    function testVerifySignatures(
        bytes memory encoded,
        GuardianSetParams memory params
    ) public {
        vm.assume(encoded.length > 0);
        vm.assume(paramsAreWellFormed(params));

        bytes32 message = keccak256(encoded);
        Structs.GuardianSet memory guardianSet = generateGuardianSet(params);
        Structs.Signature[] memory sigs = generateSignatures(
            params.privateKeys,
            guardianSet.keys,
            message
        );

        (bool valid, string memory reason) = messages.verifySignatures(message, sigs, guardianSet);
        assertEq(valid, true);
        assertEq(bytes(reason).length, 0);
    }
}
