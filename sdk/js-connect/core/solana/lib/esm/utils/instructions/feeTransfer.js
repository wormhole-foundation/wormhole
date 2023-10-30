var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { PublicKey, SystemProgram, } from '@solana/web3.js';
import { deriveFeeCollectorKey, getWormholeBridgeData } from '../accounts';
export function createBridgeFeeTransferInstruction(connection, wormholeProgramId, payer, commitment) {
    return __awaiter(this, void 0, void 0, function* () {
        const fee = yield getWormholeBridgeData(connection, wormholeProgramId, commitment).then((data) => data.config.fee);
        return SystemProgram.transfer({
            fromPubkey: new PublicKey(payer),
            toPubkey: deriveFeeCollectorKey(wormholeProgramId),
            lamports: fee,
        });
    });
}
