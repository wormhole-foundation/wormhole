// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "../../contracts/interfaces/relayer/IDeliveryProviderTyped.sol";
import "../../contracts/relayer/deliveryProvider/DeliveryProvider.sol";
import "../../contracts/relayer/deliveryProvider/DeliveryProviderSetup.sol";
import "../../contracts/relayer/deliveryProvider/DeliveryProviderImplementation.sol";
import "../../contracts/relayer/deliveryProvider/DeliveryProviderProxy.sol";
import "../../contracts/relayer/deliveryProvider/DeliveryProviderMessages.sol";
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

        initializeDeliveryProvider();

        // you shall not pass
        vm.prank(oracleOwner);
        vm.expectRevert(abi.encodeWithSignature("CallerMustBeOwner()"));
        deliveryProvider.updatePrice(updateChainId, updateGasPrice, updateNativeCurrencyPrice);
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

        initializeDeliveryProvider();

        // update the price with reasonable values
        deliveryProvider.updatePrice(dstChainId, dstGasPrice, dstNativeCurrencyPrice);

        // you shall not pass
        vm.expectRevert("srcNativeCurrencyPrice == 0");
        deliveryProvider.quoteDeliveryOverhead(dstChainId);
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

        initializeDeliveryProvider();

        // update the price with reasonable values
        //vm.prank(deliveryProvider.owner());
        deliveryProvider.updatePrice(TEST_ORACLE_CHAIN_ID, srcGasPrice, srcNativeCurrencyPrice);

        // you shall not pass
        vm.expectRevert("dstNativeCurrencyPrice == 0");
        deliveryProvider.quoteDeliveryOverhead(dstChainId);
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
}
