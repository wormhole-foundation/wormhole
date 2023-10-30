"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.SolanaWormholeCore = void 0;
const connect_sdk_1 = require("@wormhole-foundation/connect-sdk");
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
const utils_1 = require("./utils");
const SOLANA_SEQ_LOG = 'Program log: Sequence: ';
class SolanaWormholeCore {
    constructor(network, chain, connection, contracts) {
        this.network = network;
        this.chain = chain;
        this.connection = connection;
        this.contracts = contracts;
        this.chainId = (0, connect_sdk_1.toChainId)(chain);
        const coreBridgeAddress = contracts.coreBridge;
        if (!coreBridgeAddress)
            throw new Error(`CoreBridge contract Address for chain ${chain} not found`);
        this.coreBridge = (0, utils_1.createReadOnlyWormholeProgramInterface)(coreBridgeAddress, connection);
    }
    static fromProvider(connection, config) {
        return __awaiter(this, void 0, void 0, function* () {
            const [network, chain] = yield connect_sdk_solana_1.SolanaPlatform.chainFromRpc(connection);
            return new SolanaWormholeCore(network, chain, connection, config[chain].contracts);
        });
    }
    publishMessage(sender, message) {
        throw new Error('Method not implemented.');
    }
    parseTransaction(txid) {
        var _a, _b, _c, _d, _e, _f;
        return __awaiter(this, void 0, void 0, function* () {
            const response = yield this.connection.getTransaction(txid);
            if (!response || !((_a = response.meta) === null || _a === void 0 ? void 0 : _a.innerInstructions[0].instructions))
                throw new Error('transaction not found');
            const instructions = (_b = response.meta) === null || _b === void 0 ? void 0 : _b.innerInstructions[0].instructions;
            const accounts = response.transaction.message.accountKeys;
            // find the instruction where the programId equals the Wormhole ProgramId and the emitter equals the Token Bridge
            const bridgeInstructions = instructions.filter((i) => {
                const programId = accounts[i.programIdIndex].toString();
                const wormholeCore = this.coreBridge.programId.toString();
                return programId === wormholeCore;
            });
            if (bridgeInstructions.length === 0)
                throw new Error('no bridge messages found');
            // TODO: unsure about the single bridge instruction and the [2] index, will this always be the case?
            const [logmsg] = bridgeInstructions;
            const emitterAcct = accounts[logmsg.accounts[2]];
            const emitter = (0, connect_sdk_1.toNative)(this.chain, emitterAcct.toString());
            const sequence = (_f = (_e = (_d = (_c = response.meta) === null || _c === void 0 ? void 0 : _c.logMessages) === null || _d === void 0 ? void 0 : _d.filter((msg) => msg.startsWith(SOLANA_SEQ_LOG))) === null || _e === void 0 ? void 0 : _e[0]) === null || _f === void 0 ? void 0 : _f.replace(SOLANA_SEQ_LOG, '');
            if (!sequence) {
                throw new Error('sequence not found');
            }
            return [
                {
                    chain: this.chain,
                    emitter: emitter.toUniversalAddress(),
                    sequence: BigInt(sequence),
                },
            ];
        });
    }
}
exports.SolanaWormholeCore = SolanaWormholeCore;
