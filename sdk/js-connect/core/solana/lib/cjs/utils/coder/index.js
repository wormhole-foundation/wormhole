"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.WormholeCoder = exports.WormholeInstruction = void 0;
const accounts_1 = require("./accounts");
const events_1 = require("./events");
const instruction_1 = require("./instruction");
const state_1 = require("./state");
const types_1 = require("./types");
var instruction_2 = require("./instruction");
Object.defineProperty(exports, "WormholeInstruction", { enumerable: true, get: function () { return instruction_2.WormholeInstruction; } });
class WormholeCoder {
    constructor(idl) {
        this.instruction = new instruction_1.WormholeInstructionCoder(idl);
        this.accounts = new accounts_1.WormholeAccountsCoder(idl);
        this.state = new state_1.WormholeStateCoder(idl);
        this.events = new events_1.WormholeEventsCoder(idl);
        this.types = new types_1.WormholeTypesCoder(idl);
    }
}
exports.WormholeCoder = WormholeCoder;
