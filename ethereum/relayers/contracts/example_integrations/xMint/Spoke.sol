// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.4;

import "@openzeppelin/contracts/token/ERC20/ERC20.sol";
//import "solidity-bytes-utils/BytesLib.sol";

import "../../interfaces/IWormhole.sol";
import "../../interfaces/ITokenBridge.sol";
import "../../interfaces/IWormholeReceiver.sol";
import "../../interfaces/IWormholeRelayer.sol";

contract XmintSpoke is IWormholeReceiver {
    // using BytesLib for bytes;

    address owner;

    IWormhole core_bridge;
    ITokenBridge token_bridge;
    IWormholeRelayer core_relayer;
    uint16 hub_contract_chain;
    bytes32 hub_contract_address;

    uint32 nonce = 1;
    uint8 consistencyLevel = 200;

    uint32 SAFE_DELIVERY_GAS_CAPTURE = 1000000; //Capture 1 million gas for fees

    event Log(string indexed str);

    constructor(
        address coreBridgeAddress,
        address tokenBridgeAddress,
        address coreRelayerAddress,
        uint16 hubChain,
        bytes32 hubContractwhFormat
    ) {
        owner = msg.sender;
        core_bridge = IWormhole(coreBridgeAddress);
        token_bridge = ITokenBridge(tokenBridgeAddress);
        core_relayer = IWormholeRelayer(coreRelayerAddress);
        hub_contract_chain = hubChain;
        hub_contract_address = hub_contract_address;
    }

    //This function captures native (ETH) tokens from the user, requests a token transfer to the hub contract,
    //And then requests delivery from relayer network.
    function purchaseTokens() public payable {
        //Calculate how many tokens will be required to cover transaction fees.
        uint256 deliveryFeeBuffer =
            core_relayer.quoteGas(hub_contract_chain, SAFE_DELIVERY_GAS_CAPTURE, core_relayer.getDefaultRelayProvider());

        //require that enough funds were paid to cover this transaction and the relay costs
        require(msg.value > deliveryFeeBuffer + core_bridge.messageFee());

        uint256 bridgeAmount = msg.value - deliveryFeeBuffer - core_bridge.messageFee();

        (bool success, bytes memory data) = address(token_bridge).call{value: bridgeAmount + core_bridge.messageFee()}(
            abi.encodeCall(
                ITokenBridge.wrapAndTransferETHWithPayload,
                (hub_contract_chain, hub_contract_address, nonce, abi.encode(core_relayer.toWormholeFormat(msg.sender)))
            )
        );

        //Request delivery from the relayer network.
        requestDelivery();
    }

    //This function receives messages back from the Hub contract and distributes the tokens to the user.
    function receiveWormholeMessages(bytes[] memory vaas, bytes[] memory additionalData) public payable override {
        //Complete the token bridge transfer
        ITokenBridge.TransferWithPayload memory transferResult =
            token_bridge.parseTransferWithPayload(token_bridge.completeTransferWithPayload(vaas[0]));
        require(
            transferResult.fromAddress == hub_contract_address
                && core_bridge.parseVM(vaas[0]).emitterChainId == hub_contract_chain
        );

        //TODO is the token address the token being transferred, or the origin address?
        ERC20 token = ERC20(core_relayer.fromWormholeFormat(transferResult.tokenAddress));

        token.transfer(
            core_relayer.fromWormholeFormat(bytesToBytes32(transferResult.payload, 0)), transferResult.amount
        );
    }

    function requestDelivery() internal {
        uint256 maxTransactionFee =
            core_relayer.quoteGas(hub_contract_chain, SAFE_DELIVERY_GAS_CAPTURE, core_relayer.getDefaultRelayProvider());
        uint256 receiverValue = 0;

        IWormholeRelayer.Send memory request = IWormholeRelayer.Send({
            targetChain: hub_contract_chain,
            targetAddress: hub_contract_address,
            refundAddress: hub_contract_address, // This will be ignored on the target chain because the intent is to perform a forward
            maxTransactionFee: maxTransactionFee,
            receiverValue: receiverValue, // not needed in this case.
            relayParameters: core_relayer.getDefaultRelayParams() //no overrides
        });

        core_relayer.send{value: maxTransactionFee + receiverValue}(
            request, nonce, core_relayer.getDefaultRelayProvider()
        );
    }

    function bytesToBytes32(bytes memory b, uint256 offset) private pure returns (bytes32) {
        bytes32 out;

        for (uint256 i = 0; i < 32; i++) {
            out |= bytes32(b[offset + i] & 0xFF) >> (i * 8);
        }
        return out;
    }
}
