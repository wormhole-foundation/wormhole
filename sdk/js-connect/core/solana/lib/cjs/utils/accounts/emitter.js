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
exports.getProgramSequenceTracker = exports.getEmitterKeys = exports.deriveWormholeEmitterKey = void 0;
const connect_sdk_solana_1 = require("@wormhole-foundation/connect-sdk-solana");
const sequence_1 = require("./sequence");
function deriveWormholeEmitterKey(emitterProgramId) {
    return connect_sdk_solana_1.utils.deriveAddress([Buffer.from('emitter')], emitterProgramId);
}
exports.deriveWormholeEmitterKey = deriveWormholeEmitterKey;
function getEmitterKeys(emitterProgramId, wormholeProgramId) {
    const emitter = deriveWormholeEmitterKey(emitterProgramId);
    return {
        emitter,
        sequence: (0, sequence_1.deriveEmitterSequenceKey)(emitter, wormholeProgramId),
    };
}
exports.getEmitterKeys = getEmitterKeys;
function getProgramSequenceTracker(connection, emitterProgramId, wormholeProgramId, commitment) {
    return __awaiter(this, void 0, void 0, function* () {
        return (0, sequence_1.getSequenceTracker)(connection, deriveWormholeEmitterKey(emitterProgramId), wormholeProgramId, commitment);
    });
}
exports.getProgramSequenceTracker = getProgramSequenceTracker;
