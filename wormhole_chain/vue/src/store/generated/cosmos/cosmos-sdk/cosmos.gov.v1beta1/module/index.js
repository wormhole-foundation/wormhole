// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.
import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgSubmitProposal } from "./types/cosmos/gov/v1beta1/tx";
import { MsgVote } from "./types/cosmos/gov/v1beta1/tx";
import { MsgDeposit } from "./types/cosmos/gov/v1beta1/tx";
import { MsgVoteWeighted } from "./types/cosmos/gov/v1beta1/tx";
const types = [
    ["/cosmos.gov.v1beta1.MsgSubmitProposal", MsgSubmitProposal],
    ["/cosmos.gov.v1beta1.MsgVote", MsgVote],
    ["/cosmos.gov.v1beta1.MsgDeposit", MsgDeposit],
    ["/cosmos.gov.v1beta1.MsgVoteWeighted", MsgVoteWeighted],
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
        msgSubmitProposal: (data) => ({ typeUrl: "/cosmos.gov.v1beta1.MsgSubmitProposal", value: data }),
        msgVote: (data) => ({ typeUrl: "/cosmos.gov.v1beta1.MsgVote", value: data }),
        msgDeposit: (data) => ({ typeUrl: "/cosmos.gov.v1beta1.MsgDeposit", value: data }),
        msgVoteWeighted: (data) => ({ typeUrl: "/cosmos.gov.v1beta1.MsgVoteWeighted", value: data }),
    };
};
const queryClient = async ({ addr: addr } = { addr: "http://localhost:1317" }) => {
    return new Api({ baseUrl: addr });
};
export { txClient, queryClient, };
