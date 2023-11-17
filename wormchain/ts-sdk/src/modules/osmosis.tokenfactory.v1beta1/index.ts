//@ts-nocheck
// THIS FILE IS GENERATED AUTOMATICALLY. DO NOT MODIFY.

import { StdFee } from "@cosmjs/launchpad";
import { SigningStargateClient } from "@cosmjs/stargate";
import { Registry, OfflineSigner, EncodeObject, DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgCreateDenom } from "./types/osmosis/tokenfactory/v1beta1/tx";
import { MsgMint } from "./types/osmosis/tokenfactory/v1beta1/tx";
import { MsgForceTransfer } from "./types/osmosis/tokenfactory/v1beta1/tx";
import { MsgSetDenomMetadata } from "./types/osmosis/tokenfactory/v1beta1/tx";
import { MsgBurn } from "./types/osmosis/tokenfactory/v1beta1/tx";
import { MsgChangeAdmin } from "./types/osmosis/tokenfactory/v1beta1/tx";


const types = [
  ["/osmosis.tokenfactory.v1beta1.MsgCreateDenom", MsgCreateDenom],
  ["/osmosis.tokenfactory.v1beta1.MsgMint", MsgMint],
  ["/osmosis.tokenfactory.v1beta1.MsgForceTransfer", MsgForceTransfer],
  ["/osmosis.tokenfactory.v1beta1.MsgSetDenomMetadata", MsgSetDenomMetadata],
  ["/osmosis.tokenfactory.v1beta1.MsgBurn", MsgBurn],
  ["/osmosis.tokenfactory.v1beta1.MsgChangeAdmin", MsgChangeAdmin],
  
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
    msgCreateDenom: (data: MsgCreateDenom): EncodeObject => ({ typeUrl: "/osmosis.tokenfactory.v1beta1.MsgCreateDenom", value: MsgCreateDenom.fromPartial( data ) }),
    msgMint: (data: MsgMint): EncodeObject => ({ typeUrl: "/osmosis.tokenfactory.v1beta1.MsgMint", value: MsgMint.fromPartial( data ) }),
    msgForceTransfer: (data: MsgForceTransfer): EncodeObject => ({ typeUrl: "/osmosis.tokenfactory.v1beta1.MsgForceTransfer", value: MsgForceTransfer.fromPartial( data ) }),
    msgSetDenomMetadata: (data: MsgSetDenomMetadata): EncodeObject => ({ typeUrl: "/osmosis.tokenfactory.v1beta1.MsgSetDenomMetadata", value: MsgSetDenomMetadata.fromPartial( data ) }),
    msgBurn: (data: MsgBurn): EncodeObject => ({ typeUrl: "/osmosis.tokenfactory.v1beta1.MsgBurn", value: MsgBurn.fromPartial( data ) }),
    msgChangeAdmin: (data: MsgChangeAdmin): EncodeObject => ({ typeUrl: "/osmosis.tokenfactory.v1beta1.MsgChangeAdmin", value: MsgChangeAdmin.fromPartial( data ) }),
    
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
