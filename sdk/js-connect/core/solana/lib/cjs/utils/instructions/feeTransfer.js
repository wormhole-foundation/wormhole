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
exports.createBridgeFeeTransferInstruction = void 0;
const web3_js_1 = require("@solana/web3.js");
const accounts_1 = require("../accounts");
function createBridgeFeeTransferInstruction(connection, wormholeProgramId, payer, commitment) {
    return __awaiter(this, void 0, void 0, function* () {
        const fee = yield (0, accounts_1.getWormholeBridgeData)(connection, wormholeProgramId, commitment).then((data) => data.config.fee);
        return web3_js_1.SystemProgram.transfer({
            fromPubkey: new web3_js_1.PublicKey(payer),
            toPubkey: (0, accounts_1.deriveFeeCollectorKey)(wormholeProgramId),
            lamports: fee,
        });
    });
}
exports.createBridgeFeeTransferInstruction = createBridgeFeeTransferInstruction;
