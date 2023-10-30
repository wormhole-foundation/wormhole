"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TokenBridgeCoder = exports.TokenBridgeInstruction = void 0;
const accounts_1 = require("./accounts");
const events_1 = require("./events");
const instruction_1 = require("./instruction");
const state_1 = require("./state");
const types_1 = require("./types");
var instruction_2 = require("./instruction");
Object.defineProperty(exports, "TokenBridgeInstruction", { enumerable: true, get: function () { return instruction_2.TokenBridgeInstruction; } });
class TokenBridgeCoder {
    constructor(idl) {
        this.instruction = new instruction_1.TokenBridgeInstructionCoder(idl);
        this.accounts = new accounts_1.TokenBridgeAccountsCoder(idl);
        this.state = new state_1.TokenBridgeStateCoder(idl);
        this.events = new events_1.TokenBridgeEventsCoder(idl);
        this.types = new types_1.TokenBridgeTypesCoder(idl);
    }
}
exports.TokenBridgeCoder = TokenBridgeCoder;
