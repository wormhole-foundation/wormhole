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

    /// @notice Emitted when the bridge is temporarily paused via `pause`.
    /// @param by The pauser that paused the bridge.
    /// @param pauseExpiry The timestamp (unix seconds) at which the pause becomes permissionlessly
    ///        liftable via `unpauseExpired`.
    event Paused(address indexed by, uint256 pauseExpiry);

    /// @notice Emitted when the bridge is frozen via `freeze`.
    /// @param by The freezer that froze the bridge.
    /// @param pauseExpiry The pause expiry, set to the maximum timestamp by `freeze`.
    event Frozen(address indexed by, uint256 pauseExpiry);

    /// @notice Emitted when the bridge is unpaused via `unpause`.
    event Unpaused(address indexed by);

    /// @notice Emitted when an expired pause is lifted permissionlessly via `unpauseExpired`.
    event UnpauseExpired(address indexed by);

    /// @dev Custom errors are used in place of revert strings to keep `BridgeImplementation` under
    ///      the 24,576-byte EIP-170 limit.

    /// @notice Reverts when a `notPaused`-guarded entry point is called while the bridge is paused.
    error BridgePaused();
    /// @notice Reverts when `pause()` is called while the pauser role is unassigned, or by anyone
    ///         other than the configured pauser. The "unassigned" branch is checked first so that
    ///         an all-zero `pauser` is never treated as an authorized caller.
    error NotPauser();
    /// @notice Reverts when `unpause()` is called while the unpauser role is unassigned, or by
    ///         anyone other than the configured unpauser. The "unassigned" branch is checked first
    ///         so that an all-zero `unpauser` is never treated as an authorized caller.
    error NotUnpauser();
    /// @notice Reverts when `freeze()` is called while the freezer role is unassigned, or by anyone
    ///         other than the configured freezer. The "unassigned" branch is checked first so that
    ///         an all-zero `freezer` is never treated as an authorized caller.
    error NotFreezer();
    /// @notice Reverts when `unpause()` / `unpauseExpired()` is called while the bridge is not paused.
    error NotPaused();
    /// @notice Reverts when `unpauseExpired()` is called before the current pause has expired
    ///         (`block.timestamp < pauseExpiry`).
    error NotExpired();
    /// @notice Reverts when `pause()` would not push `pauseExpiry` forward — i.e. the bridge already
    ///         has an equal-or-later expiry (e.g. it is frozen). A lower-trust pauser must not be
    ///         able to shorten a `freeze`, and a call that changes nothing reverts rather than
    ///         emitting a misleading success.
    error PauseNotExtended();
    /// @notice Reverts when `msg.value` does not cover the wormhole message fee.
    error InsufficientFee();
    /// @notice Reverts when an arbiter / relayer fee exceeds the transfer amount.
    error FeeExceedsAmount();
    /// @notice Reverts when a transfer VAA is not from a registered token-bridge emitter.
    error InvalidEmitter();
    /// @notice Reverts when no wrapped asset is registered for a given (chain, token) pair.
    error WrappedAssetNotFound();
    /// @notice Reverts when `createWrapped` is called for a token native to this chain.
    error OnlyForeignTokens();
    /// @notice Reverts when `createWrapped` is called for a (chain, token) that already has a wrapper.
    error WrappedAssetAlreadyExists();
    /// @notice Reverts when truncating a 32-byte value to an EVM address loses non-zero high bytes.
    error InvalidEVMAddress();
    /// @notice Reverts when a payload-3 transfer is redeemed by anyone other than the recipient.
    error InvalidSender();
    /// @notice Reverts when a transfer VAA has already been redeemed.
    error TransferAlreadyCompleted();
    /// @notice Reverts when redeeming a transfer whose target chain is not this chain.
    error InvalidTargetChain();
    /// @notice Reverts when an unwrap-ETH path is taken with a token other than WETH.
    error OnlyWETH();
    /// @notice Reverts when an outbound transfer would exceed the per-token outstanding bridged cap.
    error OutstandingExceedsMax();
    /// @notice Reverts when an AssetMeta payload is malformed.
    error InvalidAssetMeta();
    /// @notice Reverts when a Transfer payload is malformed.
    error InvalidTransferPayload();
    /// @notice Reverts when an unknown payload id is encountered.
    error InvalidPayloadId();

    /// @dev Reverts if the bridge is paused. See the "Pausing" section of whitepapers/0003_token_bridge.md.
    /// @dev Implemented as a shared internal function (rather than inlining the check in the
    ///      modifier body) so the bytecode is emitted once and JUMPed to from every callsite,
    ///      keeping `BridgeImplementation` under the 24,576-byte EIP-170 limit.
    function _requireNotPaused() internal view {
        if (paused()) revert BridgePaused();
    }

    modifier notPaused() {
        _requireNotPaused();
        _;
    }

    /// @dev Temporary-pause duration: 5 days, in seconds (`block.timestamp` is seconds). A `pause`
    ///      holds the bridge for this long; the pauser must re-`pause` to extend it (a dead-man's
    ///      switch — the hold lapses if the pauser stops acting). See
    ///      whitepapers/0003_token_bridge.md.
    uint64 constant PAUSE_DURATION = 5 days;

    /// @dev Authorize `msg.sender` against a configured role address, reverting `err` (a 4-byte
    ///      custom-error selector) if the role is unassigned (all-zero) or the caller does not
    ///      match. The unassigned check is first so an all-zero role is never authorized. Shared by
    ///      pause/freeze/unpause and reverts via assembly so the three callsites collapse to one
    ///      body — keeping `BridgeImplementation` under the EIP-170 limit.
    function _requireRole(address role, bytes4 err) internal view {
        if (role == address(0) || msg.sender != role) {
            assembly {
                mstore(0x0, err)
                revert(0x0, 0x4)
            }
        }
    }

    /// @notice Temporarily pause the bridge. Only callable by the configured pauser. Pushes
    ///         `pauseExpiry` to `block.timestamp + PAUSE_DURATION` (5 days) and sets `paused`.
    ///         Reverts with `PauseNotExtended` if the new expiry would not be strictly later than
    ///         the current one — so a lower-trust pauser can never curtail a `freeze`, and a call
    ///         that would change nothing fails loudly instead of emitting a misleading success.
    ///         Not idempotent: each successful call extends the window.
    /// @dev The pauser is configured via the `SetPauserAddresses` (action 4) governance VAA and may
    ///      be left unassigned; when unassigned this entry point reverts before comparing
    ///      `msg.sender`, so an all-zero `pauser` is never authorized.
    /// @dev Intentionally does not check `isFork()`. On a forked chain the pre-fork pauser can
    ///      still pause the bridge without first waiting for a `submitRecoverChainId` governance
    ///      VAA — letting whoever holds the pauser key shut the bridge down on the fork
    ///      immediately, before chain-id recovery completes.
    function pause() external {
        _requireRole(pauser(), NotPauser.selector);
        uint64 newExpiry = uint64(block.timestamp) + PAUSE_DURATION;
        // Never reduce (or leave unchanged) an expiry already at/further out (e.g. one set by
        // `freeze`): a pause that doesn't extend the window is a no-op, so revert rather than
        // emit a misleading success.
        if (newExpiry <= pauseExpiry()) revert PauseNotExtended();
        setPauseExpiry(newExpiry);
        setPaused(true);
        emit Paused(msg.sender, newExpiry);
    }

    /// @notice Freeze the bridge for the maximum duration. Only callable by the configured freezer.
    ///         Sets `paused` to `true` and `pauseExpiry` to the maximum timestamp. The higher-trust
    ///         counterpart to the temporary, self-expiring `pause`: a frozen bridge will not become
    ///         permissionlessly unpausable in practice and can only be lifted by the `unpauser`.
    ///         Idempotent.
    /// @dev The freezer may be left unassigned; when unassigned this reverts before comparing
    ///      `msg.sender`, so an all-zero `freezer` is never authorized.
    /// @dev Intentionally does not check `isFork()`; see the matching note on `pause()`.
    function freeze() external {
        _requireRole(freezer(), NotFreezer.selector);
        setPauseExpiry(type(uint64).max);
        setPaused(true);
        emit Frozen(msg.sender, type(uint64).max);
    }

    /// @notice Unpause the bridge. Only callable by the configured unpauser. Clears `paused` and
    ///         sets `pauseExpiry` to `block.timestamp`. The privileged path to lift any pause
    ///         (including a `freeze`) early. Reverts if the unpauser is unassigned or the bridge is
    ///         not currently paused.
    /// @dev Setting `pauseExpiry` to the current time (rather than 0) leaves on-chain evidence of
    ///      the last unpause while bringing any stale `freeze` expiry down to the present, so it
    ///      cannot block a later `pause`. Exempt from `notPaused` so it remains callable while
    ///      paused. Does not check `isFork()`; see the note on `pause()`.
    function unpause() external {
        _requireRole(unpauser(), NotUnpauser.selector);
        if (!paused()) revert NotPaused();
        _clearPauseToNow();
        emit Unpaused(msg.sender);
    }

    /// @dev Shared tail of `unpause`/`unpauseExpired`: clear `paused` and bring `pauseExpiry` down
    ///      to now (so a stale `freeze` expiry cannot block a later `pause`). Emitted once and
    ///      JUMPed to from both callsites to keep `BridgeImplementation` under the EIP-170 limit.
    function _clearPauseToNow() internal {
        setPauseExpiry(uint64(block.timestamp));
        setPaused(false);
    }

    /// @notice Permissionlessly unpause the bridge once its pause has expired. Clears `paused` and
    ///         sets `pauseExpiry` to `block.timestamp`. No role required. Reverts if the bridge is
    ///         not currently paused or `block.timestamp < pauseExpiry`.
    /// @dev Bounds a `pauser`-initiated pause to `PAUSE_DURATION` without requiring the `unpauser`
    ///      to act. The boolean `paused` remains authoritative — a pause is only lifted by an
    ///      explicit `unpause`/`unpauseExpired` call, never silently by the passage of time.
    function unpauseExpired() external {
        if (!paused()) revert NotPaused();
        if (block.timestamp < pauseExpiry()) revert NotExpired();
        _clearPauseToNow();
        emit UnpauseExpired(msg.sender);
    }

    /*
     *  @dev Produce a AssetMeta message for a given token
     */
    function attestToken(address tokenAddress, uint32 nonce) public payable notPaused returns (uint64 sequence) {
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

    /*
     *  @notice Send eth through portal by first wrapping it to WETH.
     */
    function wrapAndTransferETH(
        uint16 recipientChain,
        bytes32 recipient,
        uint256 arbiterFee,
        uint32 nonce
    ) public payable notPaused returns (uint64 sequence) {
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

    /*
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
    ) public payable notPaused returns (uint64 sequence) {
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

    function _wrapAndTransferETH(uint256 arbiterFee) internal returns (BridgeStructs.TransferResult memory transferResult) {
        uint wormholeFee = wormhole().messageFee();

        if (wormholeFee >= msg.value) revert InsufficientFee();

        uint amount = msg.value - wormholeFee;

        if (arbiterFee > amount) revert FeeExceedsAmount();

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

    /*
     *  @notice Send ERC20 token through portal.
     */
    function transferTokens(
        address token,
        uint256 amount,
        uint16 recipientChain,
        bytes32 recipient,
        uint256 arbiterFee,
        uint32 nonce
    ) public payable nonReentrant notPaused returns (uint64 sequence) {
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

    /*
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
    ) public payable nonReentrant notPaused returns (uint64 sequence) {
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

    /*
     *  @notice Initiate a transfer
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
        if (fee > amount) revert FeeExceedsAmount();

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
        uint16 tokenChain,
        bytes32 tokenAddress,
        uint256 amount,
        uint16 recipientChain,
        bytes32 recipient,
        uint256 callValue,
        uint32 nonce,
        bytes memory payload
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

    function updateWrapped(bytes memory encodedVm) external notPaused returns (address token) {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVm);

        require(valid, reason);
        if (!verifyBridgeVM(vm)) revert InvalidEmitter();

        BridgeStructs.AssetMeta memory meta = parseAssetMeta(vm.payload);
        return _updateWrapped(meta, vm.sequence);
    }

    function _updateWrapped(BridgeStructs.AssetMeta memory meta, uint64 sequence) internal returns (address token) {
        address wrapped = wrappedAsset(meta.tokenChain, meta.tokenAddress);
        if (wrapped == address(0)) revert WrappedAssetNotFound();

        // Update metadata
        TokenImplementation(wrapped).updateDetails(bytes32ToString(meta.name), bytes32ToString(meta.symbol), sequence);

        return wrapped;
    }

    function createWrapped(bytes memory encodedVm) external notPaused returns (address token) {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVm);

        require(valid, reason);
        if (!verifyBridgeVM(vm)) revert InvalidEmitter();

        BridgeStructs.AssetMeta memory meta = parseAssetMeta(vm.payload);
        return _createWrapped(meta, vm.sequence);
    }

    // Creates a wrapped asset using AssetMeta
    function _createWrapped(BridgeStructs.AssetMeta memory meta, uint64 sequence) internal returns (address token) {
        if (meta.tokenChain == chainId()) revert OnlyForeignTokens();
        if (wrappedAsset(meta.tokenChain, meta.tokenAddress) != address(0)) revert WrappedAssetAlreadyExists();

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

    /*
     * @notice Complete a contract-controlled transfer of an ERC20 token.
     *
     * @dev The transaction can only be redeemed by the recipient, typically a
     * contract.
     *
     * @param encodedVm    A byte array containing a VAA signed by the guardians.
     *
     * @return The byte array representing a BridgeStructs.TransferWithPayload.
     */
    function completeTransferWithPayload(bytes memory encodedVm) public notPaused returns (bytes memory) {
        return _completeTransfer(encodedVm, false);
    }

    /*
     * @notice Complete a contract-controlled transfer of WETH, and unwrap to ETH.
     *
     * @dev The transaction can only be redeemed by the recipient, typically a
     * contract.
     *
     * @param encodedVm    A byte array containing a VAA signed by the guardians.
     *
     * @return The byte array representing a BridgeStructs.TransferWithPayload.
     */
    function completeTransferAndUnwrapETHWithPayload(bytes memory encodedVm) public notPaused returns (bytes memory) {
        return _completeTransfer(encodedVm, true);
    }

    /*
     * @notice Complete a transfer of an ERC20 token.
     *
     * @dev The msg.sender gets paid the associated fee.
     *
     * @param encodedVm A byte array containing a VAA signed by the guardians.
     */
    function completeTransfer(bytes memory encodedVm) public notPaused {
        _completeTransfer(encodedVm, false);
    }

    /*
     * @notice Complete a transfer of WETH and unwrap to eth.
     *
     * @dev The msg.sender gets paid the associated fee.
     *
     * @param encodedVm A byte array containing a VAA signed by the guardians.
     */
    function completeTransferAndUnwrapETH(bytes memory encodedVm) public notPaused {
        _completeTransfer(encodedVm, true);
    }

    /*
     * @dev Truncate a 32 byte array to a 20 byte address.
     *      Reverts if the array contains non-0 bytes in the first 12 bytes.
     *
     * @param bytes32 bytes The 32 byte array to be converted.
     */
    function _truncateAddress(bytes32 b) internal pure returns (address) {
        if (bytes12(b) != 0) revert InvalidEVMAddress();
        return address(uint160(uint256(b)));
    }

    // Execute a Transfer message
    function _completeTransfer(bytes memory encodedVm, bool unwrapWETH) internal returns (bytes memory) {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVm);

        require(valid, reason);
        if (!verifyBridgeVM(vm)) revert InvalidEmitter();

        BridgeStructs.Transfer memory transfer = _parseTransferCommon(vm.payload);

        // payload 3 must be redeemed by the designated proxy contract
        address transferRecipient = _truncateAddress(transfer.to);
        if (transfer.payloadID == 3) {
            if (msg.sender != transferRecipient) revert InvalidSender();
        }

        if (isTransferCompleted(vm.hash)) revert TransferAlreadyCompleted();
        setTransferCompleted(vm.hash);

        // emit `TransferRedeemed` event
        emit TransferRedeemed(vm.emitterChainId, vm.emitterAddress, vm.sequence);

        if (transfer.toChain != chainId()) revert InvalidTargetChain();

        IERC20 transferToken;
        if (transfer.tokenChain == chainId()) {
            transferToken = IERC20(_truncateAddress(transfer.tokenAddress));

            // track outstanding token amounts
            bridgedIn(address(transferToken), transfer.amount);
        } else {
            address wrapped = wrappedAsset(transfer.tokenChain, transfer.tokenAddress);
            if (wrapped == address(0)) revert WrappedAssetNotFound();

            transferToken = IERC20(wrapped);
        }

        if (unwrapWETH && address(transferToken) != address(WETH())) revert OnlyWETH();

        // query decimals
        (,bytes memory queriedDecimals) = address(transferToken).staticcall(abi.encodeWithSignature("decimals()"));
        uint8 decimals = abi.decode(queriedDecimals, (uint8));

        // adjust decimals
        uint256 nativeAmount = deNormalizeAmount(transfer.amount, decimals);
        uint256 nativeFee = deNormalizeAmount(transfer.fee, decimals);

        // transfer fee to arbiter
        if (nativeFee > 0 && transferRecipient != msg.sender) {
            if (nativeFee > nativeAmount) revert FeeExceedsAmount();

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
        if (outstanding + normalizedAmount > type(uint64).max) revert OutstandingExceedsMax();
        setOutstandingBridged(token, outstanding + normalizedAmount);
    }

    function bridgedIn(address token, uint normalizedAmount) internal {
        setOutstandingBridged(token, outstandingBridged(token) - normalizedAmount);
    }

    function verifyBridgeVM(IWormhole.VM memory vm) internal view returns (bool){
        if (isFork()) revert InvalidFork();
        return bridgeContracts(vm.emitterChainId) == vm.emitterAddress;
    }

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

    /*
     * @dev Parse a token metadata attestation (payload id 2)
     */
    function parseAssetMeta(bytes memory encoded) public pure returns (BridgeStructs.AssetMeta memory meta) {
        uint index = 0;

        meta.payloadID = encoded.toUint8(index);
        index += 1;

        if (meta.payloadID != 2) revert InvalidAssetMeta();

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

        if (encoded.length != index) revert InvalidAssetMeta();
    }

    /*
     * @dev Parse a token transfer (payload id 1).
     *
     * @params encoded The byte array corresponding to the token transfer (not
     *                 the whole VAA, only the payload)
     */
    function parseTransfer(bytes memory encoded) public pure returns (BridgeStructs.Transfer memory transfer) {
        uint index = 0;

        transfer.payloadID = encoded.toUint8(index);
        index += 1;

        if (transfer.payloadID != 1) revert InvalidTransferPayload();

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

        if (encoded.length != index) revert InvalidTransferPayload();
    }

    /*
     * @dev Parse a token transfer with payload (payload id 3).
     *
     * @params encoded The byte array corresponding to the token transfer (not
     *                 the whole VAA, only the payload)
     */
    function parseTransferWithPayload(bytes memory encoded) public pure returns (BridgeStructs.TransferWithPayload memory transfer) {
        uint index = 0;

        transfer.payloadID = encoded.toUint8(index);
        index += 1;

        if (transfer.payloadID != 3) revert InvalidTransferPayload();

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

    /*
     * @dev Parses either a type 1 transfer or a type 3 transfer ("transfer with
     *      payload") as a Transfer struct. The fee is set to 0 for type 3
     *      transfers, since they have no fees associated with them.
     *
     *      The sole purpose of this function is to get around the local
     *      variable count limitation in _completeTransfer.
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
            revert InvalidPayloadId();
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
