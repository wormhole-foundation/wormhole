//@ts-nocheck
// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (_) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgExecuteGovernanceVAA } from "./types/tokenbridge/tx";
import { MsgAttestToken } from "./types/tokenbridge/tx";
import { MsgTransfer } from "./types/tokenbridge/tx";
import { MsgExecuteVAA } from "./types/tokenbridge/tx";
var types = [
    ["/certusone.wormholechain.tokenbridge.MsgExecuteGovernanceVAA", MsgExecuteGovernanceVAA],
    ["/certusone.wormholechain.tokenbridge.MsgAttestToken", MsgAttestToken],
    ["/certusone.wormholechain.tokenbridge.MsgTransfer", MsgTransfer],
    ["/certusone.wormholechain.tokenbridge.MsgExecuteVAA", MsgExecuteVAA],
];
export var MissingWalletError = new Error("wallet is required");
export var registry = new Registry(types);
var defaultFee = {
    amount: [],
    gas: "200000",
};
var txClient = function (wallet, _a) {
    var _b = _a === void 0 ? { addr: "http://localhost:26657" } : _a, addr = _b.addr;
    return __awaiter(void 0, void 0, void 0, function () {
        var client, address;
        return __generator(this, function (_c) {
            switch (_c.label) {
                case 0:
                    if (!wallet)
                        throw MissingWalletError;
                    if (!addr) return [3 /*break*/, 2];
                    return [4 /*yield*/, SigningStargateClient.connectWithSigner(addr, wallet, { registry: registry })];
                case 1:
                    client = _c.sent();
                    return [3 /*break*/, 4];
                case 2: return [4 /*yield*/, SigningStargateClient.offline(wallet, { registry: registry })];
                case 3:
                    client = _c.sent();
                    _c.label = 4;
                case 4: return [4 /*yield*/, wallet.getAccounts()];
                case 5:
                    address = (_c.sent())[0].address;
                    return [2 /*return*/, {
                            signAndBroadcast: function (msgs, _a) {
                                var _b = _a === void 0 ? { fee: defaultFee, memo: "" } : _a, fee = _b.fee, memo = _b.memo;
                                return client.signAndBroadcast(address, msgs, fee, memo);
                            },
                            msgExecuteGovernanceVAA: function (data) { return ({ typeUrl: "/certusone.wormholechain.tokenbridge.MsgExecuteGovernanceVAA", value: MsgExecuteGovernanceVAA.fromPartial(data) }); },
                            msgAttestToken: function (data) { return ({ typeUrl: "/certusone.wormholechain.tokenbridge.MsgAttestToken", value: MsgAttestToken.fromPartial(data) }); },
                            msgTransfer: function (data) { return ({ typeUrl: "/certusone.wormholechain.tokenbridge.MsgTransfer", value: MsgTransfer.fromPartial(data) }); },
                            msgExecuteVAA: function (data) { return ({ typeUrl: "/certusone.wormholechain.tokenbridge.MsgExecuteVAA", value: MsgExecuteVAA.fromPartial(data) }); },
                        }];
            }
        });
    });
};
var queryClient = function (_a) {
    var _b = _a === void 0 ? { addr: "http://localhost:1317" } : _a, addr = _b.addr;
    return __awaiter(void 0, void 0, void 0, function () {
        return __generator(this, function (_c) {
            return [2 /*return*/, new Api({ baseUrl: addr })];
        });
    });
};
export { txClient, queryClient, };
