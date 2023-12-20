// SPDX-License-Identifier: Apache 2
pragma solidity >=0.6.12 <0.9.0;

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

contract EndpointManager is IEndpointManager, OwnableUpgradeable {
    using BytesParsing for bytes;

    address immutable token;
    bool immutable isLockingMode;
    uint16 immutable chainId;
    uint256 immutable evmChainId;

    uint64 sequence;
    uint8 threshold;
    mapping(address => bool) public isEndpoint;
    address[] endpoints;

    // Maps are keyed by (chainId, sequence) tuple.
    // This is computed as keccak256(abi.encodedPacked(uint16, uint64))
    mapping(bytes32 => mapping(address => bool))
        public chainSequenceAttestations;
    mapping(bytes32 => uint8) public chainSequenceAttestationCounts;

    modifier onlyEndpoint() {
        if (!isEndpoint[msg.sender]) {
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
    }

    /// @notice Called by the user to send the token cross-chain.
    ///         This function will either lock or burn the sender's tokens.
    ///         Finally, this function will call into the Endpoint contracts to send a message with the incrementing sequence number, msgType = 1y, and the token transfer payload.
    function transfer(
        uint256 amount,
        uint16 recipientChain,
        bytes32 recipient
    ) external payable returns (uint64 msgSequence) {
        // check up front that msg.value will cover the delivery price
        uint256 totalPriceQuote = 0;
        uint256[] memory endpointQuotes = new uint256[](endpoints.length);
        for (uint256 i = 0; i < endpoints.length; i++) {
            uint256 endpointPriceQuote = IEndpoint(endpoints[i])
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
        for (uint256 i = 0; i < endpoints.length; i++) {
            IEndpoint(endpoints[i]).sendMessage{value: endpointQuotes[i]}(
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

    /// @notice Called by a Endpoint contract to deliver a verified attestation.
    ///         This function will decode the payload as an EndpointManagerMessage to extract the sequence, msgType, and other parameters.
    ///         When the threshold is reached for a sequence, this function will execute logic to handle the action specified by the msgType and payload.
    function attestationReceived(bytes memory payload) external onlyEndpoint {
        // verify chain has not forked
        if (isFork()) {
            revert InvalidFork(evmChainId, block.chainid);
        }

        // parse the payload as an EndpointManagerMessage
        EndpointManagerMessage memory message = parseEndpointManagerMessage(
            payload
        );

        bytes32 chainSequenceKey = getChainSequenceKey(
            message.chainId,
            message.sequence
        );

        // if the attestation for this sender has already been received, revert
        if (chainSequenceAttestations[chainSequenceKey][msg.sender] == true) {
            revert SequenceAttestationAlreadyReceived(
                message.sequence,
                msg.sender
            );
        }

        // add the Endpoint attestation for the sequence number
        chainSequenceAttestations[chainSequenceKey][msg.sender] = true;

        // increment the attestations for the sequence
        chainSequenceAttestationCounts[chainSequenceKey]++;

        // end early if the threshold hasn't been met.
        // otherwise, continue with execution for the message type.
        if (chainSequenceAttestationCounts[chainSequenceKey] < threshold) {
            return;
        }

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
        return endpoints;
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
    }

    function setEndpoint(address endpoint) external onlyOwner {
        if (endpoint == address(0)) {
            revert InvalidEndpointZeroAddress();
        }

        if (isEndpoint[endpoint]) {
            revert AlreadyRegisteredEndpoint(endpoint);
        }
        isEndpoint[endpoint] = true;
        endpoints.push(endpoint);
        emit EndpointAdded(endpoint);
    }

    function removeEndpoint(address endpoint) external onlyOwner {
        if (endpoint == address(0)) {
            revert InvalidEndpointZeroAddress();
        }

        if (!isEndpoint[endpoint]) {
            revert NonRegisteredEndpoint(endpoint);
        }

        delete isEndpoint[endpoint];

        for (uint256 i = 0; i < endpoints.length; i++) {
            if (endpoints[i] == endpoint) {
                endpoints[i] = endpoints[endpoints.length - 1];
                endpoints.pop();
                break;
            }
        }

        emit EndpointRemoved(endpoint);
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

    function getChainSequenceKey(
        uint16 _chainId,
        uint64 _sequence
    ) public pure returns (bytes32) {
        return keccak256(abi.encodePacked(_chainId, _sequence));
    }
}
