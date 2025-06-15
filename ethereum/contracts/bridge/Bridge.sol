// contracts/Bridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";

import "../libraries/external/BytesLib.sol";

import "./BridgeGetters.sol";
import "./BridgeSetters.sol";
import "./BridgeStructs.sol";
import "./BridgeGovernance.sol";

import "./token/Token.sol";
import "./token/TokenImplementation.sol";

contract Bridge is BridgeGovernance, ReentrancyGuard {
    using BytesLib for bytes;

    /**
     * @notice Emitted when a transfer is completed by the token bridge.
     * @param emitterChainId Wormhole chain ID of emitter on the source chain.
     * @param emitterAddress Address (bytes32 zero-left-padded) of emitter on the source chain.
     * @param sequence Sequence of the Wormhole message.
     */
    event TransferRedeemed(
        uint16 indexed emitterChainId,
        bytes32 indexed emitterAddress,
        uint64 indexed sequence
    );

    /**
    * @notice Attests to a token by producing an `AssetMeta` message and publishing it.
    * @dev This function gathers metadata for a given ERC20 token and generates an `AssetMeta` message. 
    *      It then publishes the message to the Wormhole network. The function supports tokens that do not 
    *      implement the optional ERC20 methods for symbol and name by handling missing values gracefully.
    * @param tokenAddress The address of the ERC20 token to attest.
    * @param nonce A unique identifier for the message to prevent replay attacks.
    * @return sequence The sequence number of the published message, used to track the message on the Wormhole network.
    *
    * The `AssetMeta` message includes:
    * - `payloadID`: The identifier for the asset meta payload (set to 2).
    * - `tokenAddress`: The address of the ERC20 token, encoded as a 32-byte value.
    * - `tokenChain`: The chain ID where the token resides.
    * - `decimals`: The number of decimals used by the token.
    * - `symbol`: The symbol of the token, encoded as a 32-byte value.
    * - `name`: The name of the token, encoded as a 32-byte value.
    *
    * Emits:
    * - A message on the Wormhole network with the encoded `AssetMeta` data.
    * 
    * Requirements:
    * - The token must be a valid ERC20 token address.
    * - The function must be called with enough Ether to cover the Wormhole fee.
    *
    * Reverts:
    * - The function may revert if the token contract does not support the `decimals`, `symbol`, or `name` methods, 
    *   or if the Wormhole message publication fails.
    */
    function attestToken(address tokenAddress, uint32 nonce) public payable returns (uint64 sequence) {
        // decimals, symbol & token are not part of the core ERC20 token standard, so we need to support contracts that dont implement them
        (,bytes memory queriedDecimals) = tokenAddress.staticcall(abi.encodeWithSignature("decimals()"));
        (,bytes memory queriedSymbol) = tokenAddress.staticcall(abi.encodeWithSignature("symbol()"));
        (,bytes memory queriedName) = tokenAddress.staticcall(abi.encodeWithSignature("name()"));

        uint8 decimals = abi.decode(queriedDecimals, (uint8));

        string memory symbolString = abi.decode(queriedSymbol, (string));
        string memory nameString = abi.decode(queriedName, (string));

        bytes32 symbol;
        bytes32 name;
        assembly {
            // first 32 bytes hold string length
            symbol := mload(add(symbolString, 32))
            name := mload(add(nameString, 32))
        }

        BridgeStructs.AssetMeta memory meta = BridgeStructs.AssetMeta({
        payloadID : 2,
        tokenAddress : bytes32(uint256(uint160(tokenAddress))), // Address of the token. Left-zero-padded if shorter than 32 bytes
        tokenChain : chainId(), // Chain ID of the token
        decimals : decimals, // Number of decimals of the token (big-endian uint8)
        symbol : symbol, // Symbol of the token (UTF-8)
        name : name // Name of the token (UTF-8)
        });

        bytes memory encoded = encodeAssetMeta(meta);

        sequence = wormhole().publishMessage{
            value : msg.value
        }(nonce, encoded, finality());
    }

   /**
    * @notice Sends ETH through the portal by first wrapping it into WETH.
    * @dev This function wraps the ETH sent with the transaction into WETH, deducts the Wormhole fee and arbiter fee, and then logs the transfer details. 
    *      The function handles the wrapping, fee calculation, and transfer process, and returns the sequence number of the logged transfer.
    * @param recipientChain The chain ID where the recipient is located.
    * @param recipient The address of the recipient on the target chain, encoded as a 32-byte value.
    * @param arbiterFee The fee paid to the arbiter for processing the transfer.
    * @param nonce A unique identifier for the transfer to prevent replay attacks.
    * @return sequence The sequence number of the logged transfer, used to track the transfer on the Wormhole network.
    *
    * The function performs the following steps:
    * - Calls `_wrapAndTransferETH` to wrap the ETH into WETH, deducts fees, and refunds any excess ETH (dust) to the sender.
    * - Logs the transfer details using `logTransfer`, which records the transaction on the Wormhole network.
    *
    * Emits:
    * - A log entry with the transfer details including the wrapped WETH amount, arbiter fee, and Wormhole fee.
    *
    * Requirements:
    * - The function must be called with enough ETH to cover the Wormhole fee and arbiter fee.
    * - The wrapped amount must be sufficient after deducting fees.
    *
    * Reverts:
    * - The function may revert if the internal `_wrapAndTransferETH` call fails due to insufficient ETH or other conditions.
    */
    function wrapAndTransferETH(
        uint16 recipientChain,
        bytes32 recipient,
        uint256 arbiterFee,
        uint32 nonce
    ) public payable returns (uint64 sequence) {
        BridgeStructs.TransferResult
            memory transferResult = _wrapAndTransferETH(arbiterFee);
        sequence = logTransfer(
            transferResult.tokenChain,
            transferResult.tokenAddress,
            transferResult.normalizedAmount,
            recipientChain,
            recipient,
            transferResult.normalizedArbiterFee,
            transferResult.wormholeFee,
            nonce
        );
    }

    /**
     *  @notice Send eth through portal by first wrapping it.
     *
     *  @dev This type of transfer is called a "contract-controlled transfer".
     *  There are three differences from a regular token transfer:
     *  1) Additional arbitrary payload can be attached to the message
     *  2) Only the recipient (typically a contract) can redeem the transaction
     *  3) The sender's address (msg.sender) is also included in the transaction payload
     *
     *  With these three additional components, xDapps can implement cross-chain
     *  composable interactions.
     */
    function wrapAndTransferETHWithPayload(
        uint16 recipientChain,
        bytes32 recipient,
        uint32 nonce,
        bytes memory payload
    ) public payable returns (uint64 sequence) {
        BridgeStructs.TransferResult
            memory transferResult = _wrapAndTransferETH(0);
        sequence = logTransferWithPayload(
            transferResult.tokenChain,
            transferResult.tokenAddress,
            transferResult.normalizedAmount,
            recipientChain,
            recipient,
            transferResult.wormholeFee,
            nonce,
            payload
        );
    }
    /**
    * @notice Wraps ETH into WETH and handles the transfer process.
    * @dev This internal function manages the wrapping of ETH into WETH, deducts the Wormhole fee, and ensures that any excess ETH (dust) is refunded to the sender. 
    *      It also normalizes the amounts for the transfer and arbiter fee, and tracks the token amounts using `bridgeOut`.
    * @param arbiterFee The fee paid to the arbiter for processing the transfer.
    * @return transferResult The result of the transfer, including the token chain, token address, normalized amount, normalized arbiter fee, and Wormhole fee.
    *
    * The `transferResult` structure includes:
    * - `tokenChain`: The chain ID where the WETH resides (the current chain).
    * - `tokenAddress`: The address of the WETH contract, encoded as a 32-byte value.
    * - `normalizedAmount`: The amount of WETH, normalized for decimal places.
    * - `normalizedArbiterFee`: The arbiter fee, normalized for decimal places.
    * - `wormholeFee`: The fee paid for using the Wormhole, which is deducted from the total value.
    *
    * Requirements:
    * - The function requires that the total value sent (`msg.value`) is greater than the Wormhole fee.
    * - The arbiter fee must be less than or equal to the amount of ETH available after deducting the Wormhole fee.
    *
    * Emits:
    * - WETH deposits the ETH into the WETH contract.
    * - Excess ETH (dust) is refunded to the sender.
    *
    * Reverts:
    * - If the total value sent is smaller than the Wormhole fee, the transaction will revert with "value is smaller than wormhole fee".
    * - If the arbiter fee exceeds the amount available after the Wormhole fee is deducted, the transaction will revert with "fee is bigger than amount minus wormhole fee".
    */
    function _wrapAndTransferETH(uint256 arbiterFee) internal returns (BridgeStructs.TransferResult memory transferResult) {
        uint wormholeFee = wormhole().messageFee();

        require(wormholeFee < msg.value, "value is smaller than wormhole fee");

        uint amount = msg.value - wormholeFee;

        require(arbiterFee <= amount, "fee is bigger than amount minus wormhole fee");

        uint normalizedAmount = normalizeAmount(amount, 18);
        uint normalizedArbiterFee = normalizeAmount(arbiterFee, 18);

        // refund dust
        uint dust = amount - deNormalizeAmount(normalizedAmount, 18);
        if (dust > 0) {
            payable(msg.sender).transfer(dust);
        }

        // deposit into WETH
        WETH().deposit{
            value : amount - dust
        }();

        // track and check outstanding token amounts
        bridgeOut(address(WETH()), normalizedAmount);

        transferResult = BridgeStructs.TransferResult({
            tokenChain : chainId(),
            tokenAddress : bytes32(uint256(uint160(address(WETH())))),
            normalizedAmount : normalizedAmount,
            normalizedArbiterFee : normalizedArbiterFee,
            wormholeFee : wormholeFee
        });
    }

    /**
    * @notice Transfers ERC20 tokens through the portal.
    * @dev This function facilitates the transfer of ERC20 tokens across chains using the portal.
    *      It involves calling an internal function to handle the token transfer and then logging the transfer details.
    * @param token The address of the ERC20 token to transfer.
    * @param amount The amount of tokens to transfer.
    * @param recipientChain The chain ID of the recipient's chain.
    * @param recipient The address of the recipient on the target chain.
    * @param arbiterFee The fee paid to the arbiter for processing the transfer.
    * @param nonce A unique identifier for the transfer to prevent replay attacks.
    * @return sequence The sequence number of the logged transfer, used to track the transfer on the target chain.
    *
    * Emits:
    * - Transfer details are logged using `logTransfer`.
    * 
    * Reverts:
    * - The function will revert if the internal `_transferTokens` function fails or if the transfer logging fails.
    */
    function transferTokens(
        address token,
        uint256 amount,
        uint16 recipientChain,
        bytes32 recipient,
        uint256 arbiterFee,
        uint32 nonce
    ) public payable nonReentrant returns (uint64 sequence) {
        BridgeStructs.TransferResult memory transferResult = _transferTokens(
            token,
            amount,
            arbiterFee
        );
        sequence = logTransfer(
            transferResult.tokenChain,
            transferResult.tokenAddress,
            transferResult.normalizedAmount,
            recipientChain,
            recipient,
            transferResult.normalizedArbiterFee,
            transferResult.wormholeFee,
            nonce
        );
    }

    /**
     *  @notice Send ERC20 token through portal.
     *
     *  @dev This type of transfer is called a "contract-controlled transfer".
     *  There are three differences from a regular token transfer:
     *  1) Additional arbitrary payload can be attached to the message
     *  2) Only the recipient (typically a contract) can redeem the transaction
     *  3) The sender's address (msg.sender) is also included in the transaction payload
     *
     *  With these three additional components, xDapps can implement cross-chain
     *  composable interactions.
     */
    function transferTokensWithPayload(
        address token,
        uint256 amount,
        uint16 recipientChain,
        bytes32 recipient,
        uint32 nonce,
        bytes memory payload
    ) public payable nonReentrant returns (uint64 sequence) {
        BridgeStructs.TransferResult memory transferResult = _transferTokens(
            token,
            amount,
            0
        );
        sequence = logTransferWithPayload(
            transferResult.tokenChain,
            transferResult.tokenAddress,
            transferResult.normalizedAmount,
            recipientChain,
            recipient,
            transferResult.wormholeFee,
            nonce,
            payload
        );
    }

    /**
    * @notice Initiates a token transfer.
    * @dev This internal function handles the transfer of ERC20 tokens, including the processing of tokens on the current chain and burning tokens on foreign chains.
    *      It normalizes the amount of tokens and arbiter fees based on the token's decimal places, and ensures that dust amounts that cannot be bridged are not deposited.
    * @param token The address of the ERC20 token to transfer.
    * @param amount The amount of tokens to transfer.
    * @param arbiterFee The fee paid to the arbiter for processing the transfer.
    * @return transferResult The result of the transfer, including the token chain, token address, normalized amount, normalized arbiter fee, and wormhole fee.
    *
    * The `transferResult` structure includes:
    * - `tokenChain`: The chain ID where the token resides.
    * - `tokenAddress`: The address of the token contract.
    * - `normalizedAmount`: The amount of tokens, normalized for decimal places.
    * - `normalizedArbiterFee`: The arbiter fee, normalized for decimal places.
    * - `wormholeFee`: The fee paid for using the wormhole, which is equal to the `msg.value`.
    *
    * Requirements:
    * - The function assumes that `msg.sender` has authorized the contract to transfer tokens on their behalf.
    * - The function assumes that the `TokenImplementation` contract is properly set up and can handle the `burn` operation.
    *
    * Reverts:
    * - The function will revert if the token transfer fails or if the balance check operations do not behave as expected.
    */
    function _transferTokens(address token, uint256 amount, uint256 arbiterFee) internal returns (BridgeStructs.TransferResult memory transferResult) {
        // determine token parameters
        uint16 tokenChain;
        bytes32 tokenAddress;
        if (isWrappedAsset(token)) {
            tokenChain = TokenImplementation(token).chainId();
            tokenAddress = TokenImplementation(token).nativeContract();
        } else {
            tokenChain = chainId();
            tokenAddress = bytes32(uint256(uint160(token)));
        }

        // query tokens decimals
        (,bytes memory queriedDecimals) = token.staticcall(abi.encodeWithSignature("decimals()"));
        uint8 decimals = abi.decode(queriedDecimals, (uint8));

        // don't deposit dust that can not be bridged due to the decimal shift
        amount = deNormalizeAmount(normalizeAmount(amount, decimals), decimals);

        if (tokenChain == chainId()) {
            // query own token balance before transfer
            (,bytes memory queriedBalanceBefore) = token.staticcall(abi.encodeWithSelector(IERC20.balanceOf.selector, address(this)));
            uint256 balanceBefore = abi.decode(queriedBalanceBefore, (uint256));

            // transfer tokens
            SafeERC20.safeTransferFrom(IERC20(token), msg.sender, address(this), amount);

            // query own token balance after transfer
            (,bytes memory queriedBalanceAfter) = token.staticcall(abi.encodeWithSelector(IERC20.balanceOf.selector, address(this)));
            uint256 balanceAfter = abi.decode(queriedBalanceAfter, (uint256));

            // correct amount for potential transfer fees
            amount = balanceAfter - balanceBefore;
        } else {
            SafeERC20.safeTransferFrom(IERC20(token), msg.sender, address(this), amount);

            TokenImplementation(token).burn(address(this), amount);
        }

        // normalize amounts decimals
        uint256 normalizedAmount = normalizeAmount(amount, decimals);
        uint256 normalizedArbiterFee = normalizeAmount(arbiterFee, decimals);

        // track and check outstanding token amounts
        if (tokenChain == chainId()) {
            bridgeOut(token, normalizedAmount);
        }

        transferResult = BridgeStructs.TransferResult({
            tokenChain : tokenChain,
            tokenAddress : tokenAddress,
            normalizedAmount : normalizedAmount,
            normalizedArbiterFee : normalizedArbiterFee,
            wormholeFee : msg.value
        });
    }

    function normalizeAmount(uint256 amount, uint8 decimals) internal pure returns(uint256){
        if (decimals > 8) {
            amount /= 10 ** (decimals - 8);
        }
        return amount;
    }

    function deNormalizeAmount(uint256 amount, uint8 decimals) internal pure returns(uint256){
        if (decimals > 8) {
            amount *= 10 ** (decimals - 8);
        }
        return amount;
    }

    function logTransfer(
        uint16 tokenChain,
        bytes32 tokenAddress,
        uint256 amount,
        uint16 recipientChain,
        bytes32 recipient,
        uint256 fee,
        uint256 callValue,
        uint32 nonce
    ) internal returns (uint64 sequence) {
        require(fee <= amount, "fee exceeds amount");

        BridgeStructs.Transfer memory transfer = BridgeStructs.Transfer({
            payloadID: 1,
            amount: amount,
            tokenAddress: tokenAddress,
            tokenChain: tokenChain,
            to: recipient,
            toChain: recipientChain,
            fee: fee
        });

        sequence = wormhole().publishMessage{value: callValue}(
            nonce,
            encodeTransfer(transfer),
            finality()
        );
    }

    /*
     * @dev Publish a token transfer message with payload.
     *
     * @return The sequence number of the published message.
     */
    function logTransferWithPayload(
       -
    ) internal returns (uint64 sequence) {
        BridgeStructs.TransferWithPayload memory transfer = BridgeStructs
            .TransferWithPayload({
                payloadID: 3,
                amount: amount,
                tokenAddress: tokenAddress,
                tokenChain: tokenChain,
                to: recipient,
                toChain: recipientChain,
                fromAddress : bytes32(uint256(uint160(msg.sender))),
                payload: payload
            });

        sequence = wormhole().publishMessage{value: callValue}(
            nonce,
            encodeTransferWithPayload(transfer),
            finality()
        );
    }
    /**
     * @notice Updates the wrapped token by processing the encoded VAA (Verified Action Approval) from the Wormhole bridge.
     * @dev This function parses and verifies the encoded virtual machine (VM) from the Wormhole bridge.
     *      It ensures the validity of the VM and checks that the VM is from a legitimate emitter.
     *      Finally, it parses the asset metadata and updates the wrapped token.
     * @param encodedVm The encoded virtual machine (VM) data from the Wormhole bridge.
     * @return token The address of the updated wrapped token.
     *
     * Requirements:
     * - The `encodedVm` must be a valid Wormhole VM.
     * - The VM must be from a verified emitter (bridge).
     * - The asset metadata must be successfully parsed from the VM's payload.
     * - The `_updateWrapped` function will handle the actual update of the wrapped token.
     *
     * Reverts:
     * - If the VM is invalid or the emitter is not verified, the transaction will revert with the respective error.
     */
    function updateWrapped(bytes memory encodedVm) external returns (address token) {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVm);

        require(valid, reason);
        require(verifyBridgeVM(vm), "invalid emitter");

        BridgeStructs.AssetMeta memory meta = parseAssetMeta(vm.payload);
        return _updateWrapped(meta, vm.sequence);
    }

    function _updateWrapped(BridgeStructs.AssetMeta memory meta, uint64 sequence) internal returns (address token) {
        address wrapped = wrappedAsset(meta.tokenChain, meta.tokenAddress);
        require(wrapped != address(0), "wrapped asset does not exists");

        // Update metadata
        TokenImplementation(wrapped).updateDetails(bytes32ToString(meta.name), bytes32ToString(meta.symbol), sequence);

        return wrapped;
    }
    /**
    * @notice Creates a new wrapped token by processing the encoded VAA (Verified Action Approval) from the Wormhole bridge.
    * @dev This function parses and verifies the encoded virtual machine (VM) from the Wormhole bridge.
    *      It ensures the validity of the VM and checks that the VM is from a legitimate emitter.
    *      After parsing the asset metadata, it creates a new wrapped token.
    * @param encodedVm The encoded virtual machine (VM) data from the Wormhole bridge.
    * @return token The address of the newly created wrapped token.
    *
    * Requirements:
    * - The `encodedVm` must be a valid Wormhole VM.
    * - The VM must be from a verified emitter (bridge).
    * - The asset metadata must be successfully parsed from the VM's payload.
    * - The `_createWrapped` function will handle the actual creation of the wrapped token.
    *
    * Reverts:
    * - If the VM is invalid or the emitter is not verified, the transaction will revert with the respective error.
    */
    function createWrapped(bytes memory encodedVm) external returns (address token) {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVm);

        require(valid, reason);
        require(verifyBridgeVM(vm), "invalid emitter");

        BridgeStructs.AssetMeta memory meta = parseAssetMeta(vm.payload);
        return _createWrapped(meta, vm.sequence);
    }

    /**
    * @notice Creates a new wrapped asset using the provided `AssetMeta` information.
    * @dev This function deploys a new `BridgeToken` contract as a wrapped version of the token specified by the `AssetMeta`.
    *      It requires that the token to be wrapped is from a foreign chain (i.e., not the current chain).
    *      The function uses `create2` to deploy the `BridgeToken` contract with a unique address determined by the token's chain ID and address.
    *      It also initializes the `BridgeToken` contract with relevant parameters.
    * @param meta The metadata for the asset to be wrapped, represented by the `BridgeStructs.AssetMeta` structure.
    * @param sequence The sequence number used to initialize the `BridgeToken`.
    * @return token The address of the newly created wrapped asset.
    *
    * Requirements:
    * - The `tokenChain` in `meta` must not match the current chain ID.
    * - A wrapped asset must not already exist for the given `tokenChain` and `tokenAddress`.
    *
    * Reverts:
    * - If the token chain is the same as the current chain ID, the transaction will revert with "can only wrap tokens from foreign chains".
    * - If a wrapped asset already exists for the specified token, the transaction will revert with "wrapped asset already exists".
    * - If the deployment of the `BridgeToken` contract fails, the transaction will revert with no data.
    */
    function _createWrapped(BridgeStructs.AssetMeta memory meta, uint64 sequence) internal returns (address token) {
        require(meta.tokenChain != chainId(), "can only wrap tokens from foreign chains");
        require(wrappedAsset(meta.tokenChain, meta.tokenAddress) == address(0), "wrapped asset already exists");

        // initialize the TokenImplementation
        bytes memory initialisationArgs = abi.encodeWithSelector(
            TokenImplementation.initialize.selector,
            bytes32ToString(meta.name),
            bytes32ToString(meta.symbol),
            meta.decimals,
            sequence,

            address(this),

            meta.tokenChain,
            meta.tokenAddress
        );

        // initialize the BeaconProxy
        bytes memory constructorArgs = abi.encode(address(this), initialisationArgs);

        // deployment code
        bytes memory bytecode = abi.encodePacked(type(BridgeToken).creationCode, constructorArgs);

        bytes32 salt = keccak256(abi.encodePacked(meta.tokenChain, meta.tokenAddress));

        assembly {
            token := create2(0, add(bytecode, 0x20), mload(bytecode), salt)

            if iszero(extcodesize(token)) {
                revert(0, 0)
            }
        }

        setWrappedAsset(meta.tokenChain, meta.tokenAddress, token);
    }

    /**
     * @notice Complete a contract-controlled transfer of an ERC20 token.
     *
     * @dev The transaction can only be redeemed by the recipient, typically a
     * contract.
     *
     * @param encodedVm    A byte array containing a VAA signed by the guardians.
     *
     * @return The byte array representing a BridgeStructs.TransferWithPayload.
     */
    function completeTransferWithPayload(bytes memory encodedVm) public returns (bytes memory) {
        return _completeTransfer(encodedVm, false);
    }

    /**
     * @notice Complete a contract-controlled transfer of WETH, and unwrap to ETH.
     *
     * @dev The transaction can only be redeemed by the recipient, typically a
     * contract.
     *
     * @param encodedVm    A byte array containing a VAA signed by the guardians.
     *
     * @return The byte array representing a BridgeStructs.TransferWithPayload.
     */
    function completeTransferAndUnwrapETHWithPayload(bytes memory encodedVm) public returns (bytes memory) {
        return _completeTransfer(encodedVm, true);
    }

    /**
     * @notice Complete a transfer of an ERC20 token.
     *
     * @dev The msg.sender gets paid the associated fee.
     *
     * @param encodedVm A byte array containing a VAA signed by the guardians.
     */
    function completeTransfer(bytes memory encodedVm) public {
        _completeTransfer(encodedVm, false);
    }

    /**
     * @notice Complete a transfer of WETH and unwrap to eth.
     *
     * @dev The msg.sender gets paid the associated fee.
     *
     * @param encodedVm A byte array containing a VAA signed by the guardians.
     */
    function completeTransferAndUnwrapETH(bytes memory encodedVm) public {
        _completeTransfer(encodedVm, true);
    }

    /**
     * @dev Truncate a 32 byte array to a 20 byte address.
     *      Reverts if the array contains non-0 bytes in the first 12 bytes.
     *
     * @param bytes32 bytes The 32 byte array to be converted.
     */
    function _truncateAddress(bytes32 b) internal pure returns (address) {
        require(bytes12(b) == 0, "invalid EVM address");
        return address(uint160(uint256(b)));
    }

    /**
    * @notice Executes a transfer message, handling the transfer of tokens or ETH as specified by the Wormhole message.
    * @dev This internal function parses and verifies the Wormhole message, processes the transfer by either 
    *      unwrapping WETH or transferring ERC20 tokens, and manages the associated fees. It also ensures 
    *      that the transfer is completed only once and emits a `TransferRedeemed` event.
    * @param encodedVm The encoded Wormhole message to be processed.
    * @param unwrapWETH A boolean indicating whether to unwrap WETH into ETH during the transfer.
    * @return The payload of the Wormhole message.
    *
    * The function performs the following steps:
    * - Parses and verifies the Wormhole message to ensure it is valid and from an authorized emitter.
    * - Determines whether the transfer is of type 3 (with payload) and verifies the sender if so.
    * - Checks if the transfer has already been completed to prevent double processing.
    * - Emits a `TransferRedeemed` event to signal that the transfer has been processed.
    * - Determines the appropriate token contract and checks if WETH needs to be unwrapped.
    * - Queries the token's decimals and adjusts the amounts based on the token's decimal precision.
    * - Transfers the arbiter fee to the arbiter if applicable, either unwrapping WETH or transferring ERC20 tokens.
    * - Transfers the remaining amount to the recipient, handling both WETH and ERC20 tokens.
    *
    * Requirements:
    * - The Wormhole message must be valid and from an authorized emitter.
    * - The transfer must not have been completed previously.
    * - If `unwrapWETH` is true, the token must be WETH.
    * - The arbiter fee must be less than or equal to the transferred amount.
    * - If `unwrapWETH` is true, sufficient ETH must be available to withdraw and transfer.
    *
    * Emits:
    * - `TransferRedeemed` event indicating that the transfer has been processed.
    *
    * Reverts:
    * - If the Wormhole message is invalid or from an unauthorized emitter.
    * - If the transfer has already been completed.
    * - If the arbiter fee is greater than the transferred amount.
    * - If `unwrapWETH` is true and the token is not WETH.
    */
    function _completeTransfer(bytes memory encodedVm, bool unwrapWETH) internal returns (bytes memory) {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVm);

        require(valid, reason);
        require(verifyBridgeVM(vm), "invalid emitter");

        BridgeStructs.Transfer memory transfer = _parseTransferCommon(vm.payload);

        // payload 3 must be redeemed by the designated proxy contract
        address transferRecipient = _truncateAddress(transfer.to);
        if (transfer.payloadID == 3) {
            require(msg.sender == transferRecipient, "invalid sender");
        }

        require(!isTransferCompleted(vm.hash), "transfer already completed");
        setTransferCompleted(vm.hash);

        // emit `TransferRedeemed` event
        emit TransferRedeemed(vm.emitterChainId, vm.emitterAddress, vm.sequence);

        require(transfer.toChain == chainId(), "invalid target chain");

        IERC20 transferToken;
        if (transfer.tokenChain == chainId()) {
            transferToken = IERC20(_truncateAddress(transfer.tokenAddress));

            // track outstanding token amounts
            bridgedIn(address(transferToken), transfer.amount);
        } else {
            address wrapped = wrappedAsset(transfer.tokenChain, transfer.tokenAddress);
            require(wrapped != address(0), "no wrapper for this token created yet");

            transferToken = IERC20(wrapped);
        }

        require(unwrapWETH == false || address(transferToken) == address(WETH()), "invalid token, can only unwrap WETH");

        // query decimals
        (,bytes memory queriedDecimals) = address(transferToken).staticcall(abi.encodeWithSignature("decimals()"));
        uint8 decimals = abi.decode(queriedDecimals, (uint8));

        // adjust decimals
        uint256 nativeAmount = deNormalizeAmount(transfer.amount, decimals);
        uint256 nativeFee = deNormalizeAmount(transfer.fee, decimals);

        // transfer fee to arbiter
        if (nativeFee > 0 && transferRecipient != msg.sender) {
            require(nativeFee <= nativeAmount, "fee higher than transferred amount");

            if (unwrapWETH) {
                WETH().withdraw(nativeFee);

                payable(msg.sender).transfer(nativeFee);
            } else {
                if (transfer.tokenChain != chainId()) {
                    // mint wrapped asset
                    TokenImplementation(address(transferToken)).mint(msg.sender, nativeFee);
                } else {
                    SafeERC20.safeTransfer(transferToken, msg.sender, nativeFee);
                }
            }
        } else {
            // set fee to zero in case transferRecipient == feeRecipient
            nativeFee = 0;
        }

        // transfer bridged amount to recipient
        uint transferAmount = nativeAmount - nativeFee;

        if (unwrapWETH) {
            WETH().withdraw(transferAmount);

            payable(transferRecipient).transfer(transferAmount);
        } else {
            if (transfer.tokenChain != chainId()) {
                // mint wrapped asset
                TokenImplementation(address(transferToken)).mint(transferRecipient, transferAmount);
            } else {
                SafeERC20.safeTransfer(transferToken, transferRecipient, transferAmount);
            }
        }

        return vm.payload;
    }

    function bridgeOut(address token, uint normalizedAmount) internal {
        uint outstanding = outstandingBridged(token);
        require(outstanding + normalizedAmount <= type(uint64).max, "transfer exceeds max outstanding bridged token amount");
        setOutstandingBridged(token, outstanding + normalizedAmount);
    }

    function bridgedIn(address token, uint normalizedAmount) internal {
        setOutstandingBridged(token, outstandingBridged(token) - normalizedAmount);
    }

    function verifyBridgeVM(IWormhole.VM memory vm) internal view returns (bool){
        require(!isFork(), "invalid fork");
        return bridgeContracts(vm.emitterChainId) == vm.emitterAddress;
    }

    /**
     * @notice Encodes the asset metadata into a single `bytes` object.
     * @dev This function takes the `AssetMeta` structure and encodes its fields into a `bytes` array.
     *      The fields are packed together using `abi.encodePacked`.
     * @param meta The asset metadata to encode, represented by the `BridgeStructs.AssetMeta` structure.
     * @return encoded The encoded asset metadata as a `bytes` array.
     *
     * The encoded metadata includes:
     * - `payloadID`: The ID of the payload.
     * - `tokenAddress`: The address of the token.
     * - `tokenChain`: The chain ID where the token resides.
     * - `decimals`: The number of decimals the token uses.
     * - `symbol`: The symbol of the token.
     * - `name`: The name of the token.
     */
    function encodeAssetMeta(BridgeStructs.AssetMeta memory meta) public pure returns (bytes memory encoded) {
        encoded = abi.encodePacked(
            meta.payloadID,
            meta.tokenAddress,
            meta.tokenChain,
            meta.decimals,
            meta.symbol,
            meta.name
        );
    }

    /**
    * @notice Encodes the transfer data into a single `bytes` object.
    * @dev This function takes the `Transfer` structure and encodes its fields into a `bytes` array.
    *      The fields are packed together using `abi.encodePacked`.
    * @param transfer The transfer data to encode, represented by the `BridgeStructs.Transfer` structure.
    * @return encoded The encoded transfer data as a `bytes` array.
    *
    * The encoded transfer data includes:
    * - `payloadID`: The ID of the payload.
    * - `amount`: The amount being transferred.
    * - `tokenAddress`: The address of the token being transferred.
    * - `tokenChain`: The chain ID where the token resides.
    * - `to`: The recipient address.
    * - `toChain`: The chain ID of the recipient.
    * - `fee`: The fee associated with the transfer.
    */

    function encodeTransfer(BridgeStructs.Transfer memory transfer) public pure returns (bytes memory encoded) {
        encoded = abi.encodePacked(
            transfer.payloadID,
            transfer.amount,
            transfer.tokenAddress,
            transfer.tokenChain,
            transfer.to,
            transfer.toChain,
            transfer.fee
        );
    }
    /**
     * @notice Encodes the transfer data with an additional payload into a single `bytes` object.
     * @dev This function takes the `TransferWithPayload` structure and encodes its fields into a `bytes` array.
     *      The fields are packed together using `abi.encodePacked`.
     * @param transfer The transfer data with payload to encode, represented by the `BridgeStructs.TransferWithPayload` structure.
     * @return encoded The encoded transfer data with payload as a `bytes` array.
     *
     * The encoded transfer data includes:
     * - `payloadID`: The ID of the payload.
     * - `amount`: The amount being transferred.
     * - `tokenAddress`: The address of the token being transferred.
     * - `tokenChain`: The chain ID where the token resides.
     * - `to`: The recipient address.
     * - `toChain`: The chain ID of the recipient.
     * - `fromAddress`: The address of the sender.
     * - `payload`: Additional payload data associated with the transfer.
     */
    function encodeTransferWithPayload(BridgeStructs.TransferWithPayload memory transfer) public pure returns (bytes memory encoded) {
        encoded = abi.encodePacked(
            transfer.payloadID,
            transfer.amount,
            transfer.tokenAddress,
            transfer.tokenChain,
            transfer.to,
            transfer.toChain,
            transfer.fromAddress,
            transfer.payload
        );
    }

    function parsePayloadID(bytes memory encoded) public pure returns (uint8 payloadID) {
        payloadID = encoded.toUint8(0);
    }

   /**
    * @notice Parses encoded asset metadata and returns it as an `AssetMeta` structure.
    * @dev This function decodes the provided `bytes` array into an `AssetMeta` structure.
    *      The function sequentially extracts the fields from the encoded data and verifies the integrity of the data.
    * @param encoded The encoded asset metadata as a `bytes` array.
    * @return meta The parsed asset metadata represented by the `BridgeStructs.AssetMeta` structure.
    *
    * The decoded asset metadata includes:
    * - `payloadID`: The ID of the payload (must be 2 for valid `AssetMeta`).
    * - `tokenAddress`: The address of the token.
    * - `tokenChain`: The chain ID where the token resides.
    * - `decimals`: The number of decimals the token uses.
    * - `symbol`: The symbol of the token.
    * - `name`: The name of the token.
    *
    * Requirements:
    * - The `payloadID` must be 2, indicating valid `AssetMeta` data.
    * - The length of the encoded data must match the expected length after parsing.
    *
    * Reverts:
    * - If the `payloadID` is not 2, the transaction will revert with "invalid AssetMeta".
    * - If the length of the encoded data does not match the expected length after parsing, the transaction will revert with "invalid AssetMeta".
    */
    function parseAssetMeta(bytes memory encoded) public pure returns (BridgeStructs.AssetMeta memory meta) {
        uint index = 0;

        meta.payloadID = encoded.toUint8(index);
        index += 1;

        require(meta.payloadID == 2, "invalid AssetMeta");

        meta.tokenAddress = encoded.toBytes32(index);
        index += 32;

        meta.tokenChain = encoded.toUint16(index);
        index += 2;

        meta.decimals = encoded.toUint8(index);
        index += 1;

        meta.symbol = encoded.toBytes32(index);
        index += 32;

        meta.name = encoded.toBytes32(index);
        index += 32;

        require(encoded.length == index, "invalid AssetMeta");
    }

    /**
    * @notice Parses encoded transfer data and returns it as a `Transfer` structure.
    * @dev This function decodes the provided `bytes` array into a `Transfer` structure.
    *      The function sequentially extracts the fields from the encoded data and verifies the integrity of the data.
    * @param encoded The encoded transfer data as a `bytes` array.
    * @return transfer The parsed transfer data represented by the `BridgeStructs.Transfer` structure.
    *
    * The decoded transfer data includes:
    * - `payloadID`: The ID of the payload (must be 1 for valid `Transfer` data).
    * - `amount`: The amount being transferred.
    * - `tokenAddress`: The address of the token being transferred.
    * - `tokenChain`: The chain ID where the token resides.
    * - `to`: The recipient address.
    * - `toChain`: The chain ID of the recipient.
    * - `fee`: The fee associated with the transfer.
    *
    * Requirements:
    * - The `payloadID` must be 1, indicating valid `Transfer` data.
    * - The length of the encoded data must match the expected length after parsing.
    *
    * Reverts:
    * - If the `payloadID` is not 1, the transaction will revert with "invalid Transfer".
    * - If the length of the encoded data does not match the expected length after parsing, the transaction will revert with "invalid Transfer".
    */
    function parseTransfer(bytes memory encoded) public pure returns (BridgeStructs.Transfer memory transfer) {
        uint index = 0;

        transfer.payloadID = encoded.toUint8(index);
        index += 1;

        require(transfer.payloadID == 1, "invalid Transfer");

        transfer.amount = encoded.toUint256(index);
        index += 32;

        transfer.tokenAddress = encoded.toBytes32(index);
        index += 32;

        transfer.tokenChain = encoded.toUint16(index);
        index += 2;

        transfer.to = encoded.toBytes32(index);
        index += 32;

        transfer.toChain = encoded.toUint16(index);
        index += 2;

        transfer.fee = encoded.toUint256(index);
        index += 32;

        require(encoded.length == index, "invalid Transfer");
    }

    /**
    * @notice Parses encoded transfer data with an additional payload and returns it as a `TransferWithPayload` structure.
    * @dev This function decodes the provided `bytes` array into a `TransferWithPayload` structure.
    *      The function sequentially extracts the fields from the encoded data and verifies the integrity of the data.
    * @param encoded The encoded transfer data with payload as a `bytes` array.
    * @return transfer The parsed transfer data with payload, represented by the `BridgeStructs.TransferWithPayload` structure.
    *
    * The decoded transfer data includes:
    * - `payloadID`: The ID of the payload (must be 3 for valid `TransferWithPayload` data).
    * - `amount`: The amount being transferred.
    * - `tokenAddress`: The address of the token being transferred.
    * - `tokenChain`: The chain ID where the token resides.
    * - `to`: The recipient address.
    * - `toChain`: The chain ID of the recipient.
    * - `fromAddress`: The address of the sender.
    * - `payload`: Additional payload data associated with the transfer.
    *
    * Requirements:
    * - The `payloadID` must be 3, indicating valid `TransferWithPayload` data.
    *
    * Reverts:
    * - If the `payloadID` is not 3, the transaction will revert with "invalid Transfer".
    */
    function parseTransferWithPayload(bytes memory encoded) public pure returns (BridgeStructs.TransferWithPayload memory transfer) {
        uint index = 0;

        transfer.payloadID = encoded.toUint8(index);
        index += 1;

        require(transfer.payloadID == 3, "invalid Transfer");

        transfer.amount = encoded.toUint256(index);
        index += 32;

        transfer.tokenAddress = encoded.toBytes32(index);
        index += 32;

        transfer.tokenChain = encoded.toUint16(index);
        index += 2;

        transfer.to = encoded.toBytes32(index);
        index += 32;

        transfer.toChain = encoded.toUint16(index);
        index += 2;

        transfer.fromAddress = encoded.toBytes32(index);
        index += 32;

        transfer.payload = encoded.slice(index, encoded.length - index);
    }

   /**
 * @notice Parses an encoded transfer data into a `Transfer` structure.
 * @dev This function decodes either a type 1 transfer or a type 3 transfer ("transfer with payload") into a `Transfer` structure.
 *      For type 3 transfers, which have no fees, the fee is set to 0.
 *      This function helps circumvent local variable count limitations in `_completeTransfer`.
 * @param encoded The encoded transfer data as a `bytes` array.
 * @return transfer The parsed transfer data represented by the `BridgeStructs.Transfer` structure.
 *
 * The decoded transfer data includes:
 * - `payloadID`: The ID of the payload (1 for type 1 transfers, 3 for type 3 transfers).
 * - `amount`: The amount being transferred.
 * - `tokenAddress`: The address of the token being transferred.
 * - `tokenChain`: The chain ID where the token resides.
 * - `to`: The recipient address.
 * - `toChain`: The chain ID of the recipient.
 * - `fee`: The fee associated with the transfer (set to 0 for type 3 transfers).
 *
 * Requirements:
 * - The `payloadID` must be either 1 or 3, indicating valid transfer types.
 *
 * Reverts:
 * - If the `payloadID` is not 1 or 3, the transaction will revert with "Invalid payload id".
 */
    function _parseTransferCommon(bytes memory encoded) public pure returns (BridgeStructs.Transfer memory transfer) {
        uint8 payloadID = parsePayloadID(encoded);

        if (payloadID == 1) {
            transfer = parseTransfer(encoded);
        } else if (payloadID == 3) {
            BridgeStructs.TransferWithPayload memory t = parseTransferWithPayload(encoded);
            transfer.payloadID = 3;
            transfer.amount = t.amount;
            transfer.tokenAddress = t.tokenAddress;
            transfer.tokenChain = t.tokenChain;
            transfer.to = t.to;
            transfer.toChain = t.toChain;
            // Type 3 payloads don't have fees.
            transfer.fee = 0;
        } else {
            revert("Invalid payload id");
        }
    }

    function bytes32ToString(bytes32 input) internal pure returns (string memory) {
        uint256 i;
        while (i < 32 && input[i] != 0) {
            i++;
        }
        bytes memory array = new bytes(i);
        for (uint c = 0; c < i; c++) {
            array[c] = input[c];
        }
        return string(array);
    }

    // we need to accept ETH sends to unwrap WETH
    receive() external payable {}
}
