// SPDX-License-Identifier: Apache-2.0
pragma solidity >=0.5.0 <0.9.0;
pragma experimental ABIEncoderV2;

import "./IHederaTokenService.sol";

abstract contract FeeHelper {
    function createFixedHbarFee(uint32 amount, address feeCollector)
        internal
        pure
        returns (IHederaTokenService.FixedFee memory fixedFee)
    {
        fixedFee.amount = amount;
        fixedFee.useHbarsForPayment = true;
        fixedFee.feeCollector = feeCollector;
    }

    function createFixedTokenFee(
        uint32 amount,
        address tokenId,
        address feeCollector
    ) internal pure returns (IHederaTokenService.FixedFee memory fixedFee) {
        fixedFee.amount = amount;
        fixedFee.tokenId = tokenId;
        fixedFee.feeCollector = feeCollector;
    }

    function createFixedSelfDenominatedFee(uint32 amount, address feeCollector)
        internal
        pure
        returns (IHederaTokenService.FixedFee memory fixedFee)
    {
        fixedFee.amount = amount;
        fixedFee.useCurrentTokenForPayment = true;
        fixedFee.feeCollector = feeCollector;
    }

    function createFractionalFee(
        uint32 numerator,
        uint32 denominator,
        bool netOfTransfers,
        address feeCollector
    )
        internal
        pure
        returns (IHederaTokenService.FractionalFee memory fractionalFee)
    {
        fractionalFee.numerator = numerator;
        fractionalFee.denominator = denominator;
        fractionalFee.netOfTransfers = netOfTransfers;
        fractionalFee.feeCollector = feeCollector;
    }

    function createFractionalFeeWithMinAndMax(
        uint32 numerator,
        uint32 denominator,
        uint32 minimumAmount,
        uint32 maximumAmount,
        bool netOfTransfers,
        address feeCollector
    )
        internal
        pure
        returns (IHederaTokenService.FractionalFee memory fractionalFee)
    {
        fractionalFee.numerator = numerator;
        fractionalFee.denominator = denominator;
        fractionalFee.minimumAmount = minimumAmount;
        fractionalFee.maximumAmount = maximumAmount;
        fractionalFee.netOfTransfers = netOfTransfers;
        fractionalFee.feeCollector = feeCollector;
    }

    function createFractionalFeeWithLimits(
        uint32 numerator,
        uint32 denominator,
        uint32 minimumAmount,
        uint32 maximumAmount,
        bool netOfTransfers,
        address feeCollector
    )
        internal
        pure
        returns (IHederaTokenService.FractionalFee memory fractionalFee)
    {
        fractionalFee.numerator = numerator;
        fractionalFee.denominator = denominator;
        fractionalFee.minimumAmount = minimumAmount;
        fractionalFee.maximumAmount = maximumAmount;
        fractionalFee.netOfTransfers = netOfTransfers;
        fractionalFee.feeCollector = feeCollector;
    }

    function createRoyaltyFeeWithoutFallback(
        uint32 numerator,
        uint32 denominator,
        address feeCollector
    ) internal pure returns (IHederaTokenService.RoyaltyFee memory royaltyFee) {
        royaltyFee.numerator = numerator;
        royaltyFee.denominator = denominator;
        royaltyFee.feeCollector = feeCollector;
    }

    function createRoyaltyFeeWithHbarFallbackFee(
        uint32 numerator,
        uint32 denominator,
        uint32 amount,
        address feeCollector
    ) internal pure returns (IHederaTokenService.RoyaltyFee memory royaltyFee) {
        royaltyFee.numerator = numerator;
        royaltyFee.denominator = denominator;
        royaltyFee.amount = amount;
        royaltyFee.useHbarsForPayment = true;
        royaltyFee.feeCollector = feeCollector;
    }

    function createRoyaltyFeeWithTokenDenominatedFallbackFee(
        uint32 numerator,
        uint32 denominator,
        uint32 amount,
        address tokenId,
        address feeCollector
    ) internal pure returns (IHederaTokenService.RoyaltyFee memory royaltyFee) {
        royaltyFee.numerator = numerator;
        royaltyFee.denominator = denominator;
        royaltyFee.amount = amount;
        royaltyFee.tokenId = tokenId;
        royaltyFee.feeCollector = feeCollector;
    }

    function createNAmountFixedFeesForHbars(
        uint8 numberOfFees,
        uint32 amount,
        address feeCollector
    ) internal pure returns (IHederaTokenService.FixedFee[] memory fixedFees) {
        fixedFees = new IHederaTokenService.FixedFee[](numberOfFees);

        for (uint8 i = 0; i < numberOfFees; i++) {
            IHederaTokenService.FixedFee
                memory fixedFee = createFixedFeeForHbars(amount, feeCollector);
            fixedFees[i] = fixedFee;
        }
    }

    function createSingleFixedFeeForToken(
        uint32 amount,
        address tokenId,
        address feeCollector
    ) internal pure returns (IHederaTokenService.FixedFee[] memory fixedFees) {
        fixedFees = new IHederaTokenService.FixedFee[](1);
        IHederaTokenService.FixedFee memory fixedFee = createFixedFeeForToken(
            amount,
            tokenId,
            feeCollector
        );
        fixedFees[0] = fixedFee;
    }

    function createFixedFeesForToken(
        uint32 amount,
        address tokenId,
        address firstFeeCollector,
        address secondFeeCollector
    ) internal pure returns (IHederaTokenService.FixedFee[] memory fixedFees) {
        fixedFees = new IHederaTokenService.FixedFee[](1);
        IHederaTokenService.FixedFee memory fixedFee1 = createFixedFeeForToken(
            amount,
            tokenId,
            firstFeeCollector
        );
        IHederaTokenService.FixedFee memory fixedFee2 = createFixedFeeForToken(
            2 * amount,
            tokenId,
            secondFeeCollector
        );
        fixedFees[0] = fixedFee1;
        fixedFees[0] = fixedFee2;
    }

    function createSingleFixedFeeForHbars(uint32 amount, address feeCollector)
        internal
        pure
        returns (IHederaTokenService.FixedFee[] memory fixedFees)
    {
        fixedFees = new IHederaTokenService.FixedFee[](1);
        IHederaTokenService.FixedFee memory fixedFee = createFixedFeeForHbars(
            amount,
            feeCollector
        );
        fixedFees[0] = fixedFee;
    }

    function createSingleFixedFeeForCurrentToken(
        uint32 amount,
        address feeCollector
    ) internal pure returns (IHederaTokenService.FixedFee[] memory fixedFees) {
        fixedFees = new IHederaTokenService.FixedFee[](1);
        IHederaTokenService.FixedFee
            memory fixedFee = createFixedFeeForCurrentToken(
                amount,
                feeCollector
            );
        fixedFees[0] = fixedFee;
    }

    function createSingleFixedFeeWithInvalidFlags(
        uint32 amount,
        address feeCollector
    ) internal pure returns (IHederaTokenService.FixedFee[] memory fixedFees) {
        fixedFees = new IHederaTokenService.FixedFee[](1);
        IHederaTokenService.FixedFee
            memory fixedFee = createFixedFeeWithInvalidFlags(
                amount,
                feeCollector
            );
        fixedFees[0] = fixedFee;
    }

    function createSingleFixedFeeWithTokenIdAndHbars(
        uint32 amount,
        address tokenId,
        address feeCollector
    ) internal pure returns (IHederaTokenService.FixedFee[] memory fixedFees) {
        fixedFees = new IHederaTokenService.FixedFee[](1);
        IHederaTokenService.FixedFee
            memory fixedFee = createFixedFeeWithTokenIdAndHbars(
                amount,
                tokenId,
                feeCollector
            );
        fixedFees[0] = fixedFee;
    }

    function createFixedFeesWithAllTypes(
        uint32 amount,
        address tokenId,
        address feeCollector
    ) internal pure returns (IHederaTokenService.FixedFee[] memory fixedFees) {
        fixedFees = new IHederaTokenService.FixedFee[](3);
        IHederaTokenService.FixedFee
            memory fixedFeeForToken = createFixedFeeForToken(
                amount,
                tokenId,
                feeCollector
            );
        IHederaTokenService.FixedFee
            memory fixedFeeForHbars = createFixedFeeForHbars(
                amount * 2,
                feeCollector
            );
        IHederaTokenService.FixedFee
            memory fixedFeeForCurrentToken = createFixedFeeForCurrentToken(
                amount * 4,
                feeCollector
            );
        fixedFees[0] = fixedFeeForToken;
        fixedFees[1] = fixedFeeForHbars;
        fixedFees[2] = fixedFeeForCurrentToken;
    }

    function createFixedFeeForToken(
        uint32 amount,
        address tokenId,
        address feeCollector
    ) internal pure returns (IHederaTokenService.FixedFee memory fixedFee) {
        fixedFee.amount = amount;
        fixedFee.tokenId = tokenId;
        fixedFee.feeCollector = feeCollector;
    }

    function createFixedFeeForHbars(uint32 amount, address feeCollector)
        internal
        pure
        returns (IHederaTokenService.FixedFee memory fixedFee)
    {
        fixedFee.amount = amount;
        fixedFee.useHbarsForPayment = true;
        fixedFee.feeCollector = feeCollector;
    }

    function createFixedFeeForCurrentToken(uint32 amount, address feeCollector)
        internal
        pure
        returns (IHederaTokenService.FixedFee memory fixedFee)
    {
        fixedFee.amount = amount;
        fixedFee.useCurrentTokenForPayment = true;
        fixedFee.feeCollector = feeCollector;
    }

    //Used for negative scenarios
    function createFixedFeeWithInvalidFlags(uint32 amount, address feeCollector)
        internal
        pure
        returns (IHederaTokenService.FixedFee memory fixedFee)
    {
        fixedFee.amount = amount;
        fixedFee.useHbarsForPayment = true;
        fixedFee.useCurrentTokenForPayment = true;
        fixedFee.feeCollector = feeCollector;
    }

    //Used for negative scenarios
    function createFixedFeeWithTokenIdAndHbars(
        uint32 amount,
        address tokenId,
        address feeCollector
    ) internal pure returns (IHederaTokenService.FixedFee memory fixedFee) {
        fixedFee.amount = amount;
        fixedFee.tokenId = tokenId;
        fixedFee.useHbarsForPayment = true;
        fixedFee.feeCollector = feeCollector;
    }

    function getEmptyFixedFees()
        internal
        pure
        returns (IHederaTokenService.FixedFee[] memory fixedFees)
    {}

    function createNAmountFractionalFees(
        uint8 numberOfFees,
        uint32 numerator,
        uint32 denominator,
        bool netOfTransfers,
        address feeCollector
    )
        internal
        pure
        returns (IHederaTokenService.FractionalFee[] memory fractionalFees)
    {
        fractionalFees = new IHederaTokenService.FractionalFee[](numberOfFees);

        for (uint8 i = 0; i < numberOfFees; i++) {
            IHederaTokenService.FractionalFee
                memory fractionalFee = createFractionalFee(
                    numerator,
                    denominator,
                    netOfTransfers,
                    feeCollector
                );
            fractionalFees[i] = fractionalFee;
        }
    }

    function createSingleFractionalFee(
        uint32 numerator,
        uint32 denominator,
        bool netOfTransfers,
        address feeCollector
    )
        internal
        pure
        returns (IHederaTokenService.FractionalFee[] memory fractionalFees)
    {
        fractionalFees = new IHederaTokenService.FractionalFee[](1);
        IHederaTokenService.FractionalFee
            memory fractionalFee = createFractionalFee(
                numerator,
                denominator,
                netOfTransfers,
                feeCollector
            );
        fractionalFees[0] = fractionalFee;
    }

    function createSingleFractionalFeeWithLimits(
        uint32 numerator,
        uint32 denominator,
        uint32 minimumAmount,
        uint32 maximumAmount,
        bool netOfTransfers,
        address feeCollector
    )
        internal
        pure
        returns (IHederaTokenService.FractionalFee[] memory fractionalFees)
    {
        fractionalFees = new IHederaTokenService.FractionalFee[](1);
        IHederaTokenService.FractionalFee
            memory fractionalFee = createFractionalFeeWithLimits(
                numerator,
                denominator,
                minimumAmount,
                maximumAmount,
                netOfTransfers,
                feeCollector
            );
        fractionalFees[0] = fractionalFee;
    }

    function getEmptyFractionalFees()
        internal
        pure
        returns (IHederaTokenService.FractionalFee[] memory fractionalFees)
    {
        fractionalFees = new IHederaTokenService.FractionalFee[](0);
    }

    function createNAmountRoyaltyFees(
        uint8 numberOfFees,
        uint32 numerator,
        uint32 denominator,
        address feeCollector
    )
        internal
        pure
        returns (IHederaTokenService.RoyaltyFee[] memory royaltyFees)
    {
        royaltyFees = new IHederaTokenService.RoyaltyFee[](numberOfFees);

        for (uint8 i = 0; i < numberOfFees; i++) {
            IHederaTokenService.RoyaltyFee memory royaltyFee = createRoyaltyFee(
                numerator,
                denominator,
                feeCollector
            );
            royaltyFees[i] = royaltyFee;
        }
    }

    function getEmptyRoyaltyFees()
        internal
        pure
        returns (IHederaTokenService.RoyaltyFee[] memory royaltyFees)
    {
        royaltyFees = new IHederaTokenService.RoyaltyFee[](0);
    }

    function createSingleRoyaltyFee(
        uint32 numerator,
        uint32 denominator,
        address feeCollector
    )
        internal
        pure
        returns (IHederaTokenService.RoyaltyFee[] memory royaltyFees)
    {
        royaltyFees = new IHederaTokenService.RoyaltyFee[](1);

        IHederaTokenService.RoyaltyFee memory royaltyFee = createRoyaltyFee(
            numerator,
            denominator,
            feeCollector
        );
        royaltyFees[0] = royaltyFee;
    }

    function createSingleRoyaltyFeeWithFallbackFee(
        uint32 numerator,
        uint32 denominator,
        uint32 amount,
        address tokenId,
        bool useHbarsForPayment,
        address feeCollector
    )
        internal
        pure
        returns (IHederaTokenService.RoyaltyFee[] memory royaltyFees)
    {
        royaltyFees = new IHederaTokenService.RoyaltyFee[](1);

        IHederaTokenService.RoyaltyFee
            memory royaltyFee = createRoyaltyFeeWithFallbackFee(
                numerator,
                denominator,
                amount,
                tokenId,
                useHbarsForPayment,
                feeCollector
            );
        royaltyFees[0] = royaltyFee;
    }

    function createRoyaltyFeesWithAllTypes(
        uint32 numerator,
        uint32 denominator,
        uint32 amount,
        address tokenId,
        address feeCollector
    )
        internal
        pure
        returns (IHederaTokenService.RoyaltyFee[] memory royaltyFees)
    {
        royaltyFees = new IHederaTokenService.RoyaltyFee[](3);
        IHederaTokenService.RoyaltyFee
            memory royaltyFeeWithoutFallback = createRoyaltyFee(
                numerator,
                denominator,
                feeCollector
            );
        IHederaTokenService.RoyaltyFee
            memory royaltyFeeWithFallbackHbar = createRoyaltyFeeWithFallbackFee(
                numerator,
                denominator,
                amount,
                address(0x0),
                true,
                feeCollector
            );
        IHederaTokenService.RoyaltyFee
            memory royaltyFeeWithFallbackToken = createRoyaltyFeeWithFallbackFee(
                numerator,
                denominator,
                amount,
                tokenId,
                false,
                feeCollector
            );
        royaltyFees[0] = royaltyFeeWithoutFallback;
        royaltyFees[1] = royaltyFeeWithFallbackHbar;
        royaltyFees[2] = royaltyFeeWithFallbackToken;
    }

    function createRoyaltyFee(
        uint32 numerator,
        uint32 denominator,
        address feeCollector
    ) internal pure returns (IHederaTokenService.RoyaltyFee memory royaltyFee) {
        royaltyFee.numerator = numerator;
        royaltyFee.denominator = denominator;
        royaltyFee.feeCollector = feeCollector;
    }

    function createRoyaltyFeeWithFallbackFee(
        uint32 numerator,
        uint32 denominator,
        uint32 amount,
        address tokenId,
        bool useHbarsForPayment,
        address feeCollector
    ) internal pure returns (IHederaTokenService.RoyaltyFee memory royaltyFee) {
        royaltyFee.numerator = numerator;
        royaltyFee.denominator = denominator;
        royaltyFee.amount = amount;
        royaltyFee.tokenId = tokenId;
        royaltyFee.useHbarsForPayment = useHbarsForPayment;
        royaltyFee.feeCollector = feeCollector;
    }
}
