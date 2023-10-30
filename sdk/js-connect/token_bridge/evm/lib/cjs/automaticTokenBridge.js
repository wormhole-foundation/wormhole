"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.EvmAutomaticTokenBridge = void 0;
const connect_sdk_1 = require("@wormhole-foundation/connect-sdk");
const connect_sdk_evm_1 = require("@wormhole-foundation/connect-sdk-evm");
const _1 = require(".");
class EvmAutomaticTokenBridge {
    constructor(network, chain, provider, contracts) {
        this.network = network;
        this.chain = chain;
        this.provider = provider;
        this.contracts = contracts;
        if (network === 'Devnet')
            throw new Error('AutomaticTokenBridge not supported on Devnet');
        this.chainId = (0, connect_sdk_evm_1.evmNetworkChainToEvmChainId)(network, chain);
        const tokenBridgeAddress = this.contracts.tokenBridge;
        if (!tokenBridgeAddress)
            throw new Error(`Wormhole Token Bridge contract for domain ${chain} not found`);
        this.tokenBridge = _1.ethers_contracts.Bridge__factory.connect(tokenBridgeAddress, provider);
        const relayerAddress = this.contracts.relayer;
        if (!relayerAddress)
            throw new Error(`Wormhole Token Bridge Relayer contract for domain ${chain} not found`);
        this.tokenBridgeRelayer =
            _1.ethers_contracts.TokenBridgeRelayer__factory.connect(relayerAddress, provider);
    }
    async *redeem(sender, vaa) {
        const senderAddr = new connect_sdk_evm_1.EvmAddress(sender).toString();
        const txReq = await this.tokenBridgeRelayer.completeTransferWithRelay.populateTransaction((0, connect_sdk_1.serialize)(vaa));
        return this.createUnsignedTx((0, connect_sdk_evm_1.addFrom)(txReq, senderAddr), 'TokenBridgeRelayer.completeTransferWithRelay');
    }
    static async fromProvider(provider, config) {
        const [network, chain] = await connect_sdk_evm_1.EvmPlatform.chainFromRpc(provider);
        return new EvmAutomaticTokenBridge(network, chain, provider, config[chain].contracts);
    }
    //alternative naming: initiateTransfer
    async *transfer(sender, recipient, token, amount, relayerFee, nativeGas) {
        const senderAddr = new connect_sdk_evm_1.EvmAddress(sender).toString();
        const recipientChainId = (0, connect_sdk_1.chainToChainId)(recipient.chain);
        const recipientAddress = recipient.address
            .toUniversalAddress()
            .toUint8Array();
        const nativeTokenGas = nativeGas ? nativeGas : 0n;
        if (token === 'native') {
            const txReq = await this.tokenBridgeRelayer.wrapAndTransferEthWithRelay.populateTransaction(nativeTokenGas, recipientChainId, recipientAddress, 0, // skip batching
            { value: relayerFee + amount + nativeTokenGas });
            yield this.createUnsignedTx((0, connect_sdk_evm_1.addFrom)(txReq, senderAddr), 'TokenBridgeRelayer.wrapAndTransferETHWithRelay');
        }
        else {
            //TODO check for ERC-2612 (permit) support on token?
            const tokenAddr = new connect_sdk_evm_1.EvmAddress(token).toString();
            // TODO: allowance?
            const txReq = await this.tokenBridgeRelayer.transferTokensWithRelay.populateTransaction(tokenAddr, amount, nativeTokenGas, recipientChainId, recipientAddress, 0);
            yield this.createUnsignedTx((0, connect_sdk_evm_1.addFrom)(txReq, senderAddr), 'TokenBridgeRelayer.transferTokensWithRelay');
        }
    }
    async getRelayerFee(sender, recipient, token) {
        const tokenId = token === 'native'
            ? (0, connect_sdk_1.nativeChainAddress)([this.chain, await this.tokenBridge.WETH()])
            : token;
        const destChainId = (0, connect_sdk_1.toChainId)(recipient.chain);
        const destTokenAddress = new connect_sdk_evm_1.EvmAddress(tokenId.address.toString()).toString();
        const tokenContract = connect_sdk_evm_1.EvmPlatform.getTokenImplementation(this.provider, destTokenAddress);
        const decimals = await tokenContract.decimals();
        return await this.tokenBridgeRelayer.calculateRelayerFee(destChainId, destTokenAddress, decimals);
    }
    createUnsignedTx(txReq, description, parallelizable = false) {
        return new connect_sdk_evm_1.EvmUnsignedTransaction((0, connect_sdk_evm_1.addChainId)(txReq, this.chainId), this.network, this.chain, description, parallelizable);
    }
}
exports.EvmAutomaticTokenBridge = EvmAutomaticTokenBridge;
