// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../contracts/interfaces/relayer/IRelayProvider.sol";
import "../../contracts/relayer/relayProvider/RelayProvider.sol";
import "../../contracts/relayer/relayProvider/RelayProviderSetup.sol";
import "../../contracts/relayer/relayProvider/RelayProviderImplementation.sol";
import "../../contracts/relayer/relayProvider/RelayProviderProxy.sol";
import "../../contracts/relayer/relayProvider/RelayProviderMessages.sol";
import "../../contracts/relayer/relayProvider/RelayProviderStructs.sol";
import "../../contracts/interfaces/relayer/TypedUnits.sol";

import "forge-std/Test.sol";

contract TestRelayProvider is Test {
    using WeiLib for Wei;
    using GasLib for Gas;
    using WeiPriceLib for WeiPrice;
    using GasPriceLib for GasPrice;

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

    function testCannotUpdatePriceWithChainIdZero(
        GasPrice updateGasPrice,
        WeiPrice updateNativeCurrencyPrice
    ) public {
        vm.assume(updateGasPrice.unwrap() > 0);
        vm.assume(updateNativeCurrencyPrice.unwrap() > 0);

        initializeRelayProvider();

        // you shall not pass
        vm.expectRevert(abi.encodeWithSignature("ChainIdIsZero()"));
        relayProvider.updatePrice(
            0, // updateChainId
            updateGasPrice,
            updateNativeCurrencyPrice
        );
    }

    function testCannotUpdatePriceWithGasPriceZero(
        uint16 updateChainId,
        WeiPrice updateNativeCurrencyPrice
    ) public {
        vm.assume(updateChainId > 0);
        vm.assume(updateNativeCurrencyPrice.unwrap() > 0);

        initializeRelayProvider();

        // you shall not pass
        vm.expectRevert(abi.encodeWithSignature("GasPriceIsZero()"));
        relayProvider.updatePrice(
            updateChainId,
            GasPrice.wrap(0), // updateGasPrice == 0
            updateNativeCurrencyPrice
        );
    }

    function testCannotUpdatePriceWithNativeCurrencyPriceZero(
        uint16 updateChainId,
        GasPrice updateGasPrice
    ) public {
        vm.assume(updateChainId > 0);
        vm.assume(updateGasPrice.unwrap() > 0);

        initializeRelayProvider();

        // you shall not pass
        vm.expectRevert(abi.encodeWithSignature("NativeCurrencyPriceIsZero()"));
        relayProvider.updatePrice(
            updateChainId,
            updateGasPrice,
            WeiPrice.wrap(0) // updateNativeCurrencyPrice == 0
        );
    }

    function testCanUpdatePriceOnlyAsOwner(
        address oracleOwner,
        uint16 updateChainId,
        GasPrice updateGasPrice,
        WeiPrice updateNativeCurrencyPrice
    ) public {
        vm.assume(oracleOwner != address(0));
        vm.assume(oracleOwner != address(this));
        vm.assume(updateChainId > 0);
        vm.assume(updateGasPrice.unwrap() > 0);
        vm.assume(updateNativeCurrencyPrice.unwrap() > 0);

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
        uint64 dstGasPrice,
        uint64 dstNativeCurrencyPrice,
        uint64 srcGasPrice,
        uint64 srcNativeCurrencyPrice
    ) public {
        vm.assume(dstChainId > 0);
        vm.assume(dstChainId != TEST_ORACLE_CHAIN_ID);
        vm.assume(dstGasPrice > 0);
        vm.assume(dstNativeCurrencyPrice > 0);
        vm.assume(srcGasPrice > 0);
        vm.assume(srcNativeCurrencyPrice > 0);
        vm.assume(uint256(dstGasPrice) * srcNativeCurrencyPrice >= dstNativeCurrencyPrice);
        vm.assume(dstGasPrice * uint256(dstNativeCurrencyPrice) / srcNativeCurrencyPrice < 2 ** 72);

        initializeRelayProvider();

        // update the prices with reasonable values
        relayProvider.updatePrice(dstChainId, GasPrice.wrap(dstGasPrice), WeiPrice.wrap(dstNativeCurrencyPrice));
        relayProvider.updatePrice(TEST_ORACLE_CHAIN_ID, GasPrice.wrap(srcGasPrice), WeiPrice.wrap(srcNativeCurrencyPrice));

        // verify price
        uint256 expected = (
            uint256(dstNativeCurrencyPrice) * (uint256(dstGasPrice)) + (srcNativeCurrencyPrice - 1)
        ) / srcNativeCurrencyPrice;
        GasPrice readValues = relayProvider.quoteGasPrice(dstChainId);
        console.log(readValues.unwrap(), expected);
        require(readValues.unwrap() == expected, "relayProvider.quotePrices != expected");
    }

    struct UpdatePrice {
        uint16 chainId;
        uint128 gasPrice;
        uint128 nativeCurrencyPrice;
    }

    function testBulkUpdatePrices(
        uint16 dstChainId,
        uint64 dstGasPrice,
        uint64 dstNativeCurrencyPrice,
        uint64 srcGasPrice,
        uint64 srcNativeCurrencyPrice
    ) public {
        vm.assume(dstChainId > 0);
        vm.assume(dstChainId != TEST_ORACLE_CHAIN_ID); // wormhole.chainId()
        vm.assume(dstGasPrice > 0);
        vm.assume(dstNativeCurrencyPrice > 0);
        vm.assume(srcGasPrice > 0);
        vm.assume(srcNativeCurrencyPrice > 0);
        vm.assume(dstGasPrice >= dstNativeCurrencyPrice / srcNativeCurrencyPrice);
        vm.assume(dstGasPrice * uint256(dstNativeCurrencyPrice) / srcNativeCurrencyPrice < 2 ** 72);

        initializeRelayProvider();

        RelayProviderStructs.UpdatePrice[] memory updates =
            new RelayProviderStructs.UpdatePrice[](2);
        updates[0] = RelayProviderStructs.UpdatePrice({
            chainId: TEST_ORACLE_CHAIN_ID,
            gasPrice: GasPrice.wrap(srcGasPrice),
            nativeCurrencyPrice: WeiPrice.wrap(srcNativeCurrencyPrice)
        });
        updates[1] = RelayProviderStructs.UpdatePrice({
            chainId: dstChainId,
            gasPrice: GasPrice.wrap(dstGasPrice),
            nativeCurrencyPrice: WeiPrice.wrap(dstNativeCurrencyPrice)
        });

        // update the prices with reasonable values
        relayProvider.updatePrices(updates);

        // verify price
        uint256 expected = (
            uint256(dstNativeCurrencyPrice) * (uint256(dstGasPrice)) + (srcNativeCurrencyPrice - 1)
        ) / srcNativeCurrencyPrice;
        GasPrice readValues = relayProvider.quoteGasPrice(dstChainId);
        require(readValues.unwrap() == expected, "relayProvider.quotePrices != expected");
    }

    function testUpdateTargetChainContracts(uint16 targetChain, bytes32 newAddress) public {
        initializeRelayProvider();

        relayProvider.updateTargetChainAddress(targetChain, newAddress);
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
