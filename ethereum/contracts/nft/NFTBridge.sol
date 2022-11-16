// contracts/NFTBridge.sol
// SPDX-License-Identifier: Apache 2

pragma solidity ^0.8.0;

import "@openzeppelin/contracts/token/ERC721/IERC721.sol";
import "@openzeppelin/contracts/token/ERC721/IERC721Receiver.sol";

import "../libraries/external/BytesLib.sol";

import "./NFTBridgeGetters.sol";
import "./NFTBridgeSetters.sol";
import "./NFTBridgeStructs.sol";
import "./NFTBridgeGovernance.sol";

import "./token/NFT.sol";
import "./token/NFTImplementation.sol";

contract NFTBridge is NFTBridgeGovernance {
    using BytesLib for bytes;

    // Initiate a Transfer
    function transferNFT(address token, uint256 tokenID, uint16 recipientChain, bytes32 recipient, uint32 nonce) public payable returns (uint64 sequence) {
        // determine token parameters
        uint16 tokenChain;
        bytes32 tokenAddress;
        if (isWrappedAsset(token)) {
            tokenChain = NFTImplementation(token).chainId();
            tokenAddress = NFTImplementation(token).nativeContract();
        } else {
            tokenChain = chainId();
            tokenAddress = bytes32(uint256(uint160(token)));
            // Verify that the correct interfaces are implemented
            require(ERC165(token).supportsInterface(type(IERC721).interfaceId), "must support the ERC721 interface");
            require(ERC165(token).supportsInterface(type(IERC721Metadata).interfaceId), "must support the ERC721-Metadata extension");
        }

        string memory symbolString;
        string memory nameString;
        string memory uriString;
        {
            if (tokenChain != 1) { // SPL tokens use cache
                (,bytes memory queriedSymbol) = token.staticcall(abi.encodeWithSignature("symbol()"));
                (,bytes memory queriedName) = token.staticcall(abi.encodeWithSignature("name()"));
                symbolString = abi.decode(queriedSymbol, (string));
                nameString = abi.decode(queriedName, (string));
            }

            (,bytes memory queriedURI) = token.staticcall(abi.encodeWithSignature("tokenURI(uint256)", tokenID));
            uriString = abi.decode(queriedURI, (string));
        }

        bytes32 symbol;
        bytes32 name;
        if (tokenChain == 1) {
            // use cached SPL token info, as the contracts uses unified values
            NFTBridgeStorage.SPLCache memory cache = splCache(tokenID);
            symbol = cache.symbol;
            name = cache.name;
            clearSplCache(tokenID);
        } else {
            assembly {
            // first 32 bytes hold string length
            // mload then loads the next word, i.e. the first 32 bytes of the strings
            // NOTE: this means that we might end up with an
            // invalid utf8 string (e.g. if we slice an emoji in half).  The VAA
            // payload specification doesn't require that these are valid utf8
            // strings, and it's cheaper to do any validation off-chain for
            // presentation purposes
                symbol := mload(add(symbolString, 32))
                name := mload(add(nameString, 32))
            }
        }

        IERC721(token).safeTransferFrom(msg.sender, address(this), tokenID);
        if (tokenChain != chainId()) {
            NFTImplementation(token).burn(tokenID);
        }

        sequence = logTransfer(NFTBridgeStructs.Transfer({
            tokenAddress : tokenAddress,
            tokenChain   : tokenChain,
            name         : name,
            symbol       : symbol,
            tokenID      : tokenID,
            uri          : uriString,
            to           : recipient,
            toChain      : recipientChain
        }), msg.value, nonce);
    }

    function logTransfer(NFTBridgeStructs.Transfer memory transfer, uint256 callValue, uint32 nonce) internal returns (uint64 sequence) {
        bytes memory encoded = encodeTransfer(transfer);

        sequence = wormhole().publishMessage{
            value : callValue
        }(nonce, encoded, finality());
    }

    function completeTransfer(bytes memory encodedVm) public {
        _completeTransfer(encodedVm);
    }

    // Execute a Transfer message
    function _completeTransfer(bytes memory encodedVm) internal {
        (IWormhole.VM memory vm, bool valid, string memory reason) = wormhole().parseAndVerifyVM(encodedVm);

        require(valid, reason);
        require(verifyBridgeVM(vm), "invalid emitter");

        NFTBridgeStructs.Transfer memory transfer = parseTransfer(vm.payload);

        require(!isTransferCompleted(vm.hash), "transfer already completed");
        setTransferCompleted(vm.hash);

        require(transfer.toChain == chainId(), "invalid target chain");

        IERC721 transferToken;
        if (transfer.tokenChain == chainId()) {
            transferToken = IERC721(address(uint160(uint256(transfer.tokenAddress))));
        } else {
            address wrapped = wrappedAsset(transfer.tokenChain, transfer.tokenAddress);

            // If the wrapped asset does not exist yet, create it
            if (wrapped == address(0)) {
                wrapped = _createWrapped(transfer.tokenChain, transfer.tokenAddress, transfer.name, transfer.symbol);
            }

            transferToken = IERC721(wrapped);
        }

        // transfer bridged NFT to recipient
        address transferRecipient = address(uint160(uint256(transfer.to)));

        if (transfer.tokenChain != chainId()) {
            if (transfer.tokenChain == 1) {
                // Cache SPL token info which otherwise would get lost
                setSplCache(transfer.tokenID, NFTBridgeStorage.SPLCache({
                    name : transfer.name,
                    symbol : transfer.symbol
                }));
            }

            // mint wrapped asset
            NFTImplementation(address(transferToken)).mint(transferRecipient, transfer.tokenID, transfer.uri);
        } else {
            transferToken.safeTransferFrom(address(this), transferRecipient, transfer.tokenID);
        }
    }

    // Creates a wrapped asset using AssetMeta
    function _createWrapped(uint16 tokenChain, bytes32 tokenAddress, bytes32 name, bytes32 symbol) internal returns (address token) {
        require(tokenChain != chainId(), "can only wrap tokens from foreign chains");
        require(wrappedAsset(tokenChain, tokenAddress) == address(0), "wrapped asset already exists");

        // SPL NFTs all use the same NFT contract, so unify the name
        if (tokenChain == 1) {
            // "Wormhole Bridged Solana-NFT" - right-padded
            name =   0x576f726d686f6c65204272696467656420536f6c616e612d4e46540000000000;
            // "WORMSPLNFT" - right-padded
            symbol = 0x574f524d53504c4e465400000000000000000000000000000000000000000000;
        }

        // initialize the NFTImplementation
        bytes memory initialisationArgs = abi.encodeWithSelector(
            NFTImplementation.initialize.selector,
            bytes32ToString(name),
            bytes32ToString(symbol),

            address(this),

            tokenChain,
            tokenAddress
        );

        // initialize the BeaconProxy
        bytes memory constructorArgs = abi.encode(address(this), initialisationArgs);

        // deployment code
        bytes memory bytecode = abi.encodePacked(type(BridgeNFT).creationCode, constructorArgs);

        bytes32 salt = keccak256(abi.encodePacked(tokenChain, tokenAddress));

        assembly {
            token := create2(0, add(bytecode, 0x20), mload(bytecode), salt)

            if iszero(extcodesize(token)) {
                revert(0, 0)
            }
        }

        setWrappedAsset(tokenChain, tokenAddress, token);
    }

    function verifyBridgeVM(IWormhole.VM memory vm) internal view returns (bool){
        require(!isFork(), "invalid fork");
        if (bridgeContracts(vm.emitterChainId) == vm.emitterAddress) {
            return true;
        }

        return false;
    }

    function encodeTransfer(NFTBridgeStructs.Transfer memory transfer) public pure returns (bytes memory encoded) {
        // There is a global limit on 200 bytes of tokenURI in Wormhole due to Solana
        require(bytes(transfer.uri).length <= 200, "tokenURI must not exceed 200 bytes");

        encoded = abi.encodePacked(
            uint8(1),
            transfer.tokenAddress,
            transfer.tokenChain,
            transfer.symbol,
            transfer.name,
            transfer.tokenID,
            uint8(bytes(transfer.uri).length),
            transfer.uri,
            transfer.to,
            transfer.toChain
        );
    }

    function parseTransfer(bytes memory encoded) public pure returns (NFTBridgeStructs.Transfer memory transfer) {
        uint index = 0;

        uint8 payloadID = encoded.toUint8(index);
        index += 1;

        require(payloadID == 1, "invalid Transfer");

        transfer.tokenAddress = encoded.toBytes32(index);
        index += 32;

        transfer.tokenChain = encoded.toUint16(index);
        index += 2;

        transfer.symbol = encoded.toBytes32(index);
        index += 32;

        transfer.name = encoded.toBytes32(index);
        index += 32;

        transfer.tokenID = encoded.toUint256(index);
        index += 32;

        // Ignore length due to malformatted payload
        index += 1;
        transfer.uri = string(encoded.slice(index, encoded.length - index - 34));

        // From here we read backwards due malformatted package
        index = encoded.length;

        index -= 2;
        transfer.toChain = encoded.toUint16(index);

        index -= 32;
        transfer.to = encoded.toBytes32(index);

        //require(encoded.length == index, "invalid Transfer");
    }

    function onERC721Received(
        address operator,
        address,
        uint256,
        bytes calldata
    ) external view returns (bytes4){
        require(operator == address(this), "can only bridge tokens via transferNFT method");
        return type(IERC721Receiver).interfaceId;
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
}
