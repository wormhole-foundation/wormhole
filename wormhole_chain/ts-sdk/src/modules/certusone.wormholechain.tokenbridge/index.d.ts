//@ts-nocheck
import { StdFee } from "@cosmjs/launchpad";
import { Registry, OfflineSigner, EncodeObject } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgExecuteGovernanceVAA } from "./types/tokenbridge/tx";
import { MsgAttestToken } from "./types/tokenbridge/tx";
import { MsgTransfer } from "./types/tokenbridge/tx";
import { MsgExecuteVAA } from "./types/tokenbridge/tx";
export declare const MissingWalletError: Error;
export declare const registry: Registry;
interface TxClientOptions {
    addr: string;
}
interface SignAndBroadcastOptions {
    fee: StdFee;
    memo?: string;
}
declare const txClient: (wallet: OfflineSigner, { addr: addr }?: TxClientOptions) => Promise<{
    signAndBroadcast: (msgs: EncodeObject[], { fee, memo }?: SignAndBroadcastOptions) => any;
    msgExecuteGovernanceVAA: (data: MsgExecuteGovernanceVAA) => EncodeObject;
    msgAttestToken: (data: MsgAttestToken) => EncodeObject;
    msgTransfer: (data: MsgTransfer) => EncodeObject;
    msgExecuteVAA: (data: MsgExecuteVAA) => EncodeObject;
}>;
interface QueryClientOptions {
    addr: string;
}
declare const queryClient: ({ addr: addr }?: QueryClientOptions) => Promise<Api<unknown>>;
export { txClient, queryClient, };
