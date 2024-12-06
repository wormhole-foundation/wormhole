//@ts-nocheck
import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgSubmitProposal } from "./types/cosmos/gov/v1beta1/tx";
import { MsgVoteWeighted } from "./types/cosmos/gov/v1beta1/tx";
import { MsgDeposit } from "./types/cosmos/gov/v1beta1/tx";
import { MsgVote } from "./types/cosmos/gov/v1beta1/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/cosmos.gov.v1beta1.MsgSubmitProposal", MsgSubmitProposal],
    ["/cosmos.gov.v1beta1.MsgVoteWeighted", MsgVoteWeighted],
    ["/cosmos.gov.v1beta1.MsgDeposit", MsgDeposit],
    ["/cosmos.gov.v1beta1.MsgVote", MsgVote],
    
];

export { msgTypes }