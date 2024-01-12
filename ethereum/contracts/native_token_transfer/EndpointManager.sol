// SPDX-License-Identifier: Apache 2
pragma solidity >=0.6.12 <0.9.0;

import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import "@openzeppelin/contracts/token/ERC20/utils/SafeERC20.sol";
import "@openzeppelin/contracts/token/ERC20/extensions/ERC20Burnable.sol";

import "wormhole-solidity-sdk/Utils.sol";
import "wormhole-solidity-sdk/libraries/BytesParsing.sol";

import "../libraries/external/OwnableUpgradeable.sol";
import "./libraries/EndpointStructs.sol";
import "./interfaces/IEndpointManager.sol";
import "./interfaces/IEndpoint.sol";
import "./interfaces/IEndpointToken.sol";

contract EndpointManager is
    IEndpointManager,
    OwnableUpgradeable,
    ReentrancyGuard
{
    using BytesParsing for bytes;

    address immutable token;
    bool immutable isLockingMode;
    uint16 immutable chainId;
    uint256 immutable evmChainId;

    uint64 sequence;
    uint8 threshold;

    // ========================= ENDPOINT REGISTRATION =========================

    // @dev Information about registered endpoints.
    struct EndpointInfo {
        // whether this endpoint is registered
        bool registered;
        // whether this endpoint is enabled
        bool enabled;
        uint8 index;
    }

    // @dev Information about registered endpoints.
    // This is the source of truth, we define a couple of derived fields below
    // for efficiency.
    mapping(address => EndpointInfo) public endpointInfos;

    // @dev List of enabled endpoints.
    // invariant: forall (a: address), endpointInfos[a].enabled <=> a in enabledEndpoints
    address[] enabledEndpoints;

    // invariant: forall (i: uint8), enabledEndpointBitmap & i == 1 <=> endpointInfos[i].enabled
    uint64 enabledEndpointBitmap;

    uint8 constant MAX_ENDPOINTS = 64;

    // @dev Total number of registered endpoints. This number can only increase.
    // invariant: numRegisteredEndpoints <= MAX_ENDPOINTS
    // invariant: forall (i: uint8),
    //   i < numRegisteredEndpoints <=> exists (a: address), endpointInfos[a].index == i
    uint8 numRegisteredEndpoints;

    // =========================================================================

    // @dev Information about attestations for a given message.
    struct AttestationInfo {
        // bitmap of endpoints that have attested to this message (NOTE: might contain disabled endpoints)
        uint64 attestedEndpoints;
        // whether this message has been executed
        bool executed;
    }

    // Maps are keyed by hash of EndpointManagerMessage.
    mapping(bytes32 => AttestationInfo) public managerMessageAttestations;

    modifier onlyEndpoint() {
        if (!endpointInfos[msg.sender].enabled) {
            revert CallerNotEndpoint(msg.sender);
        }
        _;
    }

    constructor(
        address _token,
        bool _isLockingMode,
        uint16 _chainId,
        uint256 _evmChainId
    ) {
        token = _token;
        isLockingMode = _isLockingMode;
        chainId = _chainId;
        evmChainId = _evmChainId;
        _checkEndpointsInvariants();
    }

    /// @notice Called by the user to send the token cross-chain.
    ///         This function will either lock or burn the sender's tokens.
    ///         Finally, this function will call into the Endpoint contracts to send a message with the incrementing sequence number, msgType = 1y, and the token transfer payload.
    function transfer(
        uint256 amount,
        uint16 recipientChain,
        bytes32 recipient
    ) external payable nonReentrant returns (uint64 msgSequence) {
        // check up front that msg.value will cover the delivery price
        uint256 totalPriceQuote = 0;
        uint256[] memory endpointQuotes = new uint256[](enabledEndpoints.length);
        for (uint256 i = 0; i < enabledEndpoints.length; i++) {
            uint256 endpointPriceQuote = IEndpoint(enabledEndpoints[i])
                .quoteDeliveryPrice(recipientChain);
            endpointQuotes[i] = endpointPriceQuote;
            totalPriceQuote += endpointPriceQuote;
        }
        if (msg.value < totalPriceQuote) {
            revert DeliveryPaymentTooLow(totalPriceQuote, msg.value);
        }

        // refund user extra excess value from msg.value
        uint256 excessValue = totalPriceQuote - msg.value;
        if (excessValue > 0) {
            payable(msg.sender).transfer(excessValue);
        }

        // query tokens decimals
        (, bytes memory queriedDecimals) = token.staticcall(
            abi.encodeWithSignature("decimals()")
        );
        uint8 decimals = abi.decode(queriedDecimals, (uint8));

        // don't deposit dust that can not be bridged due to the decimal shift
        amount = deNormalizeAmount(normalizeAmount(amount, decimals), decimals);

        if (isLockingMode) {
            // use transferFrom to pull tokens from the user and lock them
            // query own token balance before transfer
            uint256 balanceBefore = getTokenBalanceOf(token, address(this));

            // transfer tokens
            SafeERC20.safeTransferFrom(
                IERC20(token),
                msg.sender,
                address(this),
                amount
            );

            // query own token balance after transfer
            uint256 balanceAfter = getTokenBalanceOf(token, address(this));

            // correct amount for potential transfer fees
            amount = balanceAfter - balanceBefore;
        } else {
            // query sender's token balance before transfer
            uint256 balanceBefore = getTokenBalanceOf(token, msg.sender);

            // call the token's burn function to burn the sender's token
            ERC20Burnable(token).burnFrom(msg.sender, amount);

            // query sender's token balance after transfer
            uint256 balanceAfter = getTokenBalanceOf(token, msg.sender);

            // correct amount for potential burn fees
            amount = balanceAfter - balanceBefore;
        }

        // normalize amount decimals
        uint256 normalizedAmount = normalizeAmount(amount, decimals);

        bytes memory encodedTransferPayload = encodeNativeTokenTransfer(
            normalizedAmount,
            token,
            recipient,
            recipientChain
        );

        // construct the ManagerMessage payload
        sequence = useSequence();
        bytes memory encodedManagerPayload = encodeEndpointManagerMessage(
            chainId,
            sequence,
            1,
            encodedTransferPayload
        );

        // call into endpoint contracts to send the message
        for (uint256 i = 0; i < enabledEndpoints.length; i++) {
            IEndpoint(enabledEndpoints[i]).sendMessage{value: endpointQuotes[i]}(
                recipientChain,
                encodedManagerPayload
            );
        }

        // return the sequence number
        return sequence;
    }

    function normalizeAmount(
        uint256 amount,
        uint8 decimals
    ) internal pure returns (uint256) {
        if (decimals > 8) {
            amount /= 10 ** (decimals - 8);
        }
        return amount;
    }

    function deNormalizeAmount(
        uint256 amount,
        uint8 decimals
    ) internal pure returns (uint256) {
        if (decimals > 8) {
            amount *= 10 ** (decimals - 8);
        }
        return amount;
    }

    // @dev Mark a message as executed.
    // This function will revert if the message has already been executed.
    function _markMessageExecuted(
        bytes32 digest
    ) internal {
        // check if this message has already been executed
        if (managerMessageAttestations[digest].executed) {
            revert MessageAlreadyExecuted(digest);
        }

        // mark this message as executed
        managerMessageAttestations[digest].executed = true;
    }

    /// @notice Called by a Endpoint contract to deliver a verified attestation.
    ///         This function will decode the payload as an EndpointManagerMessage to extract the sequence, msgType, and other parameters.
    ///         When the threshold is reached for a sequence, this function will execute logic to handle the action specified by the msgType and payload.
    function attestationReceived(bytes memory payload) external onlyEndpoint {
        // verify chain has not forked
        if (isFork()) {
            revert InvalidFork(evmChainId, block.chainid);
        }

        bytes32 managerMessageHash = computeManagerMessageHash(payload);

        // set the attested flag for this endpoint.
        // TODO: this allows an endpoint to attest to a message multiple times.
        // This is fine, because attestation is idempotent (bitwise or 1), but
        // maybe we want to revert anyway?
        // TODO: factor out the bitmap logic into helper functions (or even a library)
        managerMessageAttestations[managerMessageHash].attestedEndpoints |=
            uint64(1 << endpointInfos[msg.sender].index);

        uint8 attestationCount = messageAttestations(managerMessageHash);

        // end early if the threshold hasn't been met.
        // otherwise, continue with execution for the message type.
        if (attestationCount < threshold) {
            return;
        }

        _markMessageExecuted(managerMessageHash);

        // parse the payload as an EndpointManagerMessage
        EndpointManagerMessage memory message = parseEndpointManagerMessage(
            payload
        );

        // for msgType == 1, parse the payload as a NativeTokenTransfer.
        // for other msgTypes, revert (unsupported for now)
        if (message.msgType != 1) {
            revert UnexpectedEndpointManagerMessageType(message.msgType);
        }
        NativeTokenTransfer
            memory nativeTokenTransfer = parseNativeTokenTransfer(
                message.payload
            );

        // verify that the destination chain is valid
        if (nativeTokenTransfer.toChain != chainId) {
            revert InvalidTargetChain(nativeTokenTransfer.toChain, chainId);
        }

        // calculate proper amount of tokens to unlock/mint to recipient
        // query the decimals of the token contract that's tied to this manager
        // adjust the decimals of the amount in the nativeTokenTransfer payload accordingly
        (, bytes memory queriedDecimals) = token.staticcall(
            abi.encodeWithSignature("decimals()")
        );
        uint8 decimals = abi.decode(queriedDecimals, (uint8));
        uint256 nativeTransferAmount = deNormalizeAmount(
            nativeTokenTransfer.amount,
            decimals
        );

        address transferRecipient = fromWormholeFormat(nativeTokenTransfer.to);

        if (isLockingMode) {
            // unlock tokens to the specified recipient
            SafeERC20.safeTransfer(
                IERC20(token),
                transferRecipient,
                nativeTransferAmount
            );
        } else {
            // mint tokens to the specified recipient
            IEndpointToken(token).mint(transferRecipient, nativeTransferAmount);
        }
    }

    /// @notice Returns the number of Endpoints that must attest to a msgId for it to be considered valid and acted upon.
    function getThreshold() external view returns (uint8) {
        return threshold;
    }

    /// @notice Returns the Endpoint contracts that have been registered via governance.
    function getEndpoints() external view returns (address[] memory) {
        return enabledEndpoints;
    }

    function nextSequence() public view returns (uint64) {
        return sequence;
    }

    function useSequence() internal returns (uint64 currentSequence) {
        currentSequence = nextSequence();
        incrementSequence();
    }

    function incrementSequence() internal {
        sequence++;
    }

    function setThreshold(uint8 newThreshold) external onlyOwner {
        threshold = newThreshold;
        _checkEndpointsInvariants();
    }

    function setEndpoint(address endpoint) external onlyOwner {
        if (endpoint == address(0)) {
            revert InvalidEndpointZeroAddress();
        }

        if (numRegisteredEndpoints >= MAX_ENDPOINTS) {
            revert TooManyEndpoints();
        }

        if (endpointInfos[endpoint].registered) {
            endpointInfos[endpoint].enabled = true;
        } else {
            endpointInfos[endpoint] = EndpointInfo({
                registered: true,
                enabled: true,
                index: numRegisteredEndpoints
            });
            numRegisteredEndpoints++;
        }

        enabledEndpoints.push(endpoint);

        uint64 updatedEnabledEndpointBitmap
            = enabledEndpointBitmap | uint64(1 << endpointInfos[endpoint].index);
        // ensure that this actually changed the bitmap
        assert(updatedEnabledEndpointBitmap > enabledEndpointBitmap);
        enabledEndpointBitmap = updatedEnabledEndpointBitmap;

        emit EndpointAdded(endpoint);

        _checkEndpointsInvariants();
    }

    function removeEndpoint(address endpoint) external onlyOwner {
        if (endpoint == address(0)) {
            revert InvalidEndpointZeroAddress();
        }

        if (!endpointInfos[endpoint].registered) {
            revert NonRegisteredEndpoint(endpoint);
        }

        if (!endpointInfos[endpoint].enabled) {
            revert DisabledEndpoint(endpoint);
        }

        endpointInfos[endpoint].enabled = false;

        uint64 updatedEnabledEndpointBitmap
            = enabledEndpointBitmap & uint64(~(1 << endpointInfos[endpoint].index));
        // ensure that this actually changed the bitmap
        assert(updatedEnabledEndpointBitmap < enabledEndpointBitmap);
        enabledEndpointBitmap = updatedEnabledEndpointBitmap;

        bool removed = false;

        for (uint256 i = 0; i < enabledEndpoints.length; i++) {
            if (enabledEndpoints[i] == endpoint) {
                enabledEndpoints[i] = enabledEndpoints[enabledEndpoints.length - 1];
                enabledEndpoints.pop();
                removed = true;
                break;
            }
        }
        assert(removed);

        emit EndpointRemoved(endpoint);

        _checkEndpointsInvariants();
        // we call the invariant check on the endpoint here as well, since
        // the above check only iterates through the enabled endpoints.
        _checkEndpointInvariants(endpoint);
    }

    function encodeEndpointManagerMessage(
        uint16 _chainId,
        uint64 _sequence,
        uint8 msgType,
        bytes memory payload
    ) public pure returns (bytes memory encoded) {
        // TODO -- should we check payload length here?
        // for example, CCTP integration checks payload is <= max(uint16)
        return abi.encodePacked(_chainId, _sequence, msgType, payload);
    }

    /*
     * @dev Parse a EndpointManagerMessage.
     *
     * @params encoded The byte array corresponding to the encoded message
     */
    function parseEndpointManagerMessage(
        bytes memory encoded
    ) public pure returns (EndpointManagerMessage memory managerMessage) {
        uint256 offset = 0;
        (managerMessage.chainId, offset) = encoded.asUint16(offset);
        (managerMessage.sequence, offset) = encoded.asUint64(offset);
        (managerMessage.msgType, offset) = encoded.asUint8(offset);
        (managerMessage.payload, offset) = encoded.slice(
            offset,
            encoded.length - offset
        );
    }

    function encodeNativeTokenTransfer(
        uint256 amount,
        address tokenAddr,
        bytes32 recipient,
        uint16 toChain
    ) public pure returns (bytes memory encoded) {
        return
            abi.encodePacked(
                amount,
                toWormholeFormat(tokenAddr),
                recipient,
                toChain
            );
    }

    /*
     * @dev Parse a NativeTokenTransfer.
     *
     * @params encoded The byte array corresponding to the encoded message
     */
    function parseNativeTokenTransfer(
        bytes memory encoded
    ) public pure returns (NativeTokenTransfer memory nativeTokenTransfer) {
        uint256 offset = 0;
        (nativeTokenTransfer.amount, offset) = encoded.asUint256(offset);
        (nativeTokenTransfer.tokenAddress, offset) = encoded.asBytes32(offset);
        (nativeTokenTransfer.to, offset) = encoded.asBytes32(offset);
        (nativeTokenTransfer.toChain, offset) = encoded.asUint16(offset);
    }

    function isFork() public view returns (bool) {
        return evmChainId != block.chainid;
    }

    function getTokenBalanceOf(
        address tokenAddr,
        address accountAddr
    ) internal view returns (uint256) {
        (, bytes memory queriedBalance) = tokenAddr.staticcall(
            abi.encodeWithSelector(IERC20.balanceOf.selector, accountAddr)
        );
        return abi.decode(queriedBalance, (uint256));
    }

    function computeManagerMessageHash(
        bytes memory payload
    ) public pure returns (bytes32) {
        return keccak256(payload);
    }

    // @dev Count the number of attestations from enabled endpoints for a given message.
    function messageAttestations(
        bytes32 managerMessageHash
    ) public view returns (uint8 count) {
        uint64 attestedEndpoints = managerMessageAttestations[managerMessageHash].attestedEndpoints;

        return countSetBits(attestedEndpoints & enabledEndpointBitmap);
    }

    // @dev Count the number of set bits in a uint64
    function countSetBits(uint64 x) public pure returns (uint8 count) {
        while (x != 0) {
            x &= x - 1;
            count++;
        }

        return count;
    }

    // @dev Check that the endpoint manager is in a valid state.
    // Checking these invariants is somewhat costly, but we only need to do it
    // when modifying the endpoints, which happens infrequently.
    function _checkEndpointsInvariants() internal view {
        // TODO: add custom errors for each invariant

        for (uint256 i = 0; i < enabledEndpoints.length; i++) {
            _checkEndpointInvariants(enabledEndpoints[i]);
        }

        // invariant: each endpoint is only enabled once
        for (uint256 i = 0; i < enabledEndpoints.length; i++) {
            for (uint256 j = i + 1; j < enabledEndpoints.length; j++) {
                assert(enabledEndpoints[i] != enabledEndpoints[j]);
            }
        }

        // invariant: numRegisteredEndpoints <= MAX_ENDPOINTS
        assert(numRegisteredEndpoints <= MAX_ENDPOINTS);

        // invariant: threshold <= enabledEndpoints.length
        require(threshold <= enabledEndpoints.length, "threshold <= enabledEndpoints.length");
    }

    // @dev Check that the endpoint is in a valid state.
    function _checkEndpointInvariants(address endpoint) internal view {
        EndpointInfo memory endpointInfo = endpointInfos[endpoint];

        // if an endpoint is not registered, it should not be enabled
        assert(endpointInfo.registered || (!endpointInfo.enabled && endpointInfo.index == 0));

        bool endpointInEnabledBitmap = (enabledEndpointBitmap & uint64(1 << endpointInfo.index)) != 0;
        bool endpointEnabled = endpointInfo.enabled;

        bool endpointInEnabledEndpoints = false;

        for (uint256 i = 0; i < enabledEndpoints.length; i++) {
            if (enabledEndpoints[i] == endpoint) {
                endpointInEnabledEndpoints = true;
                break;
            }
        }

        // invariant: endpointInfos[endpoint].enabled <=> enabledEndpointBitmap & (1 << endpointInfos[endpoint].index) != 0
        assert(endpointInEnabledBitmap == endpointEnabled);

        // invariant: endpointInfos[endpoint].enabled <=> endpoint in enabledEndpoints
        assert(endpointInEnabledEndpoints == endpointEnabled);

        assert(endpointInfo.index < numRegisteredEndpoints);
    }
}
