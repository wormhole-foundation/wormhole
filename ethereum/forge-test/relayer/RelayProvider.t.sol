// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../contracts/interfaces/relayer/IDeliveryProviderTyped.sol";
import "../../contracts/relayer/deliveryProvider/DeliveryProvider.sol";
import "../../contracts/relayer/deliveryProvider/DeliveryProviderSetup.sol";
import "../../contracts/relayer/deliveryProvider/DeliveryProviderImplementation.sol";
import "../../contracts/relayer/deliveryProvider/DeliveryProviderProxy.sol";
import "../../contracts/relayer/deliveryProvider/DeliveryProviderStructs.sol";
import "../../contracts/interfaces/relayer/TypedUnits.sol";
import {MockWormhole} from "./MockWormhole.sol";
import {IWormhole} from "../../contracts/interfaces/IWormhole.sol";
import {WormholeSimulator, FakeWormholeSimulator} from "./WormholeSimulator.sol";

import "forge-std/Test.sol";

contract TestDeliveryProvider is Test {
    using WeiLib for Wei;
    using GasLib for Gas;
    using WeiPriceLib for WeiPrice;
    using GasPriceLib for GasPrice;

    uint16 constant TEST_ORACLE_CHAIN_ID = 2;

    DeliveryProvider internal deliveryProvider;

    function initializeDeliveryProvider() internal {
        DeliveryProviderSetup deliveryProviderSetup = new DeliveryProviderSetup();
        DeliveryProviderImplementation deliveryProviderImplementation =
            new DeliveryProviderImplementation();
        DeliveryProviderProxy myDeliveryProvider = new DeliveryProviderProxy(
            address(deliveryProviderSetup),
            abi.encodeCall(
                DeliveryProviderSetup.setup,
                (
                    address(deliveryProviderImplementation),
                    TEST_ORACLE_CHAIN_ID
                )
            )
        );

        deliveryProvider = DeliveryProvider(address(myDeliveryProvider));

        require(deliveryProvider.owner() == address(this), "owner() != expected");
        require(deliveryProvider.chainId() == TEST_ORACLE_CHAIN_ID, "chainId() != expected");
    }

    function testCannotUpdatePriceWithChainIdZero(
        GasPrice updateGasPrice,
        WeiPrice updateNativeCurrencyPrice
    ) public {
        vm.assume(updateGasPrice.unwrap() > 0);
        vm.assume(updateNativeCurrencyPrice.unwrap() > 0);

        initializeDeliveryProvider();

        // you shall not pass
        vm.expectRevert(abi.encodeWithSignature("ChainIdIsZero()"));
        deliveryProvider.updatePrice(
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

        initializeDeliveryProvider();

        // you shall not pass
        vm.expectRevert(abi.encodeWithSignature("GasPriceIsZero()"));
        deliveryProvider.updatePrice(
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

        initializeDeliveryProvider();

        // you shall not pass
        vm.expectRevert(abi.encodeWithSignature("NativeCurrencyPriceIsZero()"));
        deliveryProvider.updatePrice(
            updateChainId,
            updateGasPrice,
            WeiPrice.wrap(0) // updateNativeCurrencyPrice == 0
        );
    }

    function testCanUpdatePriceOnlyAsOwnerOrPriceWallet(
        address pricingWallet,
        address maliciousUser,
        uint16 updateChainId,
        GasPrice updateGasPrice,
        WeiPrice updateNativeCurrencyPrice
    ) public {
        vm.assume(maliciousUser != address(0));
        vm.assume(pricingWallet != address(0));
        vm.assume(maliciousUser != pricingWallet);
        vm.assume(maliciousUser != address(this));
        vm.assume(updateChainId > 0);
        vm.assume(updateGasPrice.unwrap() > 0);
        vm.assume(updateNativeCurrencyPrice.unwrap() > 0);
        vm.assume(updateGasPrice.unwrap() < type(uint64).max);
        vm.assume(updateNativeCurrencyPrice.unwrap() < type(uint128).max);

        initializeDeliveryProvider();
        deliveryProvider.updatePricingWallet(pricingWallet);

        // you shall not pass
        vm.prank(maliciousUser);
        vm.expectRevert(abi.encodeWithSignature("CallerMustBeOwnerOrPricingWallet()"));
        deliveryProvider.updatePrice(updateChainId, updateGasPrice, updateNativeCurrencyPrice);

        // pricing wallet
        vm.prank(pricingWallet);
        deliveryProvider.updatePrice(updateChainId, updateGasPrice, updateNativeCurrencyPrice);

        // owner
        deliveryProvider.updatePrice(updateChainId, updateGasPrice, updateNativeCurrencyPrice);
    }


    function testCannotGetPriceBeforeUpdateSrcPrice(
        uint16 dstChainId,
        uint64 dstGasPrice,
        WeiPrice dstNativeCurrencyPrice
    )
        public
    {
        vm.assume(dstChainId > 0);
        vm.assume(dstChainId != TEST_ORACLE_CHAIN_ID);
        vm.assume(dstGasPrice > 0);
        vm.assume(dstNativeCurrencyPrice.unwrap() > 0);

        initializeDeliveryProvider();

        // update the price with reasonable values
        deliveryProvider.updatePrice(dstChainId, GasPrice.wrap(dstGasPrice), dstNativeCurrencyPrice);

        // you shall not pass
        vm.expectRevert(abi.encodeWithSignature("PriceIsZero(uint16)", TEST_ORACLE_CHAIN_ID));
        deliveryProvider.quoteDeliveryOverhead(dstChainId);
    }

    function testCannotGetPriceBeforeUpdateDstPrice(
        uint16 dstChainId,
        uint64 srcGasPrice,
        WeiPrice srcNativeCurrencyPrice
    )
        public
    {
        vm.assume(dstChainId > 0);
        vm.assume(dstChainId != TEST_ORACLE_CHAIN_ID);
        vm.assume(srcGasPrice > 0);
        vm.assume(srcNativeCurrencyPrice.unwrap() > 0);

        initializeDeliveryProvider();

        // update the price with reasonable values
        //vm.prank(deliveryProvider.owner());
        deliveryProvider.updatePrice(TEST_ORACLE_CHAIN_ID, GasPrice.wrap(srcGasPrice), srcNativeCurrencyPrice);

        // you shall not pass
        vm.expectRevert(abi.encodeWithSignature("PriceIsZero(uint16)", dstChainId));
        deliveryProvider.quoteDeliveryOverhead(dstChainId);
    }
    

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

        initializeDeliveryProvider();

        // update the prices with reasonable values
        deliveryProvider.updatePrice(
            dstChainId, GasPrice.wrap(dstGasPrice), WeiPrice.wrap(dstNativeCurrencyPrice)
        );
        deliveryProvider.updatePrice(
            TEST_ORACLE_CHAIN_ID, GasPrice.wrap(srcGasPrice), WeiPrice.wrap(srcNativeCurrencyPrice)
        );

        // verify price
        uint256 expected = (
            uint256(dstNativeCurrencyPrice) * (uint256(dstGasPrice)) + (srcNativeCurrencyPrice - 1)
        ) / srcNativeCurrencyPrice;
        GasPrice readValues = deliveryProvider.quoteGasPrice(dstChainId);
        console.log(readValues.unwrap(), expected);
        require(readValues.unwrap() == expected, "deliveryProvider.quotePrices != expected");
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

        initializeDeliveryProvider();

        DeliveryProviderStructs.UpdatePrice[] memory updates =
            new DeliveryProviderStructs.UpdatePrice[](2);
        updates[0] = DeliveryProviderStructs.UpdatePrice({
            chainId: TEST_ORACLE_CHAIN_ID,
            gasPrice: GasPrice.wrap(srcGasPrice),
            nativeCurrencyPrice: WeiPrice.wrap(srcNativeCurrencyPrice)
        });
        updates[1] = DeliveryProviderStructs.UpdatePrice({
            chainId: dstChainId,
            gasPrice: GasPrice.wrap(dstGasPrice),
            nativeCurrencyPrice: WeiPrice.wrap(dstNativeCurrencyPrice)
        });

        // update the prices with reasonable values
        deliveryProvider.updatePrices(updates);

        // verify price
        uint256 expected = (
            uint256(dstNativeCurrencyPrice) * (uint256(dstGasPrice)) + (srcNativeCurrencyPrice - 1)
        ) / srcNativeCurrencyPrice;
        GasPrice readValues = deliveryProvider.quoteGasPrice(dstChainId);
        require(readValues.unwrap() == expected, "deliveryProvider.quotePrices != expected");
    }

    function testUpdateTargetChainContracts(uint16 targetChain, bytes32 newAddress) public {
        initializeDeliveryProvider();

        deliveryProvider.updateTargetChainAddress(targetChain, newAddress);
        bytes32 updated = deliveryProvider.getTargetChainAddress(targetChain);

        assertTrue(newAddress == updated);
    }

    function testUpdateRewardAddress(address payable newAddress) public {
        initializeDeliveryProvider();

        deliveryProvider.updateRewardAddress(newAddress);
        address payable updated = deliveryProvider.getRewardAddress();

        assertTrue(newAddress == updated);
    }

    function testQuoteDeliveryOverhead(
        uint16 dstChainId,
        uint64 dstGasPrice,
        uint64 dstNativeCurrencyPrice,
        uint64 srcGasPrice,
        uint64 srcNativeCurrencyPrice,
        uint32 gasOverhead
    ) public {
        initializeDeliveryProvider();

        vm.assume(dstChainId > 0);
        vm.assume(dstChainId != TEST_ORACLE_CHAIN_ID); // wormhole.chainId()
        vm.assume(dstGasPrice > 0);
        vm.assume(dstNativeCurrencyPrice > 0);
        vm.assume(srcGasPrice > 0);
        vm.assume(srcNativeCurrencyPrice > 0);
        vm.assume(dstGasPrice >= dstNativeCurrencyPrice / srcNativeCurrencyPrice);
        vm.assume(dstGasPrice * uint256(dstNativeCurrencyPrice) / srcNativeCurrencyPrice < 2 ** 72);

        vm.assume(gasOverhead < uint256(2)**31);



        // update the prices with reasonable values
        deliveryProvider.updatePrice(
            dstChainId, GasPrice.wrap(dstGasPrice), WeiPrice.wrap(dstNativeCurrencyPrice)
        );
        deliveryProvider.updatePrice(
            TEST_ORACLE_CHAIN_ID, GasPrice.wrap(srcGasPrice), WeiPrice.wrap(srcNativeCurrencyPrice)
        );

        deliveryProvider.updateAssetConversionBuffer(dstChainId, 5, 100);

        deliveryProvider.updateDeliverGasOverhead(dstChainId, Gas.wrap(gasOverhead));

        deliveryProvider.updateMaximumBudget(dstChainId, Wei.wrap(uint256(2)**191));

        // verify price
        uint256 expectedOverhead = (
            uint256(dstNativeCurrencyPrice) * (uint256(dstGasPrice) * gasOverhead) + (srcNativeCurrencyPrice - 1)
        ) / srcNativeCurrencyPrice;

        LocalNative deliveryOverhead = deliveryProvider.quoteDeliveryOverhead(dstChainId);

        require(expectedOverhead == LocalNative.unwrap(deliveryOverhead), "deliveryProvider overhead quote is not what is expected");
    }

    function testQuoteDeliveryPrice(
        uint16 dstChainId,
        uint64 dstGasPrice,
        uint64 dstNativeCurrencyPrice,
        uint64 srcGasPrice,
        uint64 srcNativeCurrencyPrice,
        uint32 gasLimit,
        uint32 gasOverhead,
        uint64 receiverValue
    ) public {
        initializeDeliveryProvider();

        vm.assume(dstChainId > 0);
        vm.assume(dstChainId != TEST_ORACLE_CHAIN_ID); // wormhole.chainId()
        vm.assume(dstGasPrice > 0);
        vm.assume(dstNativeCurrencyPrice > 0);
        vm.assume(srcGasPrice > 0);
        vm.assume(srcNativeCurrencyPrice > 0);
        vm.assume(dstGasPrice >= dstNativeCurrencyPrice / srcNativeCurrencyPrice);
        vm.assume(dstGasPrice * uint256(dstNativeCurrencyPrice) / srcNativeCurrencyPrice < 2 ** 72);

        vm.assume(gasLimit < uint256(2)**31);
        vm.assume(gasOverhead < uint256(2)**31);

        // update the prices with reasonable values
        deliveryProvider.updatePrice(
            dstChainId, GasPrice.wrap(dstGasPrice), WeiPrice.wrap(dstNativeCurrencyPrice)
        );
        deliveryProvider.updatePrice(
            TEST_ORACLE_CHAIN_ID, GasPrice.wrap(srcGasPrice), WeiPrice.wrap(srcNativeCurrencyPrice)
        );

        deliveryProvider.updateAssetConversionBuffer(dstChainId, 5, 100);

        deliveryProvider.updateDeliverGasOverhead(dstChainId, Gas.wrap(gasOverhead));

        deliveryProvider.updateMaximumBudget(dstChainId, Wei.wrap(uint256(2)**191));

        // verify price
        uint256 expectedGasCost = (
            uint256(dstNativeCurrencyPrice) * (uint256(dstGasPrice) * (gasLimit)) + (srcNativeCurrencyPrice - 1)
        ) / srcNativeCurrencyPrice;

        uint256 expectedOverheadCost = (
            uint256(dstNativeCurrencyPrice) * (uint256(dstGasPrice) * ( gasOverhead)) + (srcNativeCurrencyPrice - 1)
        ) / srcNativeCurrencyPrice;

        uint256 expectedReceiverValueCost = (
            uint256(dstNativeCurrencyPrice) * (receiverValue) * 105 + (srcNativeCurrencyPrice * uint256(100) - 1)
        ) / srcNativeCurrencyPrice / 100;

        (LocalNative nativePriceQuote,) = deliveryProvider.quoteEvmDeliveryPrice(dstChainId, Gas.wrap(gasLimit), TargetNative.wrap(receiverValue));

        require(expectedGasCost == LocalNative.unwrap(deliveryProvider.quoteGasCost(dstChainId, Gas.wrap(gasLimit))), "Gas cost is not what is expected");
        require(expectedOverheadCost == LocalNative.unwrap(deliveryProvider.quoteGasCost(dstChainId, Gas.wrap(gasOverhead))), "Overhead cost is not what is expected");

        require(expectedGasCost + expectedOverheadCost + expectedReceiverValueCost == LocalNative.unwrap(nativePriceQuote), "deliveryProvider price quote is not what is expected");

    }

    function testIsMessageKeyTypeSupported(uint8 keyType) public {
        initializeDeliveryProvider();

        assertFalse(deliveryProvider.isMessageKeyTypeSupported(keyType));
        deliveryProvider.updateSupportedMessageKeyTypes(keyType, true);
        assertTrue(deliveryProvider.isMessageKeyTypeSupported(keyType));
        deliveryProvider.updateSupportedMessageKeyTypes(keyType, false);
        assertFalse(deliveryProvider.isMessageKeyTypeSupported(keyType));

        assertFalse(deliveryProvider.isMessageKeyTypeSupported(15));
        deliveryProvider.updateSupportedMessageKeyTypes(15, true);
        assertTrue(deliveryProvider.isMessageKeyTypeSupported(15));
        deliveryProvider.updateSupportedMessageKeyTypes(15, false);
        assertFalse(deliveryProvider.isMessageKeyTypeSupported(15));
        assertFalse(deliveryProvider.isMessageKeyTypeSupported(15));
    }
}
