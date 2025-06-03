//@ts-nocheck
import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgSoftwareUpgrade } from "./types/cosmos/upgrade/v1beta1/tx";
import { MsgCancelUpgrade } from "./types/cosmos/upgrade/v1beta1/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade", MsgSoftwareUpgrade],
    ["/cosmos.upgrade.v1beta1.MsgCancelUpgrade", MsgCancelUpgrade],
    
];

export { msgTypes }