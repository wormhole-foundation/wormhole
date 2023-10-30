var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
import { utils } from '@wormhole-foundation/connect-sdk-solana';
import { deriveEmitterSequenceKey, getSequenceTracker, } from './sequence';
export function deriveWormholeEmitterKey(emitterProgramId) {
    return utils.deriveAddress([Buffer.from('emitter')], emitterProgramId);
}
export function getEmitterKeys(emitterProgramId, wormholeProgramId) {
    const emitter = deriveWormholeEmitterKey(emitterProgramId);
    return {
        emitter,
        sequence: deriveEmitterSequenceKey(emitter, wormholeProgramId),
    };
}
export function getProgramSequenceTracker(connection, emitterProgramId, wormholeProgramId, commitment) {
    return __awaiter(this, void 0, void 0, function* () {
        return getSequenceTracker(connection, deriveWormholeEmitterKey(emitterProgramId), wormholeProgramId, commitment);
    });
}
