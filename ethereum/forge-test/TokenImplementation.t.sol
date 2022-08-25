// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "../contracts/bridge/token/TokenImplementation.sol";
import "forge-std/Test.sol";

import {console} from "forge-std/console.sol";

contract TestTokenImplementation is TokenImplementation, Test {
    struct InitiateParameters {
        string name;
        string symbol;
        uint8 decimals;
        uint64 sequence;
        address owner;
        uint16 chainId;
        bytes32 nativeContract;
    }

    function testPermit(uint256 amount, address spender) public {
        InitiateParameters memory init;
        init.name = "Valuable Token";
        init.symbol = "VALU";
        init.decimals = 8;
        init.sequence = 1;
        init.owner = 0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF;
        init.chainId = 1;
        init
            .nativeContract = 0x1337133713371337133713371337133713371337133713371337133713371337;

        initialize(
            init.name,
            init.symbol,
            init.decimals,
            init.sequence,
            init.owner,
            init.chainId,
            init.nativeContract
        );

        // prepare signer allowing for tokens to be spent
        uint256 sk = uint256(
            0xcfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
        );
        address allower = vm.addr(sk);

        // remaining arguments for `permit`
        uint256 deadline = 10;

        bytes32 PERMIT_TYPEHASH = keccak256(
            "Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)"
        );
        bytes32 structHash = keccak256(
            abi.encode(
                PERMIT_TYPEHASH,
                allower,
                spender,
                amount,
                nonces(allower),
                deadline
            )
        );

        bytes32 message = ECDSA.toTypedDataHash(DOMAIN_SEPARATOR(), structHash);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(sk, message);

        // get allowance before calling permit
        uint256 allowanceBefore = allowance(allower, spender);

        vm.assume(spender != address(0));
        permit(allower, spender, amount, deadline, v, r, s);
        uint256 allowanceAfter = allowance(allower, spender);

        require(
            allowanceAfter - allowanceBefore == amount,
            "allowance incorrect"
        );
    }

    function testFailPermitWithSameSignature(uint256 amount, address spender)
        public
    {
        InitiateParameters memory init;
        init.name = "Valuable Token";
        init.symbol = "VALU";
        init.decimals = 8;
        init.sequence = 1;
        init.owner = 0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF;
        init.chainId = 1;
        init
            .nativeContract = 0x1337133713371337133713371337133713371337133713371337133713371337;

        initialize(
            init.name,
            init.symbol,
            init.decimals,
            init.sequence,
            init.owner,
            init.chainId,
            init.nativeContract
        );

        // prepare signer allowing for tokens to be spent
        uint256 sk = uint256(
            0xcfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
        );
        address allower = vm.addr(sk);

        // remaining arguments for `permit`
        uint256 deadline = 10;

        bytes32 PERMIT_TYPEHASH = keccak256(
            "Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)"
        );
        bytes32 structHash = keccak256(
            abi.encode(
                PERMIT_TYPEHASH,
                allower,
                spender,
                amount,
                nonces(allower),
                deadline
            )
        );

        bytes32 message = ECDSA.toTypedDataHash(DOMAIN_SEPARATOR(), structHash);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(sk, message);

        vm.assume(spender != address(0));
        permit(allower, spender, amount, deadline, v, r, s);

        // try again... you shall not pass
        permit(allower, spender, amount, deadline, v, r, s);
    }

    function testFailPermitWithBadSignature(uint256 amount, address spender)
        public
    {
        InitiateParameters memory init;
        init.name = "Valuable Token";
        init.symbol = "VALU";
        init.decimals = 8;
        init.sequence = 1;
        init.owner = 0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF;
        init.chainId = 1;
        init
            .nativeContract = 0x1337133713371337133713371337133713371337133713371337133713371337;

        initialize(
            init.name,
            init.symbol,
            init.decimals,
            init.sequence,
            init.owner,
            init.chainId,
            init.nativeContract
        );

        // prepare signer allowing for tokens to be spent
        uint256 sk = uint256(
            0xcfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
        );
        address allower = vm.addr(sk);

        // remaining arguments for `permit`
        uint256 deadline = 10;

        bytes32 PERMIT_TYPEHASH = keccak256(
            "Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)"
        );
        bytes32 structHash = keccak256(
            abi.encode(
                PERMIT_TYPEHASH,
                allower,
                spender,
                amount + 1, // amount will never equal
                nonces(allower),
                deadline
            )
        );

        bytes32 message = ECDSA.toTypedDataHash(DOMAIN_SEPARATOR(), structHash);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(sk, message);

        vm.assume(spender != address(0));
        permit(allower, spender, amount, deadline, v, r, s);
    }

    function testFailPermitWithSignatureUsedAfterDeadline(
        uint256 amount,
        address spender
    ) public {
        InitiateParameters memory init;
        init.name = "Valuable Token";
        init.symbol = "VALU";
        init.decimals = 8;
        init.sequence = 1;
        init.owner = 0xDeaDbeefdEAdbeefdEadbEEFdeadbeEFdEaDbeeF;
        init.chainId = 1;
        init
            .nativeContract = 0x1337133713371337133713371337133713371337133713371337133713371337;

        initialize(
            init.name,
            init.symbol,
            init.decimals,
            init.sequence,
            init.owner,
            init.chainId,
            init.nativeContract
        );

        // prepare signer allowing for tokens to be spent
        uint256 sk = uint256(
            0xcfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
        );
        address allower = vm.addr(sk);

        // remaining arguments for `permit`
        uint256 deadline = 10;

        bytes32 PERMIT_TYPEHASH = keccak256(
            "Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)"
        );
        bytes32 structHash = keccak256(
            abi.encode(
                PERMIT_TYPEHASH,
                allower,
                spender,
                amount + 1, // amount will never equal
                nonces(allower),
                deadline
            )
        );

        bytes32 message = ECDSA.toTypedDataHash(DOMAIN_SEPARATOR(), structHash);
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(sk, message);

        // waited too long
        vm.warp(deadline + 1);

        vm.assume(spender != address(0));
        permit(allower, spender, amount, deadline, v, r, s);
    }
}
