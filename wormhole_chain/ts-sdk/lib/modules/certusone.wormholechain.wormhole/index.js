"use strict";
//@ts-nocheck
// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.
Object.defineProperty(exports, "__esModule", { value: true });
exports.queryClient = exports.txClient = exports.registry = exports.MissingWalletError = void 0;
const stargate_1 = require("@cosmjs/stargate");
const proto_signing_1 = require("@cosmjs/proto-signing");
const rest_1 = require("./rest");
const tx_1 = require("./types/wormhole/tx");
const tx_2 = require("./types/wormhole/tx");
const types = [
    ["/certusone.wormholechain.wormhole.MsgExecuteGovernanceVAA", tx_1.MsgExecuteGovernanceVAA],
    ["/certusone.wormholechain.wormhole.MsgRegisterAccountAsGuardian", tx_2.MsgRegisterAccountAsGuardian],
];
exports.MissingWalletError = new Error("wallet is required");
exports.registry = new proto_signing_1.Registry(types);
const defaultFee = {
    amount: [],
    gas: "200000",
};
const txClient = async (wallet, { addr: addr } = { addr: "http://localhost:26657" }) => {
    if (!wallet)
        throw exports.MissingWalletError;
    let client;
    if (addr) {
        client = await stargate_1.SigningStargateClient.connectWithSigner(addr, wallet, { registry: exports.registry });
    }
    else {
        client = await stargate_1.SigningStargateClient.offline(wallet, { registry: exports.registry });
    }
    const { address } = (await wallet.getAccounts())[0];
    return {
        signAndBroadcast: (msgs, { fee, memo } = { fee: defaultFee, memo: "" }) => client.signAndBroadcast(address, msgs, fee, memo),
        msgExecuteGovernanceVAA: (data) => ({ typeUrl: "/certusone.wormholechain.wormhole.MsgExecuteGovernanceVAA", value: tx_1.MsgExecuteGovernanceVAA.fromPartial(data) }),
        msgRegisterAccountAsGuardian: (data) => ({ typeUrl: "/certusone.wormholechain.wormhole.MsgRegisterAccountAsGuardian", value: tx_2.MsgRegisterAccountAsGuardian.fromPartial(data) }),
    };
};
exports.txClient = txClient;
const queryClient = async ({ addr: addr } = { addr: "http://localhost:1317" }) => {
    return new rest_1.Api({ baseUrl: addr });
};
exports.queryClient = queryClient;
