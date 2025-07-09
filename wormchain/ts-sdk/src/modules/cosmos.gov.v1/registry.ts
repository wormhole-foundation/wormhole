//@ts-nocheck
import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgSubmitProposal } from "./types/cosmos/gov/v1/tx";
import { MsgDeposit } from "./types/cosmos/gov/v1/tx";
import { MsgUpdateParams } from "./types/cosmos/gov/v1/tx";
import { MsgVote } from "./types/cosmos/gov/v1/tx";
import { MsgVoteWeighted } from "./types/cosmos/gov/v1/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/cosmos.gov.v1.MsgSubmitProposal", MsgSubmitProposal],
    ["/cosmos.gov.v1.MsgDeposit", MsgDeposit],
    ["/cosmos.gov.v1.MsgUpdateParams", MsgUpdateParams],
    ["/cosmos.gov.v1.MsgVote", MsgVote],
    ["/cosmos.gov.v1.MsgVoteWeighted", MsgVoteWeighted],
    
];

export { msgTypes }