// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.
import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgAttestToken } from "./types/tokenbridge/tx";
import { MsgExecuteVAA } from "./types/tokenbridge/tx";
import { MsgTransfer } from "./types/tokenbridge/tx";
import { MsgExecuteGovernanceVAA } from "./types/tokenbridge/tx";
const types = [
    ["/certusone.wormholechain.tokenbridge.MsgAttestToken", MsgAttestToken],
    ["/certusone.wormholechain.tokenbridge.MsgExecuteVAA", MsgExecuteVAA],
    ["/certusone.wormholechain.tokenbridge.MsgTransfer", MsgTransfer],
    ["/certusone.wormholechain.tokenbridge.MsgExecuteGovernanceVAA", MsgExecuteGovernanceVAA],
];
export const MissingWalletError = new Error("wallet is required");
const registry = new Registry(types);
const defaultFee = {
    amount: [],
    gas: "200000",
};
const txClient = async (wallet, { addr: addr } = { addr: "http://localhost:26657" }) => {
    if (!wallet)
        throw MissingWalletError;
    const client = await SigningStargateClient.connectWithSigner(addr, wallet, { registry });
    const { address } = (await wallet.getAccounts())[0];
    return {
        signAndBroadcast: (msgs, { fee, memo } = { fee: defaultFee, memo: "" }) => client.signAndBroadcast(address, msgs, fee, memo),
        msgAttestToken: (data) => ({ typeUrl: "/certusone.wormholechain.tokenbridge.MsgAttestToken", value: data }),
        msgExecuteVAA: (data) => ({ typeUrl: "/certusone.wormholechain.tokenbridge.MsgExecuteVAA", value: data }),
        msgTransfer: (data) => ({ typeUrl: "/certusone.wormholechain.tokenbridge.MsgTransfer", value: data }),
        msgExecuteGovernanceVAA: (data) => ({ typeUrl: "/certusone.wormholechain.tokenbridge.MsgExecuteGovernanceVAA", value: data }),
    };
};
const queryClient = async ({ addr: addr } = { addr: "http://localhost:1317" }) => {
    return new Api({ baseUrl: addr });
};
export { txClient, queryClient, };
