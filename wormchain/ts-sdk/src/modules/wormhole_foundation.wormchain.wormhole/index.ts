//@ts-nocheck
// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.

import { StdFee } from "@cosmjs/launchpad";
import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry, OfflineSigner, EncodeObject, DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgCreateAllowlistEntryRequest } from "./types/wormhole/tx";
import { MsgAddWasmInstantiateAllowlist } from "./types/wormhole/tx";
import { MsgRegisterAccountAsGuardian } from "./types/wormhole/tx";
import { MsgExecuteGovernanceVAA } from "./types/wormhole/tx";
import { MsgMigrateContract } from "./types/wormhole/tx";
import { MsgExecuteGatewayGovernanceVaa } from "./types/wormhole/tx";
import { MsgInstantiateContract } from "./types/wormhole/tx";
import { MsgDeleteAllowlistEntryRequest } from "./types/wormhole/tx";
import { MsgStoreCode } from "./types/wormhole/tx";
import { MsgDeleteWasmInstantiateAllowlist } from "./types/wormhole/tx";


const types = [
  ["/wormhole_foundation.wormchain.wormhole.MsgCreateAllowlistEntryRequest", MsgCreateAllowlistEntryRequest],
  ["/wormhole_foundation.wormchain.wormhole.MsgAddWasmInstantiateAllowlist", MsgAddWasmInstantiateAllowlist],
  ["/wormhole_foundation.wormchain.wormhole.MsgRegisterAccountAsGuardian", MsgRegisterAccountAsGuardian],
  ["/wormhole_foundation.wormchain.wormhole.MsgExecuteGovernanceVAA", MsgExecuteGovernanceVAA],
  ["/wormhole_foundation.wormchain.wormhole.MsgMigrateContract", MsgMigrateContract],
  ["/wormhole_foundation.wormchain.wormhole.MsgExecuteGatewayGovernanceVaa", MsgExecuteGatewayGovernanceVaa],
  ["/wormhole_foundation.wormchain.wormhole.MsgInstantiateContract", MsgInstantiateContract],
  ["/wormhole_foundation.wormchain.wormhole.MsgDeleteAllowlistEntryRequest", MsgDeleteAllowlistEntryRequest],
  ["/wormhole_foundation.wormchain.wormhole.MsgStoreCode", MsgStoreCode],
  ["/wormhole_foundation.wormchain.wormhole.MsgDeleteWasmInstantiateAllowlist", MsgDeleteWasmInstantiateAllowlist],
  
];
export const MissingWalletError = new Error("wallet is required");

export const registry = new Registry(<any>types);

const defaultFee = {
  amount: [],
  gas: "200000",
};

interface TxClientOptions {
  addr: string
}

interface SignAndBroadcastOptions {
  fee: StdFee,
  memo?: string
}

const txClient = async (wallet: OfflineSigner, { addr: addr }: TxClientOptions = { addr: "http://localhost:26657" }) => {
  if (!wallet) throw MissingWalletError;
  let client;
  if (addr) {
    client = await SigningStargateClient.connectWithSigner(addr, wallet, { registry });
  }else{
    client = await SigningStargateClient.offline( wallet, { registry });
  }
  const { address } = (await wallet.getAccounts())[0];

  return {
    signAndBroadcast: (msgs: EncodeObject[], { fee, memo }: SignAndBroadcastOptions = {fee: defaultFee, memo: ""}) => client.signAndBroadcast(address, msgs, fee,memo),
    msgCreateAllowlistEntryRequest: (data: MsgCreateAllowlistEntryRequest): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgCreateAllowlistEntryRequest", value: MsgCreateAllowlistEntryRequest.fromPartial( data ) }),
    msgAddWasmInstantiateAllowlist: (data: MsgAddWasmInstantiateAllowlist): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgAddWasmInstantiateAllowlist", value: MsgAddWasmInstantiateAllowlist.fromPartial( data ) }),
    msgRegisterAccountAsGuardian: (data: MsgRegisterAccountAsGuardian): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgRegisterAccountAsGuardian", value: MsgRegisterAccountAsGuardian.fromPartial( data ) }),
    msgExecuteGovernanceVAA: (data: MsgExecuteGovernanceVAA): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgExecuteGovernanceVAA", value: MsgExecuteGovernanceVAA.fromPartial( data ) }),
    msgMigrateContract: (data: MsgMigrateContract): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgMigrateContract", value: MsgMigrateContract.fromPartial( data ) }),
    msgExecuteGatewayGovernanceVaa: (data: MsgExecuteGatewayGovernanceVaa): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgExecuteGatewayGovernanceVaa", value: MsgExecuteGatewayGovernanceVaa.fromPartial( data ) }),
    msgInstantiateContract: (data: MsgInstantiateContract): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgInstantiateContract", value: MsgInstantiateContract.fromPartial( data ) }),
    msgDeleteAllowlistEntryRequest: (data: MsgDeleteAllowlistEntryRequest): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgDeleteAllowlistEntryRequest", value: MsgDeleteAllowlistEntryRequest.fromPartial( data ) }),
    msgStoreCode: (data: MsgStoreCode): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgStoreCode", value: MsgStoreCode.fromPartial( data ) }),
    msgDeleteWasmInstantiateAllowlist: (data: MsgDeleteWasmInstantiateAllowlist): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgDeleteWasmInstantiateAllowlist", value: MsgDeleteWasmInstantiateAllowlist.fromPartial( data ) }),
    
  };
};

interface QueryClientOptions {
  addr: string
}

const queryClient = async ({ addr: addr }: QueryClientOptions = { addr: "http://localhost:1317" }) => {
  return new Api({ baseUrl: addr });
};

export {
  txClient,
  queryClient,
};
