// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../contracts/interfaces/IRelayProvider.sol";
import "../contracts/relayProvider/RelayProvider.sol";
import "../contracts/relayProvider/RelayProviderSetup.sol";
import "../contracts/relayProvider/RelayProviderImplementation.sol";
import "../contracts/relayProvider/RelayProviderProxy.sol";
import "../contracts/relayProvider/RelayProviderMessages.sol";
import "../contracts/relayProvider/RelayProviderStructs.sol";

import "forge-std/Test.sol";

contract TestRelayProvider is Test {
    uint16 constant TEST_ORACLE_CHAIN_ID = 2;

    RelayProvider internal relayProvider;

    function initializeRelayProvider() internal {
        RelayProviderSetup relayProviderSetup = new RelayProviderSetup();
        RelayProviderImplementation relayProviderImplementation = new RelayProviderImplementation();
        RelayProviderProxy myRelayProvider = new RelayProviderProxy(
            address(relayProviderSetup),
            abi.encodeCall(
                RelayProviderSetup.setup,
                (
                    address(relayProviderImplementation),
                    TEST_ORACLE_CHAIN_ID
                )
            )
        );

        relayProvider = RelayProvider(address(myRelayProvider));

        require(relayProvider.owner() == address(this), "owner() != expected");
        require(relayProvider.chainId() == TEST_ORACLE_CHAIN_ID, "chainId() != expected");
    }

    function testCannotUpdatePriceWithChainIdZero(uint128 updateGasPrice, uint128 updateNativeCurrencyPrice) public {
        vm.assume(updateGasPrice > 0);
        vm.assume(updateNativeCurrencyPrice > 0);

        initializeRelayProvider();

        // you shall not pass
        vm.expectRevert(abi.encodeWithSignature("ChainIdIsZero()"));
        relayProvider.updatePrice(
            0, // updateChainId
            updateGasPrice,
            updateNativeCurrencyPrice
        );
    }

    function testCannotUpdatePriceWithGasPriceZero(uint16 updateChainId, uint128 updateNativeCurrencyPrice) public {
        vm.assume(updateChainId > 0);
        vm.assume(updateNativeCurrencyPrice > 0);

        initializeRelayProvider();

        // you shall not pass
        vm.expectRevert(abi.encodeWithSignature("GasPriceIsZero()"));
        relayProvider.updatePrice(
            updateChainId,
            0, // updateGasPrice == 0
            updateNativeCurrencyPrice
        );
    }

    function testCannotUpdatePriceWithNativeCurrencyPriceZero(uint16 updateChainId, uint128 updateGasPrice) public {
        vm.assume(updateChainId > 0);
        vm.assume(updateGasPrice > 0);

        initializeRelayProvider();

        // you shall not pass
        vm.expectRevert(abi.encodeWithSignature("NativeCurrencyPriceIsZero()"));
        relayProvider.updatePrice(
            updateChainId,
            updateGasPrice,
            0 // updateNativeCurrencyPrice == 0
        );
    }

    function testCanUpdatePriceOnlyAsOwner(
        address oracleOwner,
        uint16 updateChainId,
        uint128 updateGasPrice,
        uint128 updateNativeCurrencyPrice
    ) public {
        vm.assume(oracleOwner != address(0));
        vm.assume(oracleOwner != address(this));
        vm.assume(updateChainId > 0);
        vm.assume(updateGasPrice > 0);
        vm.assume(updateNativeCurrencyPrice > 0);

        initializeRelayProvider();

        // you shall not pass
        vm.prank(oracleOwner);
        vm.expectRevert(abi.encodeWithSignature("CallerMustBeOwner()"));
        relayProvider.updatePrice(updateChainId, updateGasPrice, updateNativeCurrencyPrice);
    }

    /*
    TODO: Uncomment these tests once revert messages are back in
    function testCannotGetPriceBeforeUpdateSrcPrice(
        uint16 dstChainId,
        uint128 dstGasPrice,
        uint128 dstNativeCurrencyPrice
    )
        public
    {
        vm.assume(dstChainId > 0);
        vm.assume(dstChainId != TEST_ORACLE_CHAIN_ID);
        vm.assume(dstGasPrice > 0);
        vm.assume(dstNativeCurrencyPrice > 0);

        initializeRelayProvider();

        // update the price with reasonable values
        relayProvider.updatePrice(dstChainId, dstGasPrice, dstNativeCurrencyPrice);

        // you shall not pass
        vm.expectRevert("srcNativeCurrencyPrice == 0");
        relayProvider.quoteDeliveryOverhead(dstChainId);
    }

    function testCannotGetPriceBeforeUpdateDstPrice(
        uint16 dstChainId,
        uint128 srcGasPrice,
        uint128 srcNativeCurrencyPrice
    )
        public
    {
        vm.assume(dstChainId > 0);
        vm.assume(dstChainId != TEST_ORACLE_CHAIN_ID);
        vm.assume(srcGasPrice > 0);
        vm.assume(srcNativeCurrencyPrice > 0);

        initializeRelayProvider();

        // update the price with reasonable values
        //vm.prank(relayProvider.owner());
        relayProvider.updatePrice(TEST_ORACLE_CHAIN_ID, srcGasPrice, srcNativeCurrencyPrice);

        // you shall not pass
        vm.expectRevert("dstNativeCurrencyPrice == 0");
        relayProvider.quoteDeliveryOverhead(dstChainId);
    }
    */

    function testUpdatePrice(
        uint16 dstChainId,
        uint128 dstGasPrice,
        uint64 dstNativeCurrencyPrice,
        uint128 srcGasPrice,
        uint64 srcNativeCurrencyPrice,
        uint32 gasLimit,
        uint32 deliverGasOverhead,
        uint32 targetWormholeFee
    ) public {
        vm.assume(dstChainId > 0);
        vm.assume(dstChainId != TEST_ORACLE_CHAIN_ID);
        vm.assume(dstGasPrice > 0);
        vm.assume(dstNativeCurrencyPrice > 0);
        vm.assume(srcGasPrice > 0);
        vm.assume(srcNativeCurrencyPrice > 0);
        vm.assume(uint256(dstGasPrice) * srcNativeCurrencyPrice >= dstNativeCurrencyPrice);

        initializeRelayProvider();

        // update the prices with reasonable values
        relayProvider.updatePrice(dstChainId, dstGasPrice, dstNativeCurrencyPrice);
        relayProvider.updatePrice(TEST_ORACLE_CHAIN_ID, srcGasPrice, srcNativeCurrencyPrice);

        // verify price
        uint256 expected = (uint256(dstNativeCurrencyPrice) * (uint256(dstGasPrice)) + (srcNativeCurrencyPrice - 1))
            / srcNativeCurrencyPrice;
        uint256 readValues = relayProvider.quoteGasPrice(dstChainId);
        require(readValues == expected, "relayProvider.quotePrices != expected");
    }

    struct UpdatePrice {
        uint16 chainId;
        uint128 gasPrice;
        uint128 nativeCurrencyPrice;
    }

    function testUpdatePrices(
        uint16 dstChainId,
        uint128 dstGasPrice,
        uint64 dstNativeCurrencyPrice,
        uint128 srcGasPrice,
        uint64 srcNativeCurrencyPrice,
        uint32 gasLimit,
        uint32 deliverGasOverhead,
        uint32 targetWormholeFee
    ) public {
        vm.assume(dstChainId > 0);
        vm.assume(dstChainId != TEST_ORACLE_CHAIN_ID); // wormhole.chainId()
        vm.assume(dstGasPrice > 0);
        vm.assume(dstNativeCurrencyPrice > 0);
        vm.assume(srcGasPrice > 0);
        vm.assume(srcNativeCurrencyPrice > 0);
        vm.assume(dstGasPrice >= dstNativeCurrencyPrice / srcNativeCurrencyPrice);

        initializeRelayProvider();

        RelayProviderStructs.UpdatePrice[] memory updates = new RelayProviderStructs.UpdatePrice[](2);
        updates[0] = RelayProviderStructs.UpdatePrice({
            chainId: TEST_ORACLE_CHAIN_ID,
            gasPrice: srcGasPrice,
            nativeCurrencyPrice: srcNativeCurrencyPrice
        });
        updates[1] = RelayProviderStructs.UpdatePrice({
            chainId: dstChainId,
            gasPrice: dstGasPrice,
            nativeCurrencyPrice: dstNativeCurrencyPrice
        });

        // update the prices with reasonable values
        relayProvider.updatePrices(updates);

        // verify price
        uint256 expected = (uint256(dstNativeCurrencyPrice) * (uint256(dstGasPrice)) + (srcNativeCurrencyPrice - 1))
            / srcNativeCurrencyPrice;
        uint256 readValues = relayProvider.quoteGasPrice(dstChainId);
        require(readValues == expected, "relayProvider.quotePrices != expected");
    }

    function testUpdateTargetChainContracts(bytes32 newAddress, uint16 targetChain) public {
        initializeRelayProvider();

        relayProvider.updateTargetChainAddress(newAddress, targetChain);
        bytes32 updated = relayProvider.getTargetChainAddress(targetChain);

        assertTrue(newAddress == updated);
    }

    function testUpdateRewardAddress(address payable newAddress) public {
        initializeRelayProvider();

        relayProvider.updateRewardAddress(newAddress);
        address payable updated = relayProvider.getRewardAddress();

        assertTrue(newAddress == updated);
    }

}
