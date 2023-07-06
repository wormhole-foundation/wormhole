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

    /*
     *  @dev Produce a AssetMeta message for a given token
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

    /*
     *  @notice Send eth through portal by first wrapping it to WETH.
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

    function createWrapped(bytes memory encodedVm) external returns (address token) {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVm);

        require(valid, reason);
        require(verifyBridgeVM(vm), "invalid emitter");

        BridgeStructs.AssetMeta memory meta = parseAssetMeta(vm.payload);
        return _createWrapped(meta, vm.sequence);
    }

    // Creates a wrapped asset using AssetMeta
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
    function completeTransferWithPayload(bytes memory encodedVm) public returns (bytes memory) {
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
    function completeTransferAndUnwrapETHWithPayload(bytes memory encodedVm) public returns (bytes memory) {
        return _completeTransfer(encodedVm, true);
    }

    /*
     * @notice Complete a transfer of an ERC20 token.
     *
     * @dev The msg.sender gets paid the associated fee.
     *
     * @param encodedVm A byte array containing a VAA signed by the guardians.
     */
    function completeTransfer(bytes memory encodedVm) public {
        _completeTransfer(encodedVm, false);
    }

    /*
     * @notice Complete a transfer of WETH and unwrap to eth.
     *
     * @dev The msg.sender gets paid the associated fee.
     *
     * @param encodedVm A byte array containing a VAA signed by the guardians.
     */
    function completeTransferAndUnwrapETH(bytes memory encodedVm) public {
        _completeTransfer(encodedVm, true);
    }

    /*
     * @dev Truncate a 32 byte array to a 20 byte address.
     *      Reverts if the array contains non-0 bytes in the first 12 bytes.
     *
     * @param bytes32 bytes The 32 byte array to be converted.
     */
    function _truncateAddress(bytes32 b) internal pure returns (address) {
        require(bytes12(b) == 0, "invalid EVM address");
        return address(uint160(uint256(b)));
    }

    // Execute a Transfer message
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
