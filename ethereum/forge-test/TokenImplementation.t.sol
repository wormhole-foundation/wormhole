// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/utils/cryptography/ECDSA.sol";
import "../contracts/bridge/token/TokenImplementation.sol";
import "forge-std/Test.sol";

import "forge-std/console.sol";

contract TestTokenImplementation is TokenImplementation, Test {
    uint256 constant SECP256K1_CURVE_ORDER =
        115792089237316195423570985008687907852837564279074904382605163141518161494337;

    struct InitiateParameters {
        string name;
        string symbol;
        uint8 decimals;
        uint64 sequence;
        address owner;
        uint16 chainId;
        bytes32 nativeContract;
    }

    struct SignatureSetup {
        address allower;
        bytes32 r;
        bytes32 s;
        uint8 v;
    }

    function setupTestEnvironmentWithInitialize() public {
        InitiateParameters memory init;
        init.name = "Valuable Token";
        init.symbol = "VALU";
        init.decimals = 8;
        init.sequence = 1;
        init.owner = _msgSender();
        init.chainId = 5;
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
    }

    function setupTestEnvironmentWithOldInitialize() public {
        InitiateParameters memory init;
        init.name = "Old Valuable Token";
        init.symbol = "OLD";
        init.decimals = 8;
        init.sequence = 1;
        init.owner = _msgSender();
        init.chainId = 5;
        init
            .nativeContract = 0x1337133713371337133713371337133713371337133713371337133713371337;

        _initializeNativeToken(
            init.name,
            init.symbol,
            init.decimals,
            init.sequence,
            init.owner,
            init.chainId,
            init.nativeContract
        );
    }

    function simulatePermitSignature(
        bytes32 walletPrivateKey,
        address spender,
        uint256 amount,
        uint256 deadline
    ) public view returns (SignatureSetup memory output) {
        // prepare signer allowing for tokens to be spent
        uint256 sk = uint256(walletPrivateKey);
        output.allower = vm.addr(sk);

        bytes32 PERMIT_TYPEHASH = keccak256(
            "Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)"
        );
        bytes32 structHash = keccak256(
            abi.encode(
                PERMIT_TYPEHASH,
                output.allower,
                spender,
                amount,
                nonces(output.allower),
                deadline
            )
        );

        bytes32 message = ECDSA.toTypedDataHash(DOMAIN_SEPARATOR(), structHash);
        (output.v, output.r, output.s) = vm.sign(sk, message);
    }

    // if any of these tests fail, you may have messed around with the
    // existing storage slots
    function testCheckStorageSlots() public {
        // initialize TokenImplementation
        setupTestEnvironmentWithInitialize();

        // mint some so we can check totalSupply and balances
        uint256 mintedAmount = 42069;
        _mint(_msgSender(), mintedAmount);

        // also set allowances
        uint256 allowanceAmount = 69420;
        address spender = address(0x1);
        _approve(_msgSender(), spender, allowanceAmount);

        // slot 0: name (string)
        {
            bytes32 data = vm.load(address(this), bytes32(0));
            // length 14, name = "Valuable Token"
            // data <= 31 bytes long, so length is stored at end as length * 2
            bytes memory expectedName = bytes("Valuable Token");
            require(
                uint256(data) & uint256(255) == expectedName.length * 2,
                "incorrect name length"
            );
            require(
                uint256(data) & uint256(255) == bytes(_state.name).length * 2,
                "incorrect name length"
            );
            for (uint256 i = 0; i < expectedName.length; ++i) {
                // I don't care to save this variable to storage
                require(
                    data[i] == expectedName[i],
                    "data[i] != expectedName[i]"
                );
                require(
                    data[i] == bytes(_state.name)[i],
                    "data[i] != _state.name[i]"
                );
            }
        }

        // slot 1: symbol (string)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(1)));
            // length 14, name = "Valuable Token"
            // data <= 31 bytes long, so length is stored at end as length * 2
            bytes memory expectedSymbol = bytes("VALU");
            require(
                uint256(data) & uint256(255) == expectedSymbol.length * 2,
                "incorrect symbol length"
            );
            require(
                uint256(data) & uint256(255) == bytes(_state.symbol).length * 2,
                "incorrect symbol length"
            );
            for (uint256 i = 0; i < expectedSymbol.length; ++i) {
                // I don't care to save this variable to storage
                require(
                    data[i] == expectedSymbol[i],
                    "data[i] != expectedSymbol[i]"
                );
                require(
                    data[i] == bytes(_state.symbol)[i],
                    "data[i] != _state.symbol[i]"
                );
            }
        }

        // slot 2: metaLastUpdatedSequence (uint64)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(2)));
            require(
                uint256(data) == uint256(1),
                "data != expected metaLastUpdatedSequence"
            );
            require(
                uint256(data) == uint256(_state.metaLastUpdatedSequence),
                "data != _state.metaLastUpdatedSequence"
            );
        }

        // slot 3: totalSupply (uint256)
        {
            // now verify
            bytes32 data = vm.load(address(this), bytes32(uint256(3)));
            require(
                uint256(data) == mintedAmount,
                "data != expected totalSupply"
            );
            require(
                uint256(data) == uint256(_state.totalSupply),
                "data != _state.totalSupply"
            );
        }

        // slot 4: decimals (uint8)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(4)));
            require(uint256(data) == uint256(8), "data != expected decimals");
            require(
                uint256(data) == uint256(_state.decimals),
                "data != _state.decimals"
            );
        }

        // slot 5: balances (mapping(address) => uint256)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(5)));
            require(uint256(data) == uint256(0), "data != 0");

            bytes32 mappedData = vm.load(
                address(this),
                keccak256(abi.encode(_msgSender(), 5))
            );
            require(
                uint256(mappedData) == mintedAmount,
                "data != expected balance for account"
            );
            require(
                uint256(mappedData) == _state.balances[_msgSender()],
                "data != _state.balances[_msgSender()]"
            );
        }

        // slot 6: allowances (mapping(address) => uint256)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(6)));
            require(uint256(data) == uint256(0), "data != 0");

            bytes32 mappedData = vm.load(
                address(this),
                keccak256(
                    abi.encode(spender, keccak256(abi.encode(_msgSender(), 6)))
                )
            );
            require(
                uint256(mappedData) == allowanceAmount,
                "data != expected allowance for account"
            );
            require(
                uint256(mappedData) == _state.allowances[_msgSender()][spender],
                "data != _state.allowances[_msgSender()][spender]"
            );
        }

        // slot 7: owner (address), initialized (bool), chainId (uint16)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(7)));
            require(
                (uint256(data) >> (21 * 8)) == uint256(5),
                "data[9:11] != expected chainId"
            );
            require(
                (uint256(data) >> (21 * 8)) == uint256(_state.chainId),
                "data[9:11] != _state.chainId"
            );
            require(
                uint8(data[11]) == uint8(1),
                "data[11] != expected initialized"
            );
            require(
                uint8(data[11]) == uint8(_state.initialized ? 1 : 0),
                "data[11] != _state.initialized"
            );
            require(
                uint256(data) & uint256(2**160 - 1) ==
                    uint256(uint160(_msgSender())),
                "data[12:32] != expected owner"
            );
            require(
                uint256(data) & uint256(2**160 - 1) ==
                    uint256(uint160(_state.owner)),
                "data[12:32] != _state.owner"
            );
        }

        // slot 8: nativeContract (bytes32)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(8)));
            require(
                data ==
                    0x1337133713371337133713371337133713371337133713371337133713371337,
                "data != expected nativeContract"
            );
            require(
                data == _state.nativeContract,
                "data != _state.nativeContract"
            );
        }

        // slot 9: cachedDomainSeparator (bytes32)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(9)));
            require(
                data ==
                    _buildDomainSeparator(
                        _eip712DomainNameHashed(),
                        _eip712DomainSalt()
                    ),
                "data != expected domain separator"
            );
            require(data == DOMAIN_SEPARATOR(), "data != DOMAIN_SEPARATOR()");
            require(
                data == _state.cachedDomainSeparator,
                "data != _state.cachedDomainSeparator"
            );
        }

        // slot 10: cachedChainId (uint256)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(10)));
            require(uint256(data) == block.chainid, "data != block.chainid");
            require(
                uint256(data) == _state.cachedChainId,
                "data != _state.cachedChainId"
            );
        }

        // slot 11: cachedThis (address)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(11)));
            require(
                uint256(data) == uint256(uint160(address(this))),
                "data != address(this)"
            );
            require(
                uint256(data) == uint256(uint160(_state.cachedThis)),
                "data != _state.cachedThis"
            );
        }

        // slot 12: cachedSalt (bytes32)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(12)));
            require(data == _eip712DomainSalt(), "data != expected salt");
            require(data == _state.cachedSalt, "data != _state.cachedSalt");
        }

        // slot 13: cachedHashedName (bytes32)
        {
            bytes32 data = vm.load(address(this), bytes32(uint256(13)));
            require(
                data == _eip712DomainNameHashed(),
                "data != _eip712DomainNameHashed()"
            );
            require(
                data == _state.cachedHashedName,
                "data != _state.cachedHashedName"
            );
        }
    }

    function testPermit(
        bytes32 walletPrivateKey,
        uint256 amount,
        address spender
    ) public {
        vm.assume(walletPrivateKey != bytes32(0));
        vm.assume(uint256(walletPrivateKey) < SECP256K1_CURVE_ORDER);
        vm.assume(spender != address(0));

        // initialize TokenImplementation
        setupTestEnvironmentWithInitialize();

        // prepare signer allowing for tokens to be spent
        uint256 deadline = 10;
        SignatureSetup memory signature = simulatePermitSignature(
            walletPrivateKey,
            spender,
            amount,
            deadline
        );

        // set allowance with permit
        permit(
            signature.allower,
            spender,
            amount,
            deadline,
            signature.v,
            signature.r,
            signature.s
        );

        require(
            allowance(signature.allower, spender) == amount,
            "allowance incorrect"
        );
    }

    function testFailPermitWithSameSignature(
        bytes32 walletPrivateKey,
        uint256 amount,
        address spender
    ) public {
        vm.assume(walletPrivateKey != bytes32(0));
        vm.assume(uint256(walletPrivateKey) < SECP256K1_CURVE_ORDER);
        vm.assume(spender != address(0));

        // initialize TokenImplementation
        setupTestEnvironmentWithInitialize();

        // prepare signer allowing for tokens to be spent
        uint256 deadline = 10;
        SignatureSetup memory signature = simulatePermitSignature(
            walletPrivateKey,
            spender,
            amount,
            deadline
        );

        // set allowance with permit
        permit(
            signature.allower,
            spender,
            amount,
            deadline,
            signature.v,
            signature.r,
            signature.s
        );

        // try again... you shall not pass
        // NOTE: using "testFail" instead of "test" because
        // vm.expectRevert("ERC20Permit: invalid signature") does not work
        permit(
            signature.allower,
            spender,
            amount,
            deadline,
            signature.v,
            signature.r,
            signature.s
        );
    }

    function testFailPermitWithBadSignature(
        bytes32 walletPrivateKey,
        uint256 amount,
        address spender
    ) public {
        vm.assume(walletPrivateKey != bytes32(0));
        vm.assume(uint256(walletPrivateKey) < SECP256K1_CURVE_ORDER);
        vm.assume(spender != address(0));

        // initialize TokenImplementation
        setupTestEnvironmentWithInitialize();

        // avoid overflow for this test
        uint256 wrongAmount;
        unchecked {
            wrongAmount = amount + 1; // amount will never equal
        }

        // prepare signer allowing for tokens to be spent
        uint256 deadline = 10;
        SignatureSetup memory signature = simulatePermitSignature(
            walletPrivateKey,
            spender,
            wrongAmount,
            deadline
        );

        // you shall not pass!
        // NOTE: using "testFail" instead of "test" because
        // vm.expectRevert("ERC20Permit: invalid signature") does not work
        permit(
            signature.allower,
            spender,
            amount,
            deadline,
            signature.v,
            signature.r,
            signature.s
        );
    }

    function testPermitWithSignatureUsedAfterDeadline(
        bytes32 walletPrivateKey,
        uint256 amount,
        address spender
    ) public {
        vm.assume(walletPrivateKey != bytes32(0));
        vm.assume(uint256(walletPrivateKey) < SECP256K1_CURVE_ORDER);
        vm.assume(spender != address(0));

        // initialize TokenImplementation
        setupTestEnvironmentWithInitialize();

        // prepare signer allowing for tokens to be spent
        uint256 deadline = 10;
        SignatureSetup memory signature = simulatePermitSignature(
            walletPrivateKey,
            spender,
            amount,
            deadline
        );

        // waited too long
        vm.warp(deadline + 1);

        // and fail
        vm.expectRevert("ERC20Permit: expired deadline");
        permit(
            signature.allower,
            spender,
            amount,
            deadline,
            signature.v,
            signature.r,
            signature.s
        );
    }

    function testInitializePermitState() public {
        // initialize TokenImplementation as if it were the old implementation
        setupTestEnvironmentWithOldInitialize();
        require(
            _state.cachedHashedName == bytes32(0),
            "cachedHashedName is set"
        );
        require(_state.cachedSalt == bytes32(0), "cachedSalt is set");

        // explicity call private method
        _initializePermitStateIfNeeded();
        require(
            _state.cachedHashedName == _eip712DomainNameHashed(),
            "hasnedName not cached"
        );
        require(_state.cachedSalt == _eip712DomainSalt(), "salt not cached");

        // check permit state variables
        require(
            _state.cachedChainId == block.chainid,
            "_state.cachedChainId != expected"
        );
        require(
            _state.cachedDomainSeparator ==
                keccak256(
                    abi.encode(
                        keccak256(
                            "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract,bytes32 salt)"
                        ),
                        keccak256(abi.encodePacked(name())),
                        keccak256(abi.encodePacked(_eip712DomainVersion())),
                        block.chainid,
                        address(this),
                        keccak256(abi.encodePacked(chainId(), nativeContract()))
                    )
                ),
            "_state.cachedDomainSeparator != expected"
        );
        require(
            _buildDomainSeparator(
                _eip712DomainNameHashed(),
                _eip712DomainSalt()
            ) ==
                keccak256(
                    abi.encode(
                        keccak256(
                            "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract,bytes32 salt)"
                        ),
                        keccak256(abi.encodePacked(name())),
                        keccak256(abi.encodePacked(_eip712DomainVersion())),
                        block.chainid,
                        address(this),
                        keccak256(abi.encodePacked(chainId(), nativeContract()))
                    )
                ),
            "_buildDomainSeparator() != expected"
        );
        require(
            _state.cachedThis == address(this),
            "_state.cachedThis != expected"
        );
    }

    function testPermitForPreviouslyDeployedImplementation(
        bytes32 walletPrivateKey,
        uint256 amount,
        address spender
    ) public {
        vm.assume(walletPrivateKey != bytes32(0));
        vm.assume(uint256(walletPrivateKey) < SECP256K1_CURVE_ORDER);
        vm.assume(spender != address(0));

        // initialize TokenImplementation as if it were the old implementation
        setupTestEnvironmentWithOldInitialize();
        require(_state.cachedSalt == bytes32(0), "cachedSalt is set");

        // prepare signer allowing for tokens to be spent
        uint256 deadline = 10;
        SignatureSetup memory signature = simulatePermitSignature(
            walletPrivateKey,
            spender,
            amount,
            deadline
        );

        // set allowance with permit
        permit(
            signature.allower,
            spender,
            amount,
            deadline,
            signature.v,
            signature.r,
            signature.s
        );

        require(
            allowance(signature.allower, spender) == amount,
            "allowance incorrect"
        );
    }

    // used to prevent stack too deep in test
    struct Eip712DomainOutput {
        bytes1 fields;
        string name;
        string version;
        uint256 chainId;
        address verifyingContract;
        bytes32 salt;
        uint256[] extensions;
    }

    function testPermitUsingEip712DomainValues(
        bytes32 walletPrivateKey,
        uint256 amount,
        address spender
    ) public {
        vm.assume(walletPrivateKey != bytes32(0));
        vm.assume(uint256(walletPrivateKey) < SECP256K1_CURVE_ORDER);
        vm.assume(spender != address(0));

        // initialize TokenImplementation
        setupTestEnvironmentWithInitialize();

        Eip712DomainOutput memory domain;
        (
            domain.fields,
            domain.name,
            domain.version,
            domain.chainId,
            domain.verifyingContract,
            domain.salt,
            domain.extensions
        ) = eip712Domain();
        require(domain.fields == hex"1F", "domainFields != expected");
        require(
            keccak256(abi.encodePacked(domain.name)) ==
                keccak256(abi.encodePacked(name())),
            "domainName != expected"
        );
        require(
            keccak256(abi.encodePacked(domain.name)) ==
                _eip712DomainNameHashed(),
            "domainName != _eip712DomainNameHashed()"
        );
        require(
            keccak256(abi.encodePacked(domain.version)) ==
                keccak256(abi.encodePacked("1")),
            "domainVersion != expected"
        );
        require(
            keccak256(abi.encodePacked(domain.version)) ==
                keccak256(abi.encodePacked(_eip712DomainVersion())),
            "domainVersion != _eip712DomainVersion()"
        );
        require(domain.chainId == block.chainid, "domainFields != expected");
        require(
            domain.chainId == _state.cachedChainId,
            "domainFields != _state.cachedChainId"
        );
        require(
            domain.verifyingContract == address(this),
            "domainVerifyingContract != expected"
        );
        require(
            domain.verifyingContract == _state.cachedThis,
            "domainVerifyingContract != _state.cachedThis"
        );
        require(
            domain.salt ==
                keccak256(abi.encodePacked(chainId(), nativeContract())),
            "domainFields != expected"
        );
        require(
            domain.salt == _eip712DomainSalt(),
            "domainFields != _eip712DomainSalt()"
        );
        require(
            domain.salt == _state.cachedSalt,
            "domainFields != _state.cachedSalt"
        );
        require(domain.extensions.length == 0, "domainExtensions.length != 0");

        // prepare signer allowing for tokens to be spent
        SignatureSetup memory signature;
        uint256 sk = uint256(walletPrivateKey);
        signature.allower = vm.addr(sk);

        uint256 deadline = 10;

        bytes32 structHash = keccak256(
            abi.encode(
                keccak256(
                    "Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)"
                ),
                signature.allower,
                spender,
                amount,
                nonces(signature.allower),
                deadline
            )
        );

        // build domain separator by hand using eip712Domain() output
        bytes32 domainSeparator = keccak256(
            abi.encode(
                keccak256(
                    "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract,bytes32 salt)"
                ),
                keccak256(abi.encodePacked(domain.name)),
                keccak256(abi.encodePacked(domain.version)),
                domain.chainId,
                domain.verifyingContract,
                domain.salt
            )
        );

        // sign and set allowance with permit
        (signature.v, signature.r, signature.s) = vm.sign(
            sk,
            ECDSA.toTypedDataHash(domainSeparator, structHash)
        );
        permit(
            signature.allower,
            spender,
            amount,
            deadline,
            signature.v,
            signature.r,
            signature.s
        );

        require(
            allowance(signature.allower, spender) == amount,
            "allowance incorrect"
        );
    }

    function testPermitAfterUpdateDetails(
        bytes32 walletPrivateKey,
        uint256 amount,
        address spender,
        string calldata newName
    ) public {
        vm.assume(walletPrivateKey != bytes32(0));
        vm.assume(uint256(walletPrivateKey) < SECP256K1_CURVE_ORDER);
        vm.assume(spender != address(0));
        vm.assume(bytes(newName).length <= 32);

        // initialize TokenImplementation
        setupTestEnvironmentWithInitialize();

        string memory oldName = name();
        bytes32 oldDomainSeparator = _state.cachedDomainSeparator;

        // permit before updateDetails
        {
            uint256 deadline = 10;
            SignatureSetup memory signature = simulatePermitSignature(
                walletPrivateKey,
                spender,
                amount,
                deadline
            );

            // set allowance with permit
            permit(
                signature.allower,
                spender,
                amount,
                deadline,
                signature.v,
                signature.r,
                signature.s
            );

            require(
                allowance(signature.allower, spender) == amount,
                "allowance incorrect"
            );

            // revoke allowance to prep for next test
            _approve(signature.allower, spender, 0);
        }

        // asset metadata updated here
        updateDetails(
            newName,
            "NEW", // new symbol
            _state.metaLastUpdatedSequence + 1 // new sequence
        );

        require(
            keccak256(abi.encodePacked(newName)) !=
                keccak256(abi.encodePacked(oldName)),
            "newName == oldName"
        );
        require(
            _domainSeparatorV4() != oldDomainSeparator,
            "_domainSeparatorV4() == oldDomainSeparator"
        );
        require(
            _state.cachedDomainSeparator != oldDomainSeparator,
            "_state.cachedDomainSeparator == oldDomainSeparator"
        );
        require(
            _state.cachedDomainSeparator == _domainSeparatorV4(),
            "_state.cachedDomainSeparator != _domainSeparatorV4()"
        );

        // permit after updateDetails
        {
            uint256 deadline = 10;
            SignatureSetup memory signature = simulatePermitSignature(
                walletPrivateKey,
                spender,
                amount,
                deadline
            );

            // set allowance with permit
            permit(
                signature.allower,
                spender,
                amount,
                deadline,
                signature.v,
                signature.r,
                signature.s
            );

            require(
                allowance(signature.allower, spender) == amount,
                "allowance incorrect"
            );
        }
    }

    function testPermitForOldSalt(
        bytes32 walletPrivateKey,
        uint256 amount,
        address spender
    ) public {
        vm.assume(walletPrivateKey != bytes32(0));
        vm.assume(uint256(walletPrivateKey) < SECP256K1_CURVE_ORDER);
        vm.assume(spender != address(0));

        // initialize TokenImplementation
        setupTestEnvironmentWithInitialize();

        // hijack salt
        _state.cachedSalt = keccak256(abi.encodePacked("definitely not right"));
        require(
            _state.cachedSalt != _eip712DomainSalt(),
            "_state.cachedSalt == _eip712DomainSalt()"
        );

        // prepare signer allowing for tokens to be spent
        uint256 deadline = 10;
        SignatureSetup memory signature = simulatePermitSignature(
            walletPrivateKey,
            spender,
            amount,
            deadline
        );

        // set allowance with permit
        permit(
            signature.allower,
            spender,
            amount,
            deadline,
            signature.v,
            signature.r,
            signature.s
        );

        // verify salt is correct
        require(
            _state.cachedSalt == _eip712DomainSalt(),
            "_state.cachedSalt != _eip712DomainSalt()"
        );
        // then allowance
        require(
            allowance(signature.allower, spender) == amount,
            "allowance incorrect"
        );
    }

    function testPermitForOldName(
        bytes32 walletPrivateKey,
        uint256 amount,
        address spender
    ) public {
        vm.assume(walletPrivateKey != bytes32(0));
        vm.assume(uint256(walletPrivateKey) < SECP256K1_CURVE_ORDER);
        vm.assume(spender != address(0));

        // initialize TokenImplementation
        setupTestEnvironmentWithInitialize();

        // hijack name
        _state.cachedHashedName = keccak256("definitely not right");
        require(
            _state.cachedHashedName != _eip712DomainNameHashed(),
            "_state.cachedHashedName == _eip712DomainNameHashed()"
        );

        // prepare signer allowing for tokens to be spent
        uint256 deadline = 10;
        SignatureSetup memory signature = simulatePermitSignature(
            walletPrivateKey,
            spender,
            amount,
            deadline
        );

        // set allowance with permit
        permit(
            signature.allower,
            spender,
            amount,
            deadline,
            signature.v,
            signature.r,
            signature.s
        );

        // verify name is correct
        require(
            _state.cachedHashedName == _eip712DomainNameHashed(),
            "_state.cachedHashedName != _eip712DomainNameHashed()"
        );
        // then allowance
        require(
            allowance(signature.allower, spender) == amount,
            "allowance incorrect"
        );
    }
}
