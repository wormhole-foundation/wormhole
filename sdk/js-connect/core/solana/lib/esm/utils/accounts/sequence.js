var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { PublicKey, } from '@solana/web3.js';
import { utils } from '@wormhole-foundation/connect-sdk-solana';
export function deriveEmitterSequenceKey(emitter, wormholeProgramId) {
    return utils.deriveAddress([Buffer.from('Sequence'), new PublicKey(emitter).toBytes()], wormholeProgramId);
}
export function getSequenceTracker(connection, emitter, wormholeProgramId, commitment) {
    return __awaiter(this, void 0, void 0, function* () {
        return connection
            .getAccountInfo(deriveEmitterSequenceKey(emitter, wormholeProgramId), commitment)
            .then((info) => SequenceTracker.deserialize(utils.getAccountData(info)));
    });
}
export class SequenceTracker {
    constructor(sequence) {
        this.sequence = sequence;
    }
    static deserialize(data) {
        if (data.length != 8) {
            throw new Error('data.length != 8');
        }
        return new SequenceTracker(data.readBigUInt64LE(0));
    }
    value() {
        return this.sequence;
    }
}
