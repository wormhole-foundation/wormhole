// SPDX-License-Identifier: Apache-2.0
pragma solidity >=0.5.0 <0.9.0;
pragma experimental ABIEncoderV2;

import "./HederaResponseCodes.sol";
import "./IHederaTokenService.sol";

abstract contract HederaTokenService {
    address constant precompileAddress = address(0x167);
    // 90 days in seconds
    uint32 constant defaultAutoRenewPeriod = 7776000;

    modifier nonEmptyExpiry(IHederaTokenService.HederaToken memory token)
    {
        if (token.expiry.second == 0 && token.expiry.autoRenewPeriod == 0) {
            token.expiry.autoRenewPeriod = defaultAutoRenewPeriod;
        }
        _;
    }

    /// Performs transfers among combinations of tokens and hbars
    /// @param transferList the list of hbar transfers to do
    /// @param tokenTransfers the list of transfers to do
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @custom:version 0.3.0 the signature of the previous version was cryptoTransfer(TokenTransferList[] memory tokenTransfers)
    function cryptoTransfer(IHederaTokenService.TransferList memory transferList, IHederaTokenService.TokenTransferList[] memory tokenTransfers) internal
    returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.cryptoTransfer.selector, transferList, tokenTransfers));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Performs transfers among combinations of tokens and hbars
    /// @param transferList the list of hbar transfers to do
    /// @param tokenTransfers the list of transfers to do
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @custom:version 0.3.0 the signature of the previous version was cryptoTransfer(TokenTransferList[] memory tokenTransfers)
    function delegateCryptoTransfer(IHederaTokenService.TransferList memory transferList, IHederaTokenService.TokenTransferList[] memory tokenTransfers) internal
    returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.delegatecall(
            abi.encodeWithSelector(IHederaTokenService.cryptoTransfer.selector, transferList, tokenTransfers));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Mints an amount of the token to the defined treasury account
    /// @param token The token for which to mint tokens. If token does not exist, transaction results in
    ///              INVALID_TOKEN_ID
    /// @param amount Applicable to tokens of type FUNGIBLE_COMMON. The amount to mint to the Treasury Account.
    ///               Amount must be a positive non-zero number represented in the lowest denomination of the
    ///               token. The new supply must be lower than 2^63.
    /// @param metadata Applicable to tokens of type NON_FUNGIBLE_UNIQUE. A list of metadata that are being created.
    ///                 Maximum allowed size of each metadata is 100 bytes
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return newTotalSupply The new supply of tokens. For NFTs it is the total count of NFTs
    /// @return serialNumbers If the token is an NFT the newly generate serial numbers, otherwise empty.
    function mintToken(address token, uint64 amount, bytes[] memory metadata) internal
    returns (int responseCode, uint64 newTotalSupply, int64[] memory serialNumbers)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.mintToken.selector,
            token, amount, metadata));
        (responseCode, newTotalSupply, serialNumbers) =
        success
        ? abi.decode(result, (int32, uint64, int64[]))
        : (HederaResponseCodes.UNKNOWN, 0, new int64[](0));
    }

    /// Burns an amount of the token from the defined treasury account
    /// @param token The token for which to burn tokens. If token does not exist, transaction results in
    ///              INVALID_TOKEN_ID
    /// @param amount  Applicable to tokens of type FUNGIBLE_COMMON. The amount to burn from the Treasury Account.
    ///                Amount must be a positive non-zero number, not bigger than the token balance of the treasury
    ///                account (0; balance], represented in the lowest denomination.
    /// @param serialNumbers Applicable to tokens of type NON_FUNGIBLE_UNIQUE. The list of serial numbers to be burned.
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return newTotalSupply The new supply of tokens. For NFTs it is the total count of NFTs
    function burnToken(address token, uint64 amount, int64[] memory serialNumbers) internal
    returns (int responseCode, uint64 newTotalSupply)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.burnToken.selector,
            token, amount, serialNumbers));
        (responseCode, newTotalSupply) =
        success
        ? abi.decode(result, (int32, uint64))
        : (HederaResponseCodes.UNKNOWN, 0);
    }

    ///  Associates the provided account with the provided tokens. Must be signed by the provided
    ///  Account's key or called from the accounts contract key
    ///  If the provided account is not found, the transaction will resolve to INVALID_ACCOUNT_ID.
    ///  If the provided account has been deleted, the transaction will resolve to ACCOUNT_DELETED.
    ///  If any of the provided tokens is not found, the transaction will resolve to INVALID_TOKEN_REF.
    ///  If any of the provided tokens has been deleted, the transaction will resolve to TOKEN_WAS_DELETED.
    ///  If an association between the provided account and any of the tokens already exists, the
    ///  transaction will resolve to TOKEN_ALREADY_ASSOCIATED_TO_ACCOUNT.
    ///  If the provided account's associations count exceed the constraint of maximum token associations
    ///    per account, the transaction will resolve to TOKENS_PER_ACCOUNT_LIMIT_EXCEEDED.
    ///  On success, associations between the provided account and tokens are made and the account is
    ///    ready to interact with the tokens.
    /// @param account The account to be associated with the provided tokens
    /// @param tokens The tokens to be associated with the provided account. In the case of NON_FUNGIBLE_UNIQUE
    ///               Type, once an account is associated, it can hold any number of NFTs (serial numbers) of that
    ///               token type
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function associateTokens(address account, address[] memory tokens) internal returns (int responseCode) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.associateTokens.selector,
            account, tokens));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    function associateToken(address account, address token) internal returns (int responseCode) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.associateToken.selector,
            account, token));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Dissociates the provided account with the provided tokens. Must be signed by the provided
    /// Account's key.
    /// If the provided account is not found, the transaction will resolve to INVALID_ACCOUNT_ID.
    /// If the provided account has been deleted, the transaction will resolve to ACCOUNT_DELETED.
    /// If any of the provided tokens is not found, the transaction will resolve to INVALID_TOKEN_REF.
    /// If any of the provided tokens has been deleted, the transaction will resolve to TOKEN_WAS_DELETED.
    /// If an association between the provided account and any of the tokens does not exist, the
    /// transaction will resolve to TOKEN_NOT_ASSOCIATED_TO_ACCOUNT.
    /// If a token has not been deleted and has not expired, and the user has a nonzero balance, the
    /// transaction will resolve to TRANSACTION_REQUIRES_ZERO_TOKEN_BALANCES.
    /// If a <b>fungible token</b> has expired, the user can disassociate even if their token balance is
    /// not zero.
    /// If a <b>non fungible token</b> has expired, the user can <b>not</b> disassociate if their token
    /// balance is not zero. The transaction will resolve to TRANSACTION_REQUIRED_ZERO_TOKEN_BALANCES.
    /// On success, associations between the provided account and tokens are removed.
    /// @param account The account to be dissociated from the provided tokens
    /// @param tokens The tokens to be dissociated from the provided account.
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function dissociateTokens(address account, address[] memory tokens) internal returns (int responseCode) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.dissociateTokens.selector,
            account, tokens));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    function dissociateToken(address account, address token) internal returns (int responseCode) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.dissociateToken.selector,
            account, token));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Creates a Fungible Token with the specified properties
    /// @param token the basic properties of the token being created
    /// @param initialTotalSupply Specifies the initial supply of tokens to be put in circulation. The
    /// initial supply is sent to the Treasury Account. The supply is in the lowest denomination possible.
    /// @param decimals the number of decimal places a token is divisible by
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return tokenAddress the created token's address
    function createFungibleToken(
        IHederaTokenService.HederaToken memory token,
        uint256 initialTotalSupply,
        uint256 decimals) nonEmptyExpiry(token)
    internal returns (int32 responseCode, address tokenAddress) {
        (bool success, bytes memory result) = precompileAddress.call{value : msg.value}(
            abi.encodeWithSelector(IHederaTokenService.createFungibleToken.selector,
            token, initialTotalSupply, decimals));


        (responseCode, tokenAddress) = success ? abi.decode(result, (int32, address)) : (HederaResponseCodes.UNKNOWN, address(0));
    }

    /// Creates a Fungible Token with the specified properties
    /// @param token the basic properties of the token being created
    /// @param initialTotalSupply Specifies the initial supply of tokens to be put in circulation. The
    /// initial supply is sent to the Treasury Account. The supply is in the lowest denomination possible.
    /// @param decimals the number of decimal places a token is divisible by
    /// @param fixedFees list of fixed fees to apply to the token
    /// @param fractionalFees list of fractional fees to apply to the token
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return tokenAddress the created token's address
    function createFungibleTokenWithCustomFees(
        IHederaTokenService.HederaToken memory token,
        uint64 initialTotalSupply,
        uint32 decimals,
        IHederaTokenService.FixedFee[] memory fixedFees,
        IHederaTokenService.FractionalFee[] memory fractionalFees) nonEmptyExpiry(token)
    internal returns (int responseCode, address tokenAddress) {
        (bool success, bytes memory result) = precompileAddress.call{value : msg.value}(
            abi.encodeWithSelector(IHederaTokenService.createFungibleTokenWithCustomFees.selector,
            token, initialTotalSupply, decimals, fixedFees, fractionalFees));
        (responseCode, tokenAddress) = success ? abi.decode(result, (int32, address)) : (HederaResponseCodes.UNKNOWN, address(0));
    }

    /// Creates an Non Fungible Unique Token with the specified properties
    /// @param token the basic properties of the token being created
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return tokenAddress the created token's address
    function createNonFungibleToken(IHederaTokenService.HederaToken memory token) nonEmptyExpiry(token)
    internal returns (int responseCode, address tokenAddress) {
        (bool success, bytes memory result) = precompileAddress.call{value : msg.value}(
            abi.encodeWithSelector(IHederaTokenService.createNonFungibleToken.selector, token));
        (responseCode, tokenAddress) = success ? abi.decode(result, (int32, address)) : (HederaResponseCodes.UNKNOWN, address(0));
    }

    /// Creates an Non Fungible Unique Token with the specified properties
    /// @param token the basic properties of the token being created
    /// @param fixedFees list of fixed fees to apply to the token
    /// @param royaltyFees list of royalty fees to apply to the token
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return tokenAddress the created token's address
    function createNonFungibleTokenWithCustomFees(
        IHederaTokenService.HederaToken memory token,
        IHederaTokenService.FixedFee[] memory fixedFees,
        IHederaTokenService.RoyaltyFee[] memory royaltyFees) nonEmptyExpiry(token)
    internal returns (int responseCode, address tokenAddress) {
        (bool success, bytes memory result) = precompileAddress.call{value : msg.value}(
            abi.encodeWithSelector(IHederaTokenService.createNonFungibleTokenWithCustomFees.selector,
            token, fixedFees, royaltyFees));
        (responseCode, tokenAddress) = success ? abi.decode(result, (int32, address)) : (HederaResponseCodes.UNKNOWN, address(0));
    }

    /// Retrieves fungible specific token info for a fungible token
    /// @param token The ID of the token as a solidity address
    function getFungibleTokenInfo(address token) internal returns (int responseCode, IHederaTokenService.FungibleTokenInfo memory tokenInfo) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.getFungibleTokenInfo.selector, token));
        IHederaTokenService.FungibleTokenInfo memory defaultTokenInfo;
        (responseCode, tokenInfo) = success ? abi.decode(result, (int32, IHederaTokenService.FungibleTokenInfo)) : (HederaResponseCodes.UNKNOWN, defaultTokenInfo);
    }

    /// Retrieves general token info for a given token
    /// @param token The ID of the token as a solidity address
    function getTokenInfo(address token) internal returns (int responseCode, IHederaTokenService.TokenInfo memory tokenInfo) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.getTokenInfo.selector, token));
        IHederaTokenService.TokenInfo memory defaultTokenInfo;
        (responseCode, tokenInfo) = success ? abi.decode(result, (int32, IHederaTokenService.TokenInfo)) : (HederaResponseCodes.UNKNOWN, defaultTokenInfo);
    }

    /// Retrieves non-fungible specific token info for a given NFT
    /// @param token The ID of the token as a solidity address
    function getNonFungibleTokenInfo(address token, int64 serialNumber) internal returns (int responseCode, IHederaTokenService.NonFungibleTokenInfo memory tokenInfo) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.getNonFungibleTokenInfo.selector, token, serialNumber));
        IHederaTokenService.NonFungibleTokenInfo memory defaultTokenInfo;
        (responseCode, tokenInfo) = success ? abi.decode(result, (int32, IHederaTokenService.NonFungibleTokenInfo)) : (HederaResponseCodes.UNKNOWN, defaultTokenInfo);
    }

    /// Query token custom fees
    /// @param token The token address to check
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return fixedFees Set of fixed fees for `token`
    /// @return fractionalFees Set of fractional fees for `token`
    /// @return royaltyFees Set of royalty fees for `token`
    function getTokenCustomFees(address token) internal returns (int64 responseCode,
        IHederaTokenService.FixedFee[] memory fixedFees,
        IHederaTokenService.FractionalFee[] memory fractionalFees,
        IHederaTokenService.RoyaltyFee[] memory royaltyFees) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.getTokenCustomFees.selector, token));
        IHederaTokenService.FixedFee[] memory defaultFixedFees;
        IHederaTokenService.FractionalFee[] memory defaultFractionalFees;
        IHederaTokenService.RoyaltyFee[] memory defaultRoyaltyFees;
        (responseCode, fixedFees, fractionalFees, royaltyFees) =
        success ? abi.decode
        (result, (int32, IHederaTokenService.FixedFee[], IHederaTokenService.FractionalFee[], IHederaTokenService.RoyaltyFee[]))
        : (HederaResponseCodes.UNKNOWN, defaultFixedFees, defaultFractionalFees, defaultRoyaltyFees);
    }

    /// Allows spender to withdraw from your account multiple times, up to the value amount. If this function is called
    /// again it overwrites the current allowance with value.
    /// Only Applicable to Fungible Tokens
    /// @param token The hedera token address to approve
    /// @param spender the account authorized to spend
    /// @param amount the amount of tokens authorized to spend.
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function approve(address token, address spender, uint256 amount) internal returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.approve.selector,
            token, spender, amount));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Allows spender to withdraw from your account multiple times, up to the value amount. If this function is called
    /// again it overwrites the current allowance with value.
    /// Only Applicable to Fungible Tokens
    /// @param token The hedera token address to approve
    /// @param spender the account authorized to spend
    /// @param amount the amount of tokens authorized to spend.
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function delegateApprove(address token, address spender, uint256 amount) internal returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.delegatecall(
            abi.encodeWithSelector(IHederaTokenService.approve.selector,
            token, spender, amount));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Transfers `amount` tokens from `from` to `to` using the
    //  allowance mechanism. `amount` is then deducted from the caller's allowance.
    /// Only applicable to fungible tokens
    /// @param token The address of the fungible Hedera token to transfer
    /// @param from The account address of the owner of the token, on the behalf of which to transfer `amount` tokens
    /// @param to The account address of the receiver of the `amount` tokens
    /// @param amount The amount of tokens to transfer from `from` to `to`
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function transferFrom(address token, address from, address to, uint256 amount) external returns (int64 responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.transferFrom.selector,
            token, from, to, amount));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Transfers `amount` tokens from `from` to `to` using the
    //  allowance mechanism. `amount` is then deducted from the caller's allowance.
    /// Only applicable to fungible tokens
    /// @param token The address of the fungible Hedera token to transfer
    /// @param from The account address of the owner of the token, on the behalf of which to transfer `amount` tokens
    /// @param to The account address of the receiver of the `amount` tokens
    /// @param amount The amount of tokens to transfer from `from` to `to`
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function delegateTransferFrom(address token, address from, address to, uint256 amount) external returns (int64 responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.delegatecall(
            abi.encodeWithSelector(IHederaTokenService.transferFrom.selector,
            token, from, to, amount));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Transfers `serialNumber` of `token` from `from` to `to` using the allowance mechanism.
    /// Only applicable to NFT tokens
    /// @param token The address of the non-fungible Hedera token to transfer
    /// @param from The account address of the owner of `serialNumber` of `token`
    /// @param to The account address of the receiver of `serialNumber`
    /// @param serialNumber The NFT serial number to transfer
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function transferFromNFT(address token, address from, address to, uint256 serialNumber) external returns (int64 responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.transferFromNFT.selector,
            token, from, to, serialNumber));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Transfers `serialNumber` of `token` from `from` to `to` using the allowance mechanism.
    /// Only applicable to NFT tokens
    /// @param token The address of the non-fungible Hedera token to transfer
    /// @param from The account address of the owner of `serialNumber` of `token`
    /// @param to The account address of the receiver of `serialNumber`
    /// @param serialNumber The NFT serial number to transfer
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function delegateTransferFromNFT(address token, address from, address to, uint256 serialNumber) external returns (int64 responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.delegatecall(
            abi.encodeWithSelector(IHederaTokenService.transferFromNFT.selector,
            token, from, to, serialNumber));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Returns the amount which spender is still allowed to withdraw from owner.
    /// Only Applicable to Fungible Tokens
    /// @param token The Hedera token address to check the allowance of
    /// @param owner the owner of the tokens to be spent
    /// @param spender the spender of the tokens
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function allowance(address token, address owner, address spender) internal returns (int responseCode, uint256 amount)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.allowance.selector,
            token, owner, spender));
        (responseCode, amount) = success ? abi.decode(result, (int32, uint256)) : (HederaResponseCodes.UNKNOWN, 0);
    }

    /// Allow or reaffirm the approved address to transfer an NFT the approved address does not own.
    /// Only Applicable to NFT Tokens
    /// @param token The Hedera NFT token address to approve
    /// @param approved The new approved NFT controller.  To revoke approvals pass in the zero address.
    /// @param serialNumber The NFT serial number  to approve
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function approveNFT(address token, address approved, uint256 serialNumber) internal returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.approveNFT.selector,
            token, approved, serialNumber));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Allow or reaffirm the approved address to transfer an NFT the approved address does not own.
    /// Only Applicable to NFT Tokens
    /// @param token The Hedera NFT token address to approve
    /// @param approved The new approved NFT controller.  To revoke approvals pass in the zero address.
    /// @param serialNumber The NFT serial number  to approve
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function delegateApproveNFT(address token, address approved, uint256 serialNumber) internal returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.delegatecall(
            abi.encodeWithSelector(IHederaTokenService.approveNFT.selector,
            token, approved, serialNumber));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Get the approved address for a single NFT
    /// Only Applicable to NFT Tokens
    /// @param token The Hedera NFT token address to check approval
    /// @param serialNumber The NFT to find the approved address for
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return approved The approved address for this NFT, or the zero address if there is none
    function getApproved(address token, uint256 serialNumber) internal returns (int responseCode, address approved)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.getApproved.selector,
            token, serialNumber));
        (responseCode, approved) =
        success
        ? abi.decode(result, (int32, address))
        : (HederaResponseCodes.UNKNOWN, address(0));
    }

    /// Query if token account is frozen
    /// @param token The token address to check
    /// @param account The account address associated with the token
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return frozen True if `account` is frozen for `token`
    function isFrozen(address token, address account) internal returns (int64 responseCode, bool frozen){
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.isFrozen.selector, token, account));
        (responseCode, frozen) = success ? abi.decode(result, (int32, bool)) : (HederaResponseCodes.UNKNOWN, false);
    }

    /// Query if token account has kyc granted
    /// @param token The token address to check
    /// @param account The account address associated with the token
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return kycGranted True if `account` has kyc granted for `token`
    function isKyc(address token, address account) internal returns (int64 responseCode, bool kycGranted){
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.isKyc.selector, token, account));
        (responseCode, kycGranted) = success ? abi.decode(result, (int32, bool)) : (HederaResponseCodes.UNKNOWN, false);
    }

    /// Operation to freeze token account
    /// @param token The token address
    /// @param account The account address to be frozen
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function freezeToken(address token, address account) internal returns (int64 responseCode){
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.freezeToken.selector, token, account));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Operation to unfreeze token account
    /// @param token The token address
    /// @param account The account address to be unfrozen
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function unfreezeToken(address token, address account) internal returns (int64 responseCode){
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.unfreezeToken.selector, token, account));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Operation to grant kyc to token account
    /// @param token The token address
    /// @param account The account address to grant kyc
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function grantTokenKyc(address token, address account) internal returns (int64 responseCode){
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.grantTokenKyc.selector, token, account));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Operation to revoke kyc to token account
    /// @param token The token address
    /// @param account The account address to revoke kyc
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function revokeTokenKyc(address token, address account) internal returns (int64 responseCode){
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.revokeTokenKyc.selector, token, account));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Enable or disable approval for a third party ("operator") to manage
    ///  all of `msg.sender`'s assets
    /// @param token The Hedera NFT token address to approve
    /// @param operator Address to add to the set of authorized operators
    /// @param approved True if the operator is approved, false to revoke approval
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function setApprovalForAll(address token, address operator, bool approved) internal returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.setApprovalForAll.selector,
            token, operator, approved));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Enable or disable approval for a third party ("operator") to manage
    ///  all of `msg.sender`'s assets
    /// @param token The Hedera NFT token address to approve
    /// @param operator Address to add to the set of authorized operators
    /// @param approved True if the operator is approved, false to revoke approval
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function delegateSetApprovalForAll(address token, address operator, bool approved) internal returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.delegatecall(
            abi.encodeWithSelector(IHederaTokenService.setApprovalForAll.selector,
            token, operator, approved));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Query if an address is an authorized operator for another address
    /// Only Applicable to NFT Tokens
    /// @param token The Hedera NFT token address to approve
    /// @param owner The address that owns the NFTs
    /// @param operator The address that acts on behalf of the owner
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return approved True if `operator` is an approved operator for `owner`, false otherwise
    function isApprovedForAll(address token, address owner, address operator) internal returns (int responseCode, bool approved)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.isApprovedForAll.selector,
            token, owner, operator));
        (responseCode, approved) =
        success
        ? abi.decode(result, (int32, bool))
        : (HederaResponseCodes.UNKNOWN, false);
    }

    /// Query token default freeze status
    /// @param token The token address to check
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return defaultFreezeStatus True if `token` default freeze status is frozen.
    function getTokenDefaultFreezeStatus(address token) internal returns (int responseCode, bool defaultFreezeStatus) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.getTokenDefaultFreezeStatus.selector, token));
        (responseCode, defaultFreezeStatus) = success ? abi.decode(result, (int32, bool)) : (HederaResponseCodes.UNKNOWN, false);
    }

    /// Query token default kyc status
    /// @param token The token address to check
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return defaultKycStatus True if `token` default kyc status is KycNotApplicable and false if Revoked.
    function getTokenDefaultKycStatus(address token) internal returns (int responseCode, bool defaultKycStatus) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.getTokenDefaultKycStatus.selector, token));
        (responseCode, defaultKycStatus) = success ? abi.decode(result, (int32, bool)) : (HederaResponseCodes.UNKNOWN, false);
    }

    /**********************
     * ABI v1 calls       *
     **********************/

    /// Initiates a Fungible Token Transfer
    /// @param token The ID of the token as a solidity address
    /// @param accountIds account to do a transfer to/from
    /// @param amounts The amount from the accountId at the same index
    function transferTokens(address token, address[] memory accountIds, int64[] memory amounts) internal
    returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.transferTokens.selector,
            token, accountIds, amounts));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Initiates a Non-Fungable Token Transfer
    /// @param token The ID of the token as a solidity address
    /// @param sender the sender of an nft
    /// @param receiver the receiver of the nft sent by the same index at sender
    /// @param serialNumber the serial number of the nft sent by the same index at sender
    function transferNFTs(address token, address[] memory sender, address[] memory receiver, int64[] memory serialNumber)
    internal returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.transferNFTs.selector,
            token, sender, receiver, serialNumber));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Transfers tokens where the calling account/contract is implicitly the first entry in the token transfer list,
    /// where the amount is the value needed to zero balance the transfers. Regular signing rules apply for sending
    /// (positive amount) or receiving (negative amount)
    /// @param token The token to transfer to/from
    /// @param sender The sender for the transaction
    /// @param receiver The receiver of the transaction
    /// @param amount Non-negative value to send. a negative value will result in a failure.
    function transferToken(address token, address sender, address receiver, int64 amount) internal
    returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.transferToken.selector,
            token, sender, receiver, amount));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Transfers tokens where the calling account/contract is implicitly the first entry in the token transfer list,
    /// where the amount is the value needed to zero balance the transfers. Regular signing rules apply for sending
    /// (positive amount) or receiving (negative amount)
    /// @param token The token to transfer to/from
    /// @param sender The sender for the transaction
    /// @param receiver The receiver of the transaction
    /// @param serialNumber The serial number of the NFT to transfer.
    function transferNFT(address token, address sender, address receiver, int64 serialNumber) internal
    returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.transferNFT.selector,
            token, sender, receiver, serialNumber));
        responseCode = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Operation to pause token
    /// @param token The token address to be paused
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function pauseToken(address token) internal returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.pauseToken.selector, token));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Operation to unpause token
    /// @param token The token address to be unpaused
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function unpauseToken(address token) internal returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.unpauseToken.selector, token));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Operation to wipe fungible tokens from account
    /// @param token The token address
    /// @param account The account address to revoke kyc
    /// @param amount The number of tokens to wipe
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function wipeTokenAccount(address token, address account, uint32 amount) internal returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.wipeTokenAccount.selector, token, account, amount));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Operation to wipe non fungible tokens from account
    /// @param token The token address
    /// @param account The account address to revoke kyc
    /// @param  serialNumbers The serial numbers of token to wipe
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function wipeTokenAccountNFT(address token, address account, int64[] memory serialNumbers) internal
    returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.wipeTokenAccountNFT.selector, token, account, serialNumbers));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Operation to delete token
    /// @param token The token address
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function deleteToken(address token) internal returns (int responseCode)
    {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.deleteToken.selector, token));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Operation to update token expiry info
    /// @param token The token address
    /// @param keys The token keys
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function updateTokenKeys(address token, IHederaTokenService.TokenKey[] memory keys)
    internal returns (int64 responseCode){
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.updateTokenKeys.selector, token, keys));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Query token KeyValue
    /// @param token The token address to check
    /// @param keyType The keyType of the desired KeyValue
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return key KeyValue info for key of type `keyType`
    function getTokenKey(address token, uint keyType)
    internal returns (int64 responseCode, IHederaTokenService.KeyValue memory key){
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.getTokenKey.selector, token, keyType));
        IHederaTokenService.KeyValue memory defaultKeyValueInfo;
        (responseCode, key) = success ? abi.decode(result, (int32,IHederaTokenService.KeyValue) ) : (HederaResponseCodes.UNKNOWN, defaultKeyValueInfo);
    }


    /// Query if valid token found for the given address
    /// @param token The token address
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return isTokenFlag True if valid token found for the given address
    function isToken(address token) internal returns (int64 responseCode, bool isTokenFlag) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.isToken.selector, token));
        (responseCode, isTokenFlag) = success ? abi.decode(result, (int32, bool)) : (HederaResponseCodes.UNKNOWN, false);
    }

    /// Query to return the token type for a given address
    /// @param token The token address
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return tokenType the token type. 0 is FUNGIBLE_COMMON, 1 is NON_FUNGIBLE_UNIQUE, -1 is UNRECOGNIZED
    function getTokenType(address token) internal returns (int64 responseCode, int32 tokenType) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.getTokenType.selector, token));
        (responseCode, tokenType) = success ? abi.decode(result, (int32, int32)) : (HederaResponseCodes.UNKNOWN, - 1);
    }

    /// Operation to get token expiry info
    /// @param token The token address
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return expiryInfo The expiry info of the token
    function getTokenExpiryInfo(address token) internal returns (int responseCode, IHederaTokenService.Expiry memory expiryInfo){
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.getTokenExpiryInfo.selector, token));
        IHederaTokenService.Expiry memory defaultExpiryInfo;
        (responseCode, expiryInfo) = success ? abi.decode(result, (int32, IHederaTokenService.Expiry)) : (HederaResponseCodes.UNKNOWN, defaultExpiryInfo);
    }

    /// Operation to update token expiry info
    /// @param token The token address
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function updateTokenExpiryInfo(address token, IHederaTokenService.Expiry memory expiryInfo) internal returns (int responseCode){
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.updateTokenExpiryInfo.selector, token, expiryInfo));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }

    /// Operation to update token info
    /// @param token The token address
    /// @param tokenInfo The hedera token info to update token with
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function updateTokenInfo(address token, IHederaTokenService.HederaToken memory tokenInfo) internal returns (int responseCode) {
        (bool success, bytes memory result) = precompileAddress.call(
            abi.encodeWithSelector(IHederaTokenService.updateTokenInfo.selector, token, tokenInfo));
        (responseCode) = success ? abi.decode(result, (int32)) : HederaResponseCodes.UNKNOWN;
    }
}
