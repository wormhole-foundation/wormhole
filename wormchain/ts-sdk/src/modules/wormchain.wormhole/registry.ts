//@ts-nocheck
import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgDeleteAllowlistEntryRequest } from "./types/wormchain/wormhole/tx";
import { MsgMigrateContract } from "./types/wormchain/wormhole/tx";
import { MsgExecuteGovernanceVAA } from "./types/wormchain/wormhole/tx";
import { MsgStoreCode } from "./types/wormchain/wormhole/tx";
import { MsgRegisterAccountAsGuardian } from "./types/wormchain/wormhole/tx";
import { MsgExecuteGatewayGovernanceVaa } from "./types/wormchain/wormhole/tx";
import { MsgInstantiateContract } from "./types/wormchain/wormhole/tx";
import { MsgAddWasmInstantiateAllowlist } from "./types/wormchain/wormhole/tx";
import { MsgCreateAllowlistEntryRequest } from "./types/wormchain/wormhole/tx";
import { MsgDeleteWasmInstantiateAllowlist } from "./types/wormchain/wormhole/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/wormchain.wormhole.MsgDeleteAllowlistEntryRequest", MsgDeleteAllowlistEntryRequest],
    ["/wormchain.wormhole.MsgMigrateContract", MsgMigrateContract],
    ["/wormchain.wormhole.MsgExecuteGovernanceVAA", MsgExecuteGovernanceVAA],
    ["/wormchain.wormhole.MsgStoreCode", MsgStoreCode],
    ["/wormchain.wormhole.MsgRegisterAccountAsGuardian", MsgRegisterAccountAsGuardian],
    ["/wormchain.wormhole.MsgExecuteGatewayGovernanceVaa", MsgExecuteGatewayGovernanceVaa],
    ["/wormchain.wormhole.MsgInstantiateContract", MsgInstantiateContract],
    ["/wormchain.wormhole.MsgAddWasmInstantiateAllowlist", MsgAddWasmInstantiateAllowlist],
    ["/wormchain.wormhole.MsgCreateAllowlistEntryRequest", MsgCreateAllowlistEntryRequest],
    ["/wormchain.wormhole.MsgDeleteWasmInstantiateAllowlist", MsgDeleteWasmInstantiateAllowlist],
    
];

export { msgTypes }