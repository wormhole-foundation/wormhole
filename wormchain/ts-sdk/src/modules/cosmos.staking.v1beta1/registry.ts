//@ts-nocheck
import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgUndelegate } from "./types/cosmos/staking/v1beta1/tx";
import { MsgEditValidator } from "./types/cosmos/staking/v1beta1/tx";
import { MsgBeginRedelegate } from "./types/cosmos/staking/v1beta1/tx";
import { MsgCancelUnbondingDelegation } from "./types/cosmos/staking/v1beta1/tx";
import { MsgCreateValidator } from "./types/cosmos/staking/v1beta1/tx";
import { MsgDelegate } from "./types/cosmos/staking/v1beta1/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/cosmos.staking.v1beta1.MsgUndelegate", MsgUndelegate],
    ["/cosmos.staking.v1beta1.MsgEditValidator", MsgEditValidator],
    ["/cosmos.staking.v1beta1.MsgBeginRedelegate", MsgBeginRedelegate],
    ["/cosmos.staking.v1beta1.MsgCancelUnbondingDelegation", MsgCancelUnbondingDelegation],
    ["/cosmos.staking.v1beta1.MsgCreateValidator", MsgCreateValidator],
    ["/cosmos.staking.v1beta1.MsgDelegate", MsgDelegate],
    
];

export { msgTypes }