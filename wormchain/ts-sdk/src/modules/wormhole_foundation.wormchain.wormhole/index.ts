//@ts-nocheck
// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.

import { StdFee } from "@cosmjs/launchpad";
import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry, OfflineSigner, EncodeObject, DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgExecuteGovernanceVAA } from "./types/wormhole/tx";
import { MsgStoreCode } from "./types/wormhole/tx";
import { MsgRegisterAccountAsGuardian } from "./types/wormhole/tx";
import { MsgInstantiateContract } from "./types/wormhole/tx";


const types = [
  ["/wormhole_foundation.wormchain.wormhole.MsgExecuteGovernanceVAA", MsgExecuteGovernanceVAA],
  ["/wormhole_foundation.wormchain.wormhole.MsgStoreCode", MsgStoreCode],
  ["/wormhole_foundation.wormchain.wormhole.MsgRegisterAccountAsGuardian", MsgRegisterAccountAsGuardian],
  ["/wormhole_foundation.wormchain.wormhole.MsgInstantiateContract", MsgInstantiateContract],
  
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
    msgExecuteGovernanceVAA: (data: MsgExecuteGovernanceVAA): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgExecuteGovernanceVAA", value: MsgExecuteGovernanceVAA.fromPartial( data ) }),
    msgStoreCode: (data: MsgStoreCode): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgStoreCode", value: MsgStoreCode.fromPartial( data ) }),
    msgRegisterAccountAsGuardian: (data: MsgRegisterAccountAsGuardian): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgRegisterAccountAsGuardian", value: MsgRegisterAccountAsGuardian.fromPartial( data ) }),
    msgInstantiateContract: (data: MsgInstantiateContract): EncodeObject => ({ typeUrl: "/wormhole_foundation.wormchain.wormhole.MsgInstantiateContract", value: MsgInstantiateContract.fromPartial( data ) }),
    
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
