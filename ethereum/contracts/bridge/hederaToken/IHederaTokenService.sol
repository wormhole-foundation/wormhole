// SPDX-License-Identifier: Apache-2.0
pragma solidity >=0.4.9 <0.9.0;
pragma experimental ABIEncoderV2;

interface IHederaTokenService {

    /// Transfers cryptocurrency among two or more accounts by making the desired adjustments to their
    /// balances. Each transfer list can specify up to 10 adjustments. Each negative amount is withdrawn
    /// from the corresponding account (a sender), and each positive one is added to the corresponding
    /// account (a receiver). The amounts list must sum to zero. Each amount is a number of tinybars
    /// (there are 100,000,000 tinybars in one hbar).  If any sender account fails to have sufficient
    /// hbars, then the entire transaction fails, and none of those transfers occur, though the
    /// transaction fee is still charged. This transaction must be signed by the keys for all the sending
    /// accounts, and for any receiving accounts that have receiverSigRequired == true. The signatures
    /// are in the same order as the accounts, skipping those accounts that don't need a signature.
    /// @custom:version 0.3.0 previous version did not include isApproval
    struct AccountAmount {
        // The Account ID, as a solidity address, that sends/receives cryptocurrency or tokens
        address accountID;

        // The amount of  the lowest denomination of the given token that
        // the account sends(negative) or receives(positive)
        int64 amount;

        // If true then the transfer is expected to be an approved allowance and the
        // accountID is expected to be the owner. The default is false (omitted).
        bool isApproval;
    }

    /// A sender account, a receiver account, and the serial number of an NFT of a Token with
    /// NON_FUNGIBLE_UNIQUE type. When minting NFTs the sender will be the default AccountID instance
    /// (0.0.0 aka 0x0) and when burning NFTs, the receiver will be the default AccountID instance.
    /// @custom:version 0.3.0 previous version did not include isApproval
    struct NftTransfer {
        // The solidity address of the sender
        address senderAccountID;

        // The solidity address of the receiver
        address receiverAccountID;

        // The serial number of the NFT
        int64 serialNumber;

        // If true then the transfer is expected to be an approved allowance and the
        // accountID is expected to be the owner. The default is false (omitted).
        bool isApproval;
    }

    struct TokenTransferList {
        // The ID of the token as a solidity address
        address token;

        // Applicable to tokens of type FUNGIBLE_COMMON. Multiple list of AccountAmounts, each of which
        // has an account and amount.
        AccountAmount[] transfers;

        // Applicable to tokens of type NON_FUNGIBLE_UNIQUE. Multiple list of NftTransfers, each of
        // which has a sender and receiver account, including the serial number of the NFT
        NftTransfer[] nftTransfers;
    }

    struct TransferList {
        // Multiple list of AccountAmounts, each of which has an account and amount.
        // Used to transfer hbars between the accounts in the list.
        AccountAmount[] transfers;
    }

    /// Expiry properties of a Hedera token - second, autoRenewAccount, autoRenewPeriod
    struct Expiry {
        // The epoch second at which the token should expire; if an auto-renew account and period are
        // specified, this is coerced to the current epoch second plus the autoRenewPeriod
        uint32 second;

        // ID of an account which will be automatically charged to renew the token's expiration, at
        // autoRenewPeriod interval, expressed as a solidity address
        address autoRenewAccount;

        // The interval at which the auto-renew account will be charged to extend the token's expiry
        uint32 autoRenewPeriod;
    }

    /// A Key can be a public key from either the Ed25519 or ECDSA(secp256k1) signature schemes, where
    /// in the ECDSA(secp256k1) case we require the 33-byte compressed form of the public key. We call
    /// these public keys <b>primitive keys</b>.
    /// A Key can also be the ID of a smart contract instance, which is then authorized to perform any
    /// precompiled contract action that requires this key to sign.
    /// Note that when a Key is a smart contract ID, it <i>doesn't</i> mean the contract with that ID
    /// will actually create a cryptographic signature. It only means that when the contract calls a
    /// precompiled contract, the resulting "child transaction" will be authorized to perform any action
    /// controlled by the Key.
    /// Exactly one of the possible values should be populated in order for the Key to be valid.
    struct KeyValue {

        // if set to true, the key of the calling Hedera account will be inherited as the token key
        bool inheritAccountKey;

        // smart contract instance that is authorized as if it had signed with a key
        address contractId;

        // Ed25519 public key bytes
        bytes ed25519;

        // Compressed ECDSA(secp256k1) public key bytes
        bytes ECDSA_secp256k1;

        // A smart contract that, if the recipient of the active message frame, should be treated
        // as having signed. (Note this does not mean the <i>code being executed in the frame</i>
        // will belong to the given contract, since it could be running another contract's code via
        // <tt>delegatecall</tt>. So setting this key is a more permissive version of setting the
        // contractID key, which also requires the code in the active message frame belong to the
        // the contract with the given id.)
        address delegatableContractId;
    }

    /// A list of token key types the key should be applied to and the value of the key
    struct TokenKey {

        // bit field representing the key type. Keys of all types that have corresponding bits set to 1
        // will be created for the token.
        // 0th bit: adminKey
        // 1st bit: kycKey
        // 2nd bit: freezeKey
        // 3rd bit: wipeKey
        // 4th bit: supplyKey
        // 5th bit: feeScheduleKey
        // 6th bit: pauseKey
        // 7th bit: ignored
        uint keyType;

        // the value that will be set to the key type
        KeyValue key;
    }

    /// Basic properties of a Hedera Token - name, symbol, memo, tokenSupplyType, maxSupply,
    /// treasury, freezeDefault. These properties are related both to Fungible and NFT token types.
    struct HederaToken {
        // The publicly visible name of the token. The token name is specified as a Unicode string.
        // Its UTF-8 encoding cannot exceed 100 bytes, and cannot contain the 0 byte (NUL).
        string name;

        // The publicly visible token symbol. The token symbol is specified as a Unicode string.
        // Its UTF-8 encoding cannot exceed 100 bytes, and cannot contain the 0 byte (NUL).
        string symbol;

        // The ID of the account which will act as a treasury for the token as a solidity address.
        // This account will receive the specified initial supply or the newly minted NFTs in
        // the case for NON_FUNGIBLE_UNIQUE Type
        address treasury;

        // The memo associated with the token (UTF-8 encoding max 100 bytes)
        string memo;

        // IWA compatibility. Specified the token supply type. Defaults to INFINITE
        bool tokenSupplyType;

        // IWA Compatibility. Depends on TokenSupplyType. For tokens of type FUNGIBLE_COMMON - the
        // maximum number of tokens that can be in circulation. For tokens of type NON_FUNGIBLE_UNIQUE -
        // the maximum number of NFTs (serial numbers) that can be minted. This field can never be changed!
        int64 maxSupply;

        // The default Freeze status (frozen or unfrozen) of Hedera accounts relative to this token. If
        // true, an account must be unfrozen before it can receive the token
        bool freezeDefault;

        // list of keys to set to the token
        TokenKey[] tokenKeys;

        // expiry properties of a Hedera token - second, autoRenewAccount, autoRenewPeriod
        Expiry expiry;
    }

    /// Additional post creation fungible and non fungible properties of a Hedera Token.
    struct TokenInfo {
        /// Basic properties of a Hedera Token
        HederaToken token;

        /// The number of tokens (fungible) or serials (non-fungible) of the token
        uint64 totalSupply;

        /// Specifies whether the token is deleted or not
        bool deleted;

        /// Specifies whether the token kyc was defaulted with KycNotApplicable (true) or Revoked (false)
        bool defaultKycStatus;

        /// Specifies whether the token is currently paused or not
        bool pauseStatus;

        /// The fixed fees collected when transferring the token
        FixedFee[] fixedFees;

        /// The fractional fees collected when transferring the token
        FractionalFee[] fractionalFees;

        /// The royalty fees collected when transferring the token
        RoyaltyFee[] royaltyFees;

        /// The ID of the network ledger
        string ledgerId;
    }

    /// Additional fungible properties of a Hedera Token.
    struct FungibleTokenInfo {
        /// The shared hedera token info
        TokenInfo tokenInfo;

        /// The number of decimal places a token is divisible by
        uint32 decimals;
    }

    /// Additional non fungible properties of a Hedera Token.
    struct NonFungibleTokenInfo {
        /// The shared hedera token info
        TokenInfo tokenInfo;

        /// The serial number of the nft
        int64 serialNumber;

        /// The account id specifying the owner of the non fungible token
        address ownerId;

        /// The epoch second at which the token was created.
        int64 creationTime;

        /// The unique metadata of the NFT
        bytes metadata;

        /// The account id specifying an account that has been granted spending permissions on this nft
        address spenderId;
    }

    /// A fixed number of units (hbar or token) to assess as a fee during a transfer of
    /// units of the token to which this fixed fee is attached. The denomination of
    /// the fee depends on the values of tokenId, useHbarsForPayment and
    /// useCurrentTokenForPayment. Exactly one of the values should be set.
    struct FixedFee {

        uint32 amount;

        // Specifies ID of token that should be used for fixed fee denomination
        address tokenId;

        // Specifies this fixed fee should be denominated in Hbar
        bool useHbarsForPayment;

        // Specifies this fixed fee should be denominated in the Token currently being created
        bool useCurrentTokenForPayment;

        // The ID of the account to receive the custom fee, expressed as a solidity address
        address feeCollector;
    }

    /// A fraction of the transferred units of a token to assess as a fee. The amount assessed will never
    /// be less than the given minimumAmount, and never greater than the given maximumAmount.  The
    /// denomination is always units of the token to which this fractional fee is attached.
    struct FractionalFee {
        // A rational number's numerator, used to set the amount of a value transfer to collect as a custom fee
        uint32 numerator;

        // A rational number's denominator, used to set the amount of a value transfer to collect as a custom fee
        uint32 denominator;

        // The minimum amount to assess
        uint32 minimumAmount;

        // The maximum amount to assess (zero implies no maximum)
        uint32 maximumAmount;
        bool netOfTransfers;

        // The ID of the account to receive the custom fee, expressed as a solidity address
        address feeCollector;
    }

    /// A fee to assess during a transfer that changes ownership of an NFT. Defines the fraction of
    /// the fungible value exchanged for an NFT that the ledger should collect as a royalty. ("Fungible
    /// value" includes both ℏ and units of fungible HTS tokens.) When the NFT sender does not receive
    /// any fungible value, the ledger will assess the fallback fee, if present, to the new NFT owner.
    /// Royalty fees can only be added to tokens of type type NON_FUNGIBLE_UNIQUE.
    struct RoyaltyFee {
        // A fraction's numerator of fungible value exchanged for an NFT to collect as royalty
        uint32 numerator;

        // A fraction's denominator of fungible value exchanged for an NFT to collect as royalty
        uint32 denominator;

        // If present, the fee to assess to the NFT receiver when no fungible value
        // is exchanged with the sender. Consists of:
        // amount: the amount to charge for the fee
        // tokenId: Specifies ID of token that should be used for fixed fee denomination
        // useHbarsForPayment: Specifies this fee should be denominated in Hbar
        uint32 amount;
        address tokenId;
        bool useHbarsForPayment;

        // The ID of the account to receive the custom fee, expressed as a solidity address
        address feeCollector;
    }

    /**********************
     * Direct HTS Calls   *
     **********************/

    /// Performs transfers among combinations of tokens and hbars
    /// @param transferList the list of hbar transfers to do
    /// @param tokenTransfers the list of token transfers to do
    /// @custom:version 0.3.0 the signature of the previous version was cryptoTransfer(TokenTransferList[] memory tokenTransfers)
    function cryptoTransfer(TransferList memory transferList, TokenTransferList[] memory tokenTransfers)
        external
        returns (int64 responseCode);

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
    /// @return serialNumbers If the token is an NFT the newly generate serial numbers, othersise empty.
    function mintToken(
        address token,
        uint64 amount,
        bytes[] memory metadata
    )
        external
        returns (
            int64 responseCode,
            uint64 newTotalSupply,
            int64[] memory serialNumbers
        );

    /// Burns an amount of the token from the defined treasury account
    /// @param token The token for which to burn tokens. If token does not exist, transaction results in
    ///              INVALID_TOKEN_ID
    /// @param amount  Applicable to tokens of type FUNGIBLE_COMMON. The amount to burn from the Treasury Account.
    ///                Amount must be a positive non-zero number, not bigger than the token balance of the treasury
    ///                account (0; balance], represented in the lowest denomination.
    /// @param serialNumbers Applicable to tokens of type NON_FUNGIBLE_UNIQUE. The list of serial numbers to be burned.
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return newTotalSupply The new supply of tokens. For NFTs it is the total count of NFTs
    function burnToken(
        address token,
        uint64 amount,
        int64[] memory serialNumbers
    ) external returns (int64 responseCode, uint64 newTotalSupply);

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
    function associateTokens(address account, address[] memory tokens)
        external
        returns (int64 responseCode);

    /// Single-token variant of associateTokens. Will be mapped to a single entry array call of associateTokens
    /// @param account The account to be associated with the provided token
    /// @param token The token to be associated with the provided account
    function associateToken(address account, address token)
        external
        returns (int64 responseCode);

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
    function dissociateTokens(address account, address[] memory tokens)
        external
        returns (int64 responseCode);

    /// Single-token variant of dissociateTokens. Will be mapped to a single entry array call of dissociateTokens
    /// @param account The account to be associated with the provided token
    /// @param token The token to be associated with the provided account
    function dissociateToken(address account, address token)
        external
        returns (int64 responseCode);

    /// Creates a Fungible Token with the specified properties
    /// @param token the basic properties of the token being created
    /// @param initialTotalSupply Specifies the initial supply of tokens to be put in circulation. The
    /// initial supply is sent to the Treasury Account. The supply is in the lowest denomination possible.
    /// @param decimals the number of decimal places a token is divisible by
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return tokenAddress the created token's address
    function createFungibleToken(
        HederaToken memory token,
        uint64 initialTotalSupply,
        uint32 decimals
    ) external payable returns (int64 responseCode, address tokenAddress);

    /// Creates a Fungible Token with the specified properties
    /// @param token the basic properties of the token being created
    /// @param initialTotalSupply Specifies the initial supply of tokens to be put in circulation. The
    /// initial supply is sent to the Treasury Account. The supply is in the lowest denomination possible.
    /// @param decimals the number of decimal places a token is divisible by.
    /// @param fixedFees list of fixed fees to apply to the token
    /// @param fractionalFees list of fractional fees to apply to the token
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return tokenAddress the created token's address
    function createFungibleTokenWithCustomFees(
        HederaToken memory token,
        uint64 initialTotalSupply,
        uint32 decimals,
        FixedFee[] memory fixedFees,
        FractionalFee[] memory fractionalFees
    ) external payable returns (int64 responseCode, address tokenAddress);

    /// Creates an Non Fungible Unique Token with the specified properties
    /// @param token the basic properties of the token being created
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return tokenAddress the created token's address
    function createNonFungibleToken(HederaToken memory token)
        external
        payable
        returns (int64 responseCode, address tokenAddress);

    /// Creates an Non Fungible Unique Token with the specified properties
    /// @param token the basic properties of the token being created
    /// @param fixedFees list of fixed fees to apply to the token
    /// @param royaltyFees list of royalty fees to apply to the token
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return tokenAddress the created token's address
    function createNonFungibleTokenWithCustomFees(
        HederaToken memory token,
        FixedFee[] memory fixedFees,
        RoyaltyFee[] memory royaltyFees
    ) external payable returns (int64 responseCode, address tokenAddress);

    /**********************
     * ABIV1 calls        *
     **********************/

    /// Initiates a Fungible Token Transfer
    /// @param token The ID of the token as a solidity address
    /// @param accountId account to do a transfer to/from
    /// @param amount The amount from the accountId at the same index
    function transferTokens(
        address token,
        address[] memory accountId,
        int64[] memory amount
    ) external returns (int64 responseCode);

    /// Initiates a Non-Fungable Token Transfer
    /// @param token The ID of the token as a solidity address
    /// @param sender the sender of an nft
    /// @param receiver the receiver of the nft sent by the same index at sender
    /// @param serialNumber the serial number of the nft sent by the same index at sender
    function transferNFTs(
        address token,
        address[] memory sender,
        address[] memory receiver,
        int64[] memory serialNumber
    ) external returns (int64 responseCode);

    /// Transfers tokens where the calling account/contract is implicitly the first entry in the token transfer list,
    /// where the amount is the value needed to zero balance the transfers. Regular signing rules apply for sending
    /// (positive amount) or receiving (negative amount)
    /// @param token The token to transfer to/from
    /// @param sender The sender for the transaction
    /// @param recipient The receiver of the transaction
    /// @param amount Non-negative value to send. a negative value will result in a failure.
    function transferToken(
        address token,
        address sender,
        address recipient,
        int64 amount
    ) external returns (int64 responseCode);

    /// Transfers tokens where the calling account/contract is implicitly the first entry in the token transfer list,
    /// where the amount is the value needed to zero balance the transfers. Regular signing rules apply for sending
    /// (positive amount) or receiving (negative amount)
    /// @param token The token to transfer to/from
    /// @param sender The sender for the transaction
    /// @param recipient The receiver of the transaction
    /// @param serialNumber The serial number of the NFT to transfer.
    function transferNFT(
        address token,
        address sender,
        address recipient,
        int64 serialNumber
    ) external returns (int64 responseCode);

    /// Allows spender to withdraw from your account multiple times, up to the value amount. If this function is called
    /// again it overwrites the current allowance with value.
    /// Only Applicable to Fungible Tokens
    /// @param token The hedera token address to approve
    /// @param spender the account address authorized to spend
    /// @param amount the amount of tokens authorized to spend.
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function approve(
        address token,
        address spender,
        uint256 amount
    ) external returns (int64 responseCode);

    /// Transfers `amount` tokens from `from` to `to` using the
    //  allowance mechanism. `amount` is then deducted from the caller's allowance.
    /// Only applicable to fungible tokens
    /// @param token The address of the fungible Hedera token to transfer
    /// @param from The account address of the owner of the token, on the behalf of which to transfer `amount` tokens
    /// @param to The account address of the receiver of the `amount` tokens
    /// @param amount The amount of tokens to transfer from `from` to `to`
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function transferFrom(address token, address from, address to, uint256 amount) external returns (int64 responseCode);

    /// Returns the amount which spender is still allowed to withdraw from owner.
    /// Only Applicable to Fungible Tokens
    /// @param token The Hedera token address to check the allowance of
    /// @param owner the owner of the tokens to be spent
    /// @param spender the spender of the tokens
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return allowance The amount which spender is still allowed to withdraw from owner.
    function allowance(
        address token,
        address owner,
        address spender
    ) external returns (int64 responseCode, uint256 allowance);

    /// Allow or reaffirm the approved address to transfer an NFT the approved address does not own.
    /// Only Applicable to NFT Tokens
    /// @param token The Hedera NFT token address to approve
    /// @param approved The new approved NFT controller.  To revoke approvals pass in the zero address.
    /// @param serialNumber The NFT serial number  to approve
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function approveNFT(
        address token,
        address approved,
        uint256 serialNumber
    ) external returns (int64 responseCode);

    /// Transfers `serialNumber` of `token` from `from` to `to` using the allowance mechanism.
    /// Only applicable to NFT tokens
    /// @param token The address of the non-fungible Hedera token to transfer
    /// @param from The account address of the owner of `serialNumber` of `token`
    /// @param to The account address of the receiver of `serialNumber`
    /// @param serialNumber The NFT serial number to transfer
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function transferFromNFT(address token, address from, address to, uint256 serialNumber) external returns (int64 responseCode);

    /// Get the approved address for a single NFT
    /// Only Applicable to NFT Tokens
    /// @param token The Hedera NFT token address to check approval
    /// @param serialNumber The NFT to find the approved address for
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return approved The approved address for this NFT, or the zero address if there is none
    function getApproved(address token, uint256 serialNumber)
        external
        returns (int64 responseCode, address approved);

    /// Enable or disable approval for a third party ("operator") to manage
    ///  all of `msg.sender`'s assets
    /// @param token The Hedera NFT token address to approve
    /// @param operator Address to add to the set of authorized operators
    /// @param approved True if the operator is approved, false to revoke approval
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function setApprovalForAll(
        address token,
        address operator,
        bool approved
    ) external returns (int64 responseCode);

    /// Query if an address is an authorized operator for another address
    /// Only Applicable to NFT Tokens
    /// @param token The Hedera NFT token address to approve
    /// @param owner The address that owns the NFTs
    /// @param operator The address that acts on behalf of the owner
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return approved True if `operator` is an approved operator for `owner`, false otherwise
    function isApprovedForAll(
        address token,
        address owner,
        address operator
    ) external returns (int64 responseCode, bool approved);

    /// Query if token account is frozen
    /// @param token The token address to check
    /// @param account The account address associated with the token
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return frozen True if `account` is frozen for `token`
    function isFrozen(address token, address account)
        external
        returns (int64 responseCode, bool frozen);

    /// Query if token account has kyc granted
    /// @param token The token address to check
    /// @param account The account address associated with the token
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return kycGranted True if `account` has kyc granted for `token`
    function isKyc(address token, address account)
        external
        returns (int64 responseCode, bool kycGranted);

    /// Operation to delete token
    /// @param token The token address to be deleted
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function deleteToken(address token) external returns (int64 responseCode);

    /// Query token custom fees
    /// @param token The token address to check
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return fixedFees Set of fixed fees for `token`
    /// @return fractionalFees Set of fractional fees for `token`
    /// @return royaltyFees Set of royalty fees for `token`
    function getTokenCustomFees(address token)
        external
        returns (int64 responseCode, FixedFee[] memory fixedFees, FractionalFee[] memory fractionalFees, RoyaltyFee[] memory royaltyFees);

    /// Query token default freeze status
    /// @param token The token address to check
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return defaultFreezeStatus True if `token` default freeze status is frozen.
    function getTokenDefaultFreezeStatus(address token)
        external
        returns (int64 responseCode, bool defaultFreezeStatus);
    
    /// Query token default kyc status
    /// @param token The token address to check
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return defaultKycStatus True if `token` default kyc status is KycNotApplicable and false if Revoked.
    function getTokenDefaultKycStatus(address token)
        external
        returns (int64 responseCode, bool defaultKycStatus);

    /// Query token expiry info
    /// @param token The token address to check
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return expiry Expiry info for `token`
    function getTokenExpiryInfo(address token)
        external
        returns (int64 responseCode, Expiry memory expiry);

    /// Query fungible token info
    /// @param token The token address to check
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return fungibleTokenInfo FungibleTokenInfo info for `token`
    function getFungibleTokenInfo(address token)
        external
        returns (int64 responseCode, FungibleTokenInfo memory fungibleTokenInfo);

    /// Query token info
    /// @param token The token address to check
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return tokenInfo TokenInfo info for `token`
    function getTokenInfo(address token)
        external
        returns (int64 responseCode, TokenInfo memory tokenInfo);

    /// Query token KeyValue
    /// @param token The token address to check
    /// @param keyType The keyType of the desired KeyValue
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return key KeyValue info for key of type `keyType`
    function getTokenKey(address token, uint keyType)
        external
        returns (int64 responseCode, KeyValue memory key);

    /// Query non fungible token info
    /// @param token The token address to check
    /// @param serialNumber The NFT serialNumber to check
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    /// @return nonFungibleTokenInfo NonFungibleTokenInfo info for `token` `serialNumber`
    function getNonFungibleTokenInfo(address token, int64 serialNumber)
        external
        returns (int64 responseCode, NonFungibleTokenInfo memory nonFungibleTokenInfo);

    /// Operation to freeze token account
    /// @param token The token address
    /// @param account The account address to be frozen
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function freezeToken(address token, address account)
        external
        returns (int64 responseCode);

    /// Operation to unfreeze token account
    /// @param token The token address
    /// @param account The account address to be unfrozen
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function unfreezeToken(address token, address account)
        external
        returns (int64 responseCode);

    /// Operation to grant kyc to token account
    /// @param token The token address
    /// @param account The account address to grant kyc
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function grantTokenKyc(address token, address account)
        external
        returns (int64 responseCode);

    /// Operation to revoke kyc to token account
    /// @param token The token address
    /// @param account The account address to revoke kyc
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function revokeTokenKyc(address token, address account)
        external
        returns (int64 responseCode);

    /// Operation to pause token
    /// @param token The token address to be paused
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function pauseToken(address token) external returns (int64 responseCode);

    /// Operation to unpause token
    /// @param token The token address to be unpaused
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function unpauseToken(address token) external returns (int64 responseCode);

    /// Operation to wipe fungible tokens from account
    /// @param token The token address
    /// @param account The account address to revoke kyc
    /// @param amount The number of tokens to wipe
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function wipeTokenAccount(
        address token,
        address account,
        uint32 amount
    ) external returns (int64 responseCode);

    /// Operation to wipe non fungible tokens from account
    /// @param token The token address
    /// @param account The account address to revoke kyc
    /// @param  serialNumbers The serial numbers of token to wipe
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function wipeTokenAccountNFT(
        address token,
        address account,
        int64[] memory serialNumbers
    ) external returns (int64 responseCode);

    /// Operation to update token info
    /// @param token The token address
    /// @param tokenInfo The hedera token info to update token with
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function updateTokenInfo(address token, HederaToken memory tokenInfo)
        external
        returns (int64 responseCode);

    /// Operation to update token expiry info
    /// @param token The token address
    /// @param expiryInfo The hedera token expiry info
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function updateTokenExpiryInfo(address token, Expiry memory expiryInfo)
        external
        returns (int64 responseCode);

    /// Operation to update token expiry info
    /// @param token The token address
    /// @param keys The token keys
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.
    function updateTokenKeys(address token, TokenKey[] memory keys)
        external
        returns (int64 responseCode);

    /// Query if valid token found for the given address
    /// @param token The token address
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.    
    /// @return isToken True if valid token found for the given address     
    function isToken(address token) 
        external returns 
        (int64 responseCode, bool isToken);

    /// Query to return the token type for a given address
    /// @param token The token address
    /// @return responseCode The response code for the status of the request. SUCCESS is 22.    
    /// @return tokenType the token type. 0 is FUNGIBLE_COMMON, 1 is NON_FUNGIBLE_UNIQUE, -1 is UNRECOGNIZED   
    function getTokenType(address token)
        external returns 
        (int64 responseCode, int32 tokenType);
}
