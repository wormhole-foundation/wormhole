// SPDX-License-Identifier: Apache 2

// forge test --match-contract QueryTest

pragma solidity ^0.8.4;

import "forge-std/Test.sol";
import "./QueryTest.sol";

contract TestQueryTest is Test {
    //
    // Query Request tests
    //

    function test_buildOffChainQueryRequestBytes() public {
        bytes memory req = QueryTest.buildOffChainQueryRequestBytes(
            /* version */            1,
            /* nonce */              1,
            /* numPerChainQueries */ 1,
            /* perChainQueries */    hex"0002010000004200000005307837343402ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567"
        );
        assertEq(req, hex"0100000001010002010000004200000005307837343402ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567");
    }

    function test_buildPerChainRequestBytes() public {
        bytes memory pcr = QueryTest.buildPerChainRequestBytes(
            /* chainId */    2,
            /* queryType */  1,
            /* queryBytes */ hex"00000005307837343402ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567"
        );
        assertEq(pcr, hex"0002010000004200000005307837343402ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567");
    }

    function test_buildEthCallRequestBytes() public {
        bytes memory ecr = QueryTest.buildEthCallRequestBytes(
            /* blockId */     "0x744",
            /* numCallData */ 2,
            /* callData */    hex"ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567"
        );
        assertEq(ecr, hex"00000005307837343402ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567");
    }

    function test_buildEthCallByTimestampRequestBytes() public {
        bytes memory ecr = QueryTest.buildEthCallByTimestampRequestBytes(
            /* targetTimeUs */       0x10642ac0,
            /* targetBlockHint */    "0x15d",
            /* followingBlockHint */ "0x15e",            
            /* numCallData */        2,
            /* callData */           hex"ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567"
        );
        assertEq(ecr, hex"0000000010642ac000000005307831356400000005307831356502ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567");
    }
    
    function test_buildEthCallWithFinalityRequestBytes() public {
        bytes memory ecr = QueryTest.buildEthCallWithFinalityRequestBytes(
            /* blockId */     "0x1f8",
            /* finality */    "finalized",            
            /* numCallData */ 2,
            /* callData */    hex"ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567"
        );
        assertEq(ecr, hex"0000000530783166380000000966696e616c697a656402ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567");
    }

    function test_buildEthCallDataBytes() public {
        bytes memory ecd1 = QueryTest.buildEthCallDataBytes(
            /* contractAddress */ 0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E,
            /* callData */        hex"06fdde03"
        );
        assertEq(ecd1, hex"ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03");
        bytes memory ecd2 = QueryTest.buildEthCallDataBytes(
            /* contractAddress */ 0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E,
            /* callData */        hex"313ce567"
        );
        assertEq(ecd2, hex"ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567");
    }        
        
    function test_buildSolanaAccountRequestBytes() public {
        bytes memory ecr = QueryTest.buildSolanaAccountRequestBytes(
            /* commitment */      "finalized",
            /* minContextSlot */  8069,
            /* dataSliceOffset */ 10,
            /* dataSliceLength */ 20,
            /* numAccounts */     2,
            /* accounts */        hex"165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa3019c006c48c8cbf33849cb07a3f936159cc523f9591cb1999abd45890ec5fee9b7"
        );
        assertEq(ecr, hex"0000000966696e616c697a65640000000000001f85000000000000000a000000000000001402165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa3019c006c48c8cbf33849cb07a3f936159cc523f9591cb1999abd45890ec5fee9b7");
    }

    function test_buildSolanaPdaRequestBytes() public {
        bytes32 programId = hex"02c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa";
        bytes[] memory pdas = new bytes[](2);

        bytes[] memory seeds = new bytes[](2);
        seeds[0] = hex"477561726469616e536574";
        seeds[1] = hex"00000000";
        (bytes memory seedBytes, uint8 numSeeds) = QueryTest.buildSolanaPdaSeedBytes(seeds);
        assertEq(seedBytes, hex"0000000b477561726469616e5365740000000400000000");

        pdas[0] = QueryTest.buildSolanaPdaEntry(
            programId,
            numSeeds,
            seedBytes
        );
        assertEq(pdas[0], hex"02c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa020000000b477561726469616e5365740000000400000000");
        assertEq(numSeeds, uint8(seeds.length));

        bytes[] memory seeds2 = new bytes[](2);
        seeds2[0] = hex"477561726469616e536574";
        seeds2[1] = hex"00000001";
        (bytes memory seedBytes2, uint8 numSeeds2) = QueryTest.buildSolanaPdaSeedBytes(seeds2);
        assertEq(seedBytes2, hex"0000000b477561726469616e5365740000000400000001");

        pdas[1] = QueryTest.buildSolanaPdaEntry(
            programId,
            numSeeds2,
            seedBytes2
        );
        assertEq(pdas[1], hex"02c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa020000000b477561726469616e5365740000000400000001");
        assertEq(numSeeds2, uint8(seeds2.length));

        bytes memory ecr = QueryTest.buildSolanaPdaRequestBytes(
            /* commitment */      "finalized",
            /* minContextSlot */  2303,
            /* dataSliceOffset */ 12,
            /* dataSliceLength */ 20,
            /* pdas */            pdas
        );
        assertEq(ecr, hex"0000000966696e616c697a656400000000000008ff000000000000000c00000000000000140202c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa020000000b477561726469616e536574000000040000000002c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa020000000b477561726469616e5365740000000400000001");
    }

    function test_buildSolanaPdaRequestBytesTooManyPDAs() public {
        bytes32 programId = hex"02c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa";
        bytes[] memory pdas = new bytes[](256);

        uint numPDAs = pdas.length;
        for (uint idx; idx < numPDAs;) {
            bytes[] memory seeds = new bytes[](2);
            seeds[0] = hex"477561726469616e536574";
            seeds[1] = hex"00000000";
            (bytes memory seedBytes, uint8 numSeeds) = QueryTest.buildSolanaPdaSeedBytes(seeds);

            pdas[idx] = QueryTest.buildSolanaPdaEntry(
                programId,
                numSeeds,
                seedBytes
            );

            unchecked { ++idx; }
        }

        vm.expectRevert(QueryTest.SolanaTooManyPDAs.selector);
        QueryTest.buildSolanaPdaRequestBytes(
            /* commitment */      "finalized",
            /* minContextSlot */  2303,
            /* dataSliceOffset */ 12,
            /* dataSliceLength */ 20,
            /* pdas */            pdas
        );
    }

    function test_buildSolanaPdaEntryTooManySeeds() public {
        bytes[] memory seeds = new bytes[](2);
        seeds[0] = hex"477561726469616e536574";
        seeds[1] = hex"00000000";
        (bytes memory seedBytes,) = QueryTest.buildSolanaPdaSeedBytes(seeds);
        assertEq(seedBytes, hex"0000000b477561726469616e5365740000000400000000");

        bytes32 programId = hex"02c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa";

        vm.expectRevert(QueryTest.SolanaTooManySeeds.selector);
        QueryTest.buildSolanaPdaEntry(
            programId,
            uint8(QueryTest.SolanaMaxSeeds + 1),
            seedBytes
        );
    }

    function test_buildSolanaPdaSeedBytesTooManySeeds() public {
        bytes[] memory seeds = new bytes[](QueryTest.SolanaMaxSeeds + 1);
        uint numSeeds = seeds.length;
        for (uint idx; idx < numSeeds;) {
            seeds[idx] = "junk";
            unchecked { ++idx; }
        }

        vm.expectRevert(QueryTest.SolanaTooManySeeds.selector);
        QueryTest.buildSolanaPdaSeedBytes(seeds);
    }

    function test_buildSolanaPdaSeedBytesSeedTooLong() public {
        bytes[] memory seeds = new bytes[](2);
        seeds[0] = "junk";
        seeds[1] = "This seed is too long!!!!!!!!!!!!";

        vm.expectRevert(QueryTest.SolanaSeedTooLong.selector);
        QueryTest.buildSolanaPdaSeedBytes(seeds);
    }

    //
    // Query Response tests
    //

    function test_buildQueryResponseBytes() public {
        bytes memory resp = QueryTest.buildQueryResponseBytes(
            /* version */              1,
            /* senderChainId */        0,
            /* signature */            hex"11b03bdbbe15a8f12b803d2193de5ddff72d92eaabd2763553ec3c3133182d1443719a05e2b65c87b923c6bd8aeff49f34937f90f3ab7cd33449388c60fa30a301",
            /* queryRequest */         hex"0100000001010002010000004200000005307837343402ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce567",
            /* numPerChainResponses */ 1,
            /* perChainResponses */    hex"000201000000b900000000000007446a0b819aee8945e659e37537a0bdbe03c06275be23e499819138d1eee8337e9b000000006ab13b8002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000012"
        );
        assertEq(resp, hex"01000011b03bdbbe15a8f12b803d2193de5ddff72d92eaabd2763553ec3c3133182d1443719a05e2b65c87b923c6bd8aeff49f34937f90f3ab7cd33449388c60fa30a3010000004f0100000001010002010000004200000005307837343402ddb64fe46a91d46ee29420539fc25fd07c5fea3e0000000406fdde03ddb64fe46a91d46ee29420539fc25fd07c5fea3e00000004313ce56701000201000000b900000000000007446a0b819aee8945e659e37537a0bdbe03c06275be23e499819138d1eee8337e9b000000006ab13b8002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000012");
    }

    function test_buildPerChainResponseBytes() public {
        bytes memory pcr = QueryTest.buildPerChainResponseBytes(
            /* chainId */       2,
            /* queryType */     1,
            /* responseBytes */ hex"00000000000007446a0b819aee8945e659e37537a0bdbe03c06275be23e499819138d1eee8337e9b000000006ab13b8002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000012"
        );
        assertEq(pcr, hex"000201000000b900000000000007446a0b819aee8945e659e37537a0bdbe03c06275be23e499819138d1eee8337e9b000000006ab13b8002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000012");
    }

    function test_buildEthCallResponseBytes() public {
        bytes memory ecr = QueryTest.buildEthCallResponseBytes(
            /* blockNumber */ 1860,
            /* blockHash */   hex"6a0b819aee8945e659e37537a0bdbe03c06275be23e499819138d1eee8337e9b",
            /* blockTimeUs */ 0x6ab13b80,
            /* numResults */  2,
            /* results */     hex"000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000012"
        );
        assertEq(ecr, hex"00000000000007446a0b819aee8945e659e37537a0bdbe03c06275be23e499819138d1eee8337e9b000000006ab13b8002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000012");
    }

    function test_buildEthCallByTimestampResponseBytes() public {
        bytes memory ecr = QueryTest.buildEthCallByTimestampResponseBytes(
            /* targetBlockNumber */    349,
            /* targetBlockHash */      hex"966cd846f812be43c4ee2d310f962bc592ba944c66de878e53584b8e75c6051f",
            /* targetBlockTimeUs */    0x10642ac0,
            /* followingBlockNumber */ 350,
            /* followingBlockHash */   hex"04b022afaab8da2dd80bd8e6ae55e6303473a5e1de846a5de76d619e162429ce",
            /* followingBlockTimeUs */ 0x10736d00,            
            /* numResults */           2,
            /* results */              hex"000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000012"
        );
        assertEq(ecr, hex"000000000000015d966cd846f812be43c4ee2d310f962bc592ba944c66de878e53584b8e75c6051f0000000010642ac0000000000000015e04b022afaab8da2dd80bd8e6ae55e6303473a5e1de846a5de76d619e162429ce0000000010736d0002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000012");
    }

    function test_buildEthCallWithFinalityResponseBytes() public {
        bytes memory ecr = QueryTest.buildEthCallWithFinalityResponseBytes(
            /* blockNumber */ 1860,
            /* blockHash */   hex"6a0b819aee8945e659e37537a0bdbe03c06275be23e499819138d1eee8337e9b",
            /* blockTimeUs */ 0x6ab13b80,
            /* numResults */  2,
            /* results */     hex"000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000012"
        );
        assertEq(ecr, hex"00000000000007446a0b819aee8945e659e37537a0bdbe03c06275be23e499819138d1eee8337e9b000000006ab13b8002000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000012");
    }

    function test_buildEthCallResultBytes() public {
        bytes memory ecr1 = QueryTest.buildEthCallResultBytes(
            /* result */    hex"0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000"
        );
        assertEq(ecr1, hex"000000600000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000000d5772617070656420457468657200000000000000000000000000000000000000");
        bytes memory ecr2 = QueryTest.buildEthCallResultBytes(
            /* result */    hex"0000000000000000000000000000000000000000000000000000000000000012"
        );
        assertEq(ecr2, hex"000000200000000000000000000000000000000000000000000000000000000000000012");
    }

    function test_buildSolanaAccountResponseBytes() public {
        bytes memory ecr = QueryTest.buildSolanaAccountResponseBytes(
            /* slotNumber */  5603,
            /* blockTimeUs */ 0x610cdf2510500,
            /* blockHash */   hex"e0eca895a92c0347e30538cd07c50777440de58e896dd13ff86ef0dae3e12552",
            /* numResults */  2,
            /* results */     hex"0000000000164d6000000000000000000006ddf6e1d765a193d9cbe146ceeb79ac1cb485ed5f5b37913a8cf5857eff00a90000005201000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d0000e8890423c78a09010000000000000000000000000000000000000000000000000000000000000000000000000000000000164d6000000000000000000006ddf6e1d765a193d9cbe146ceeb79ac1cb485ed5f5b37913a8cf5857eff00a90000005201000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d01000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000"
        );
        assertEq(ecr, hex"00000000000015e3000610cdf2510500e0eca895a92c0347e30538cd07c50777440de58e896dd13ff86ef0dae3e12552020000000000164d6000000000000000000006ddf6e1d765a193d9cbe146ceeb79ac1cb485ed5f5b37913a8cf5857eff00a90000005201000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d0000e8890423c78a09010000000000000000000000000000000000000000000000000000000000000000000000000000000000164d6000000000000000000006ddf6e1d765a193d9cbe146ceeb79ac1cb485ed5f5b37913a8cf5857eff00a90000005201000000574108aed69daf7e625a361864b1f74d13702f2ca56de9660e566d1d8691848d01000000000000000001000000000000000000000000000000000000000000000000000000000000000000000000");
    }

    function test_buildSolanaPdaResponseBytes() public {
        bytes memory ecr = QueryTest.buildSolanaPdaResponseBytes(
            /* slotNumber */  2303,
            /* blockTimeUs */ 0x6115e3f6d7540,
            /* blockHash */   hex"e05035785e15056a8559815e71343ce31db2abf23f65b19c982b68aee7bf207b",
            /* numResults */  1,
            /* results */     hex"4fa9188b339cfd573a0778c5deaeeee94d4bcfb12b345bf8e417e5119dae773efd0000000000116ac000000000000000000002c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa0000001457cd18b7f8a4d91a2da9ab4af05d0fbece2dcd65"
        );
        assertEq(ecr, hex"00000000000008ff0006115e3f6d7540e05035785e15056a8559815e71343ce31db2abf23f65b19c982b68aee7bf207b014fa9188b339cfd573a0778c5deaeeee94d4bcfb12b345bf8e417e5119dae773efd0000000000116ac000000000000000000002c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa0000001457cd18b7f8a4d91a2da9ab4af05d0fbece2dcd65");
    }
}
