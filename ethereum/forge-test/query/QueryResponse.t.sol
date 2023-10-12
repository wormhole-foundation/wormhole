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
    bytes resp = hex"010000ff0c222dc9e3655ec38e212e9792bf1860356d1277462b6bf747db865caca6fc08e6317b64ee3245264e371146b1d315d38c867fe1f69614368dc4430bb560f2000000005301dd9914c6010005010000004600000009307832613631616334020d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000406fdde030d500b1d8e8ef31e21c99d1db9a6444d3adf12700000000418160ddd01000501000000b90000000002a61ac4c1adff9f6e180309e7d0d94c063338ddc61c1c4474cd6957c960efe659534d040005ff312e4f90c002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d57726170706564204d6174696300000000000000000000000000000000000000000000200000000000000000000000000000000000000000007ae5649beabeddf889364a";

    bytes32 sigR = hex"ba36cd576a0f9a8a37ec5ea6a174857922f2f170cd7ec62edcbe74b1cc7258d3";
    bytes32 sigS = hex"01e8690cfd627e608d63b5d165e2190ba081bb84f5cf473fd353109e152f72fa";
    uint8 sigV = 27; // last byte plus magic 27
    uint8 sigGuardianIndex = 0;

    bytes32 expectedHash = 0xed18e80906ffa80ce953a132a9cbbcf84186955f8fc8ce0322cd68622a58570e;
    bytes32 expectedDigest = 0x5b84b19c68ee0b37899230175a92ee6eda4c5192e8bffca1d057d811bb3660e2;

    Wormhole wormhole;
    QueryResponse queryResponse;

    function setUp() public {
        wormhole = deployWormholeForTest();
        queryResponse = new QueryResponseContract();
    }

    uint16 constant TEST_CHAIN_ID = 2;
    address constant DEVNET_GUARDIAN = 0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe;
    uint16 constant GOVERNANCE_CHAIN_ID = 1;
    bytes32 constant GOVERNANCE_CONTRACT = 0x0000000000000000000000000000000000000000000000000000000000000004;

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
        bytes32 hash = queryResponse.getResponseHash(resp);
        assertEq(hash, expectedHash);
    }

    function test_getResponseDigest() public {
        bytes32 digest = queryResponse.getResponseDigest(resp);
        assertEq(digest, expectedDigest);
    }

    function test_verifyQueryResponseSignatures() public view {
        IWormhole.Signature[] memory signatures = new IWormhole.Signature[](1);
        signatures[0] = IWormhole.Signature({r: sigR, s: sigS, v: sigV, guardianIndex: sigGuardianIndex});
        queryResponse.verifyQueryResponseSignatures(address(wormhole), resp, signatures);
        // TODO: There are no assertions for this test
    }

    function test_parseAndVerifyQueryResponse() public {
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
}
