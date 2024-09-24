//@ts-nocheck
import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgSetDenomMetadata } from "./types/osmosis/tokenfactory/v1beta1/tx";
import { MsgCreateDenom } from "./types/osmosis/tokenfactory/v1beta1/tx";
import { MsgBurn } from "./types/osmosis/tokenfactory/v1beta1/tx";
import { MsgMint } from "./types/osmosis/tokenfactory/v1beta1/tx";
import { MsgChangeAdmin } from "./types/osmosis/tokenfactory/v1beta1/tx";
import { MsgForceTransfer } from "./types/osmosis/tokenfactory/v1beta1/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/osmosis.tokenfactory.v1beta1.MsgSetDenomMetadata", MsgSetDenomMetadata],
    ["/osmosis.tokenfactory.v1beta1.MsgCreateDenom", MsgCreateDenom],
    ["/osmosis.tokenfactory.v1beta1.MsgBurn", MsgBurn],
    ["/osmosis.tokenfactory.v1beta1.MsgMint", MsgMint],
    ["/osmosis.tokenfactory.v1beta1.MsgChangeAdmin", MsgChangeAdmin],
    ["/osmosis.tokenfactory.v1beta1.MsgForceTransfer", MsgForceTransfer],
    
];

export { msgTypes }