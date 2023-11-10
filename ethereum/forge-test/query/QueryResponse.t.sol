// SPDX-License-Identifier: Apache 2

// forge test --match-contract QueryResponse

pragma solidity ^0.8.0;

import "../../contracts/query/QueryResponse.sol";
import "../../contracts/Implementation.sol";
import "../../contracts/Setup.sol";
import "../../contracts/Wormhole.sol";
import "forge-std/Test.sol";

// @dev A non-abstract QueryResponse contract
contract QueryResponseContract is QueryResponse { }

contract TestQueryResponse is Test {
    // Some happy case defaults
    uint8 version = 0x01;
    uint16 senderChainId = 0x0000;
    bytes signature = hex"ff0c222dc9e3655ec38e212e9792bf1860356d1277462b6bf747db865caca6fc08e6317b64ee3245264e371146b1d315d38c867fe1f69614368dc4430bb560f200";
    uint32 queryRequestLen = 0x00000053;
    uint8 queryRequestVersion = 0x01;
    uint32 queryRequestNonce = 0xdd9914c6;
    uint8 numPerChainQueries = 0x01;
    bytes perChainQueries = hex"0005010000004600000009307832613631616334020d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde030d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000418160ddd";
    bytes perChainQueriesInner = hex"00000009307832613631616334020d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde030d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000418160ddd";
    uint8 numPerChainResponses = 0x01;
    bytes perChainResponses = hex"000501000000b90000000002a61ac4c1adff9f6e180309e7d0d94c063338ddc61c1c4474cd6957c960efe659534d040005ff312e4f90c002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d57726170706564204d6174696300000000000000000000000000000000000000000000200000000000000000000000000000000000000000007ae5649beabeddf889364a";
    bytes perChainResponsesInner = hex"00000009307832613631616334020d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde030d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000418160ddd";

    uint8 sigGuardianIndex = 0;

    Wormhole wormhole;
    QueryResponse queryResponse;

    function setUp() public {
        wormhole = deployWormholeForTest();
        queryResponse = new QueryResponseContract();
    }

    uint16 constant TEST_CHAIN_ID = 2;
    address constant DEVNET_GUARDIAN = 0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe;
    uint256 constant DEVNET_GUARDIAN_PRIVATE_KEY = 0xcfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0;
    uint16 constant GOVERNANCE_CHAIN_ID = 1;
    bytes32 constant GOVERNANCE_CONTRACT = 0x0000000000000000000000000000000000000000000000000000000000000004;

    function getSignature(bytes memory response) internal view returns (uint8 v, bytes32 r, bytes32 s) {
        bytes32 responseDigest = queryResponse.getResponseDigest(response);
        (v, r, s) = vm.sign(DEVNET_GUARDIAN_PRIVATE_KEY, responseDigest);
    }

    function concatenateQueryResponseBytesOffChain(
        uint8 _version,
        uint16 _senderChainId,
        bytes memory _signature,
        uint32 _queryRequestLen,
        uint8 _queryRequestVersion,
        uint32 _queryRequestNonce,
        uint8 _numPerChainQueries,
        bytes memory _perChainQueries,
        uint8 _numPerChainResponses,
        bytes memory _perChainResponses
    ) internal pure returns (bytes memory){
        return abi.encodePacked(
            _version,
            _senderChainId,
            _signature,
            _queryRequestLen,
            _queryRequestVersion,
            _queryRequestNonce,
            _numPerChainQueries,
            _perChainQueries,
            _numPerChainResponses,
            _perChainResponses
        );
    }

    function deployWormholeForTest() public returns (Wormhole) {
        // Deploy the Setup contract.
        Setup setup = new Setup();

        // Deploy the Implementation contract.
        Implementation implementation = new Implementation();

        address[] memory guardians = new address[](1);
        guardians[0] = DEVNET_GUARDIAN;

        // Deploy the Wormhole contract.
        wormhole = new Wormhole(
            address(setup),
            abi.encodeWithSelector(
                bytes4(keccak256("setup(address,address[],uint16,uint16,bytes32,uint256)")),
                address(implementation),
                guardians,
                TEST_CHAIN_ID,
                GOVERNANCE_CHAIN_ID,
                GOVERNANCE_CONTRACT,
                block.chainid // evm chain id
            )
        );

        return wormhole;
    }

    function test_getResponseHash() public {
        bytes memory resp = concatenateQueryResponseBytesOffChain(version, senderChainId, signature, queryRequestLen, queryRequestVersion, queryRequestNonce, numPerChainQueries, perChainQueries, numPerChainResponses, perChainResponses);
        bytes32 hash = queryResponse.getResponseHash(resp);
        bytes32 expectedHash = 0xed18e80906ffa80ce953a132a9cbbcf84186955f8fc8ce0322cd68622a58570e;
        assertEq(hash, expectedHash);
    }

    function test_getResponseDigest() public {
        bytes memory resp = concatenateQueryResponseBytesOffChain(version, senderChainId, signature, queryRequestLen, queryRequestVersion, queryRequestNonce, numPerChainQueries, perChainQueries, numPerChainResponses, perChainResponses);
        bytes32 digest = queryResponse.getResponseDigest(resp);
        bytes32 expectedDigest = 0x5b84b19c68ee0b37899230175a92ee6eda4c5192e8bffca1d057d811bb3660e2;
        assertEq(digest, expectedDigest);
    }

    function test_verifyQueryResponseSignatures() public view {
        bytes memory resp = concatenateQueryResponseBytesOffChain(version, senderChainId, signature, queryRequestLen, queryRequestVersion, queryRequestNonce, numPerChainQueries, perChainQueries, numPerChainResponses, perChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        queryResponse.verifyQueryResponseSignatures(address(wormhole), resp, signatures);
        // TODO: There are no assertions for this test
    }

    function test_parseAndVerifyQueryResponse() public {
        bytes memory resp = concatenateQueryResponseBytesOffChain(version, senderChainId, signature, queryRequestLen, queryRequestVersion, queryRequestNonce, numPerChainQueries, perChainQueries, numPerChainResponses, perChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        ParsedQueryResponse memory r = queryResponse.parseAndVerifyQueryResponse(address(wormhole), resp, signatures);
        assertEq(r.version, 1);
        assertEq(r.senderChainId, 0);
        assertEq(r.requestId, hex"ff0c222dc9e3655ec38e212e9792bf1860356d1277462b6bf747db865caca6fc08e6317b64ee3245264e371146b1d315d38c867fe1f69614368dc4430bb560f200");
        assertEq(r.nonce, 3717797062);
        assertEq(r.responses.length, 1);
        assertEq(r.responses[0].chainId, 5);
        assertEq(r.responses[0].queryType, 1);
        assertEq(r.responses[0].request, hex"00000009307832613631616334020d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde030d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000418160ddd");
        assertEq(r.responses[0].response, hex"0000000002a61ac4c1adff9f6e180309e7d0d94c063338ddc61c1c4474cd6957c960efe659534d040005ff312e4f90c002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d57726170706564204d6174696300000000000000000000000000000000000000000000200000000000000000000000000000000000000000007ae5649beabeddf889364a");
    }

    function test_parseEthCallQueryResponse() public {
        // Take the data extracted by the previous test and break it down even further.
        ParsedPerChainQueryResponse memory r = ParsedPerChainQueryResponse({
            chainId: 5,
            queryType: 1,
            request: hex"00000009307832613631616334020d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde030d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000418160ddd",
            response: hex"0000000002a61ac4c1adff9f6e180309e7d0d94c063338ddc61c1c4474cd6957c960efe659534d040005ff312e4f90c002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d57726170706564204d6174696300000000000000000000000000000000000000000000200000000000000000000000000000000000000000007ae5649beabeddf889364a"
            });

        EthCallQueryResponse memory eqr = queryResponse.parseEthCallQueryResponse(r);
        assertEq(eqr.requestBlockId, hex"307832613631616334");
        assertEq(eqr.blockNum, 44440260);
        assertEq(eqr.blockHash, hex"c1adff9f6e180309e7d0d94c063338ddc61c1c4474cd6957c960efe659534d04");
        assertEq(eqr.blockTime, 1687961579000000);
        assertEq(eqr.result.length, 2);

        assertEq(eqr.result[0].contractAddress, address(0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270));
        assertEq(eqr.result[0].callData, hex"06fdde03");
        assertEq(eqr.result[0].result, hex"0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d57726170706564204d6174696300000000000000000000000000000000000000");

        assertEq(eqr.result[1].contractAddress, address(0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270));
        assertEq(eqr.result[1].callData, hex"18160ddd");
        assertEq(eqr.result[1].result, hex"0000000000000000000000000000000000000000007ae5649beabeddf889364a");
    }

    function test_parseEthCallQueryResponseRevertWrongQueryType() public {
        // Take the data extracted by the previous test and break it down even further.
        ParsedPerChainQueryResponse memory r = ParsedPerChainQueryResponse({
            chainId: 5,
            queryType: 2,
            request: hex"00000009307832613631616334020d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde030d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000418160ddd",
            response: hex"0000000002a61ac4c1adff9f6e180309e7d0d94c063338ddc61c1c4474cd6957c960efe659534d040005ff312e4f90c002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d57726170706564204d6174696300000000000000000000000000000000000000000000200000000000000000000000000000000000000000007ae5649beabeddf889364a"
            });

        vm.expectRevert(UnsupportedQueryType.selector);
        queryResponse.parseEthCallQueryResponse(r);
    }

    function test_parseEthCallQueryResponseComparison() public {
        ParsedPerChainQueryResponse memory r = ParsedPerChainQueryResponse({
            chainId: 23,
            queryType: 1,
            request: hex"00000009307832376433333433013ce792601c936b1c81f73ea2fa77208c0a478bae00000004916d5743",
            response: hex"00000000027d3343b9848f128b3658a0b9b50aa174e3ddc15ac4e54c84ee534b6d247adbdfc300c90006056cda47a84001000000200000000000000000000000000000000000000000000000000000000000000004"
            });

        EthCallQueryResponse memory eqr = queryResponse.parseEthCallQueryResponse(r);
        assertEq(eqr.requestBlockId, "0x27d3343");
        assertEq(eqr.blockNum, 0x27d3343);
        assertEq(eqr.blockHash, hex"b9848f128b3658a0b9b50aa174e3ddc15ac4e54c84ee534b6d247adbdfc300c9");
        vm.warp(1694814937);
        assertEq(eqr.blockTime / 1_000_000, block.timestamp);
        assertEq(eqr.result.length, 1);

        assertEq(eqr.result[0].contractAddress, address(0x3ce792601c936b1c81f73Ea2fa77208C0A478BaE));
        assertEq(eqr.result[0].callData, hex"916d5743");
        bytes memory callData = eqr.result[0].callData;
        bytes4 callSignature;
        assembly {
                callSignature := mload(add(callData, 32))
            }
        assertEq(callSignature, bytes4(keccak256("getMyCounter()")));
        assertEq(eqr.result[0].result, hex"0000000000000000000000000000000000000000000000000000000000000004");
        assertEq(abi.decode(eqr.result[0].result, (uint256)), 4);
    }

    function test_parseEthCallByTimestampQueryResponse() public {
        // Take the data extracted by the previous test and break it down even further.
        ParsedPerChainQueryResponse memory r = ParsedPerChainQueryResponse({
            chainId: 2,
            queryType: 2,
            request: hex"00000003f4810cc0000000063078343237310000000630783432373202ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000418160ddd",
            response: hex"0000000000004271ec70d2f70cf1933770ae760050a75334ce650aa091665ee43a6ed488cd154b0800000003f4810cc000000000000042720b1608c2cddfd9d7fb4ec94f79ec1389e2410e611a2c2bbde94e9ad37519ebbb00000003f4904f0002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000"
            });

        EthCallByTimestampQueryResponse memory eqr = queryResponse.parseEthCallByTimestampQueryResponse(r);
        assertEq(eqr.requestTargetBlockIdHint, hex"307834323731");
        assertEq(eqr.requestFollowingBlockIdHint, hex"307834323732");
        assertEq(eqr.requestTargetTimestamp, 0x03f4810cc0);
        assertEq(eqr.targetBlockNum, 0x0000000000004271);
        assertEq(eqr.targetBlockHash, hex"ec70d2f70cf1933770ae760050a75334ce650aa091665ee43a6ed488cd154b08");
        assertEq(eqr.targetBlockTime, 0x03f4810cc0);
        assertEq(eqr.followingBlockNum, 0x0000000000004272);
        assertEq(eqr.followingBlockHash, hex"0b1608c2cddfd9d7fb4ec94f79ec1389e2410e611a2c2bbde94e9ad37519ebbb");
        assertEq(eqr.followingBlockTime, 0x03f4904f00);        
        assertEq(eqr.result.length, 2);

        assertEq(eqr.result[0].contractAddress, address(0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E));
        assertEq(eqr.result[0].callData, hex"06fdde03");
        assertEq(eqr.result[0].result, hex"0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000");

        assertEq(eqr.result[1].contractAddress, address(0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E));
        assertEq(eqr.result[1].callData, hex"18160ddd");
        assertEq(eqr.result[1].result, hex"0000000000000000000000000000000000000000000000000000000000000000");
    }

    function test_parseEthCallByTimestampQueryResponseRevertWrongQueryType() public {
        // Take the data extracted by the previous test and break it down even further.
        ParsedPerChainQueryResponse memory r = ParsedPerChainQueryResponse({
            chainId: 2,
            queryType: 1,
            request: hex"00000003f4810cc0000000063078343237310000000630783432373202ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000418160ddd",
            response: hex"0000000000004271ec70d2f70cf1933770ae760050a75334ce650aa091665ee43a6ed488cd154b0800000003f4810cc000000000000042720b1608c2cddfd9d7fb4ec94f79ec1389e2410e611a2c2bbde94e9ad37519ebbb00000003f4904f0002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000"
            });

        vm.expectRevert(UnsupportedQueryType.selector);
        queryResponse.parseEthCallByTimestampQueryResponse(r);
    }

    function test_parseEthCallWithFinalityQueryResponse() public {
        // Take the data extracted by the previous test and break it down even further.
        ParsedPerChainQueryResponse memory r = ParsedPerChainQueryResponse({
            chainId: 2,
            queryType: 3,
            request: hex"000000063078363032390000000966696e616c697a656402ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000418160ddd",
            response: hex"00000000000060299eb9c56ffdae81214867ed217f5ab37e295c196b4f04b23a795d3e4aea6ff3d700000005bb1bd58002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000"
            });

        EthCallWithFinalityQueryResponse memory eqr = queryResponse.parseEthCallWithFinalityQueryResponse(r);
        assertEq(eqr.requestBlockId, hex"307836303239");
        assertEq(eqr.requestFinality, hex"66696e616c697a6564");
        assertEq(eqr.blockNum, 0x6029);
        assertEq(eqr.blockHash, hex"9eb9c56ffdae81214867ed217f5ab37e295c196b4f04b23a795d3e4aea6ff3d7");
        assertEq(eqr.blockTime, 0x05bb1bd580);
        assertEq(eqr.result.length, 2);

        assertEq(eqr.result[0].contractAddress, address(0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E));
        assertEq(eqr.result[0].callData, hex"06fdde03");
        assertEq(eqr.result[0].result, hex"0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000");

        assertEq(eqr.result[1].contractAddress, address(0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E));
        assertEq(eqr.result[1].callData, hex"18160ddd");
        assertEq(eqr.result[1].result, hex"0000000000000000000000000000000000000000000000000000000000000000");
    }

    function test_parseEthCallWithFinalityQueryResponseRevertWrongQueryType() public {
        // Take the data extracted by the previous test and break it down even further.
        ParsedPerChainQueryResponse memory r = ParsedPerChainQueryResponse({
            chainId: 2,
            queryType: 1,
            request: hex"000000063078363032390000000966696e616c697a656402ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000418160ddd",
            response: hex"00000000000060299eb9c56ffdae81214867ed217f5ab37e295c196b4f04b23a795d3e4aea6ff3d700000005bb1bd58002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000"
            });

        vm.expectRevert(UnsupportedQueryType.selector);
        queryResponse.parseEthCallWithFinalityQueryResponse(r);
    }



    /***********************************
    *********** FUZZ TESTS *************
    ***********************************/

    

    function testFuzz_parseAndVerifyQueryResponse_fuzzVersion(uint8 _version) public {
        vm.assume(_version != 1);

        bytes memory resp = concatenateQueryResponseBytesOffChain(_version, senderChainId, signature, queryRequestLen, queryRequestVersion, queryRequestNonce, numPerChainQueries, perChainQueries, numPerChainResponses, perChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        vm.expectRevert(InvalidResponseVersion.selector);
        queryResponse.parseAndVerifyQueryResponse(address(wormhole), resp, signatures);
    }

    function testFuzz_parseAndVerifyQueryResponse_fuzzSenderChainId(uint16 _senderChainId) public {
        vm.assume(_senderChainId != 0);

        bytes memory resp = concatenateQueryResponseBytesOffChain(version, _senderChainId, signature, queryRequestLen, queryRequestVersion, queryRequestNonce, numPerChainQueries, perChainQueries, numPerChainResponses, perChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        // This could revert for multiple reasons. But the checkLength to ensure all the bytes are consumed is the backstop.
        vm.expectRevert();
        queryResponse.parseAndVerifyQueryResponse(address(wormhole), resp, signatures);
    }

    function testFuzz_parseAndVerifyQueryResponse_fuzzSignatureHappyCase(bytes memory _signature) public {
        // This signature isn't validated in the QueryResponse library, therefore it could be an 65 byte hex string
        vm.assume(_signature.length == 65);

        bytes memory resp = concatenateQueryResponseBytesOffChain(version, senderChainId, _signature, queryRequestLen, queryRequestVersion, queryRequestNonce, numPerChainQueries, perChainQueries, numPerChainResponses, perChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        ParsedQueryResponse memory r = queryResponse.parseAndVerifyQueryResponse(address(wormhole), resp, signatures);

        assertEq(r.requestId, _signature);
    }

    function testFuzz_parseAndVerifyQueryResponse_fuzzSignatureUnhappyCase(bytes memory _signature) public {
        // A signature that isn't 65 bytes long will always lead to a revert. The type of revert is unknown since it could be one of many.
        vm.assume(_signature.length != 65);

        bytes memory resp = concatenateQueryResponseBytesOffChain(version, senderChainId, _signature, queryRequestLen, queryRequestVersion, queryRequestNonce, numPerChainQueries, perChainQueries, numPerChainResponses, perChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        vm.expectRevert();
        queryResponse.parseAndVerifyQueryResponse(address(wormhole), resp, signatures);
    }

    function testFuzz_parseAndVerifyQueryResponse_fuzzQueryRequestLen(uint32 _queryRequestLen, bytes calldata _perChainQueries) public {
        // We add 6 to account for version + nonce + numPerChainQueries
        vm.assume(_queryRequestLen != _perChainQueries.length + 6); 

        bytes memory resp = concatenateQueryResponseBytesOffChain(version, senderChainId, signature, _queryRequestLen, queryRequestVersion, queryRequestNonce, numPerChainQueries, _perChainQueries, numPerChainResponses, perChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        vm.expectRevert();
        queryResponse.parseAndVerifyQueryResponse(address(wormhole), resp, signatures);
    }

    function testFuzz_parseAndVerifyQueryResponse_fuzzQueryRequestVersion(uint8 _version, uint8 _queryRequestVersion) public {
        vm.assume(_version != _queryRequestVersion); 

        bytes memory resp = concatenateQueryResponseBytesOffChain(_version, senderChainId, signature, queryRequestLen, _queryRequestVersion, queryRequestNonce, numPerChainQueries, perChainQueries, numPerChainResponses, perChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        vm.expectRevert();
        queryResponse.parseAndVerifyQueryResponse(address(wormhole), resp, signatures);
    }

    function testFuzz_parseAndVerifyQueryResponse_fuzzQueryRequestNonce(uint32 _queryRequestNonce) public {
        bytes memory resp = concatenateQueryResponseBytesOffChain(version, senderChainId, signature, queryRequestLen, queryRequestVersion, _queryRequestNonce, numPerChainQueries, perChainQueries, numPerChainResponses, perChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        ParsedQueryResponse memory r = queryResponse.parseAndVerifyQueryResponse(address(wormhole), resp, signatures);
        
        assertEq(r.nonce, _queryRequestNonce);
    }

    function testFuzz_parseAndVerifyQueryResponse_fuzzNumPerChainQueriesAndResponses(uint8 _numPerChainQueries, uint8 _numPerChainResponses) public {
        vm.assume(_numPerChainQueries != _numPerChainResponses);

        bytes memory resp = concatenateQueryResponseBytesOffChain(version, senderChainId, signature, queryRequestLen, queryRequestVersion, queryRequestNonce, _numPerChainQueries, perChainQueries, _numPerChainResponses, perChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        vm.expectRevert();
        queryResponse.parseAndVerifyQueryResponse(address(wormhole), resp, signatures);
    }

    function testFuzz_parseAndVerifyQueryResponse_fuzzChainIds(uint16 _requestChainId, uint16 _responseChainId, uint8 _requestQueryType) public {
        vm.assume(_requestChainId != _responseChainId);
        vm.assume(_requestQueryType >= queryResponse.QT_ETH_CALL() && _requestQueryType < queryResponse.QT_MAX());

        bytes memory packedPerChainQueries = abi.encodePacked(_requestChainId, _requestQueryType, uint32(perChainQueriesInner.length), perChainQueriesInner);
        bytes memory packedPerChainResponses = abi.encodePacked(_responseChainId, _requestQueryType, uint32(perChainResponsesInner.length),  perChainResponsesInner);
        bytes memory resp = concatenateQueryResponseBytesOffChain(version, senderChainId, signature, queryRequestLen, queryRequestVersion, queryRequestNonce, numPerChainQueries, packedPerChainQueries, numPerChainResponses, packedPerChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        vm.expectRevert(ChainIdMismatch.selector);
        queryResponse.parseAndVerifyQueryResponse(address(wormhole), resp, signatures);
    }

    function testFuzz_parseAndVerifyQueryResponse_fuzzRequestType(uint16 _requestQueryType, uint16 _responseQueryType) public {
        vm.assume(_requestQueryType != _responseQueryType);
        vm.assume(_requestQueryType < queryResponse.QT_ETH_CALL() || _requestQueryType >= queryResponse.QT_MAX());

        bytes memory packedPerChainQueries = abi.encodePacked(uint16(0x0005), _requestQueryType, uint32(perChainQueriesInner.length), perChainQueriesInner);
        bytes memory packedPerChainResponses = abi.encodePacked(uint16(0x0005), _responseQueryType, uint32(perChainResponsesInner.length),  perChainResponsesInner);
        bytes memory resp = concatenateQueryResponseBytesOffChain(version, senderChainId, signature, queryRequestLen, queryRequestVersion, queryRequestNonce, numPerChainQueries, packedPerChainQueries, numPerChainResponses, packedPerChainResponses);
        (uint8 sigV, bytes32 sigR, bytes32 sigS) = getSignature(resp);
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        vm.expectRevert();
        queryResponse.parseAndVerifyQueryResponse(address(wormhole), resp, signatures);
    }
}
