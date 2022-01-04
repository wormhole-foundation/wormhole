import { StdFee } from "@cosmjs/launchpad";
import { OfflineSigner, EncodeObject } from "@cosmjs/proto-signing";
import { Api } from "./rest";
import { MsgAttestToken } from "./types/tokenbridge/tx";
import { MsgExecuteVAA } from "./types/tokenbridge/tx";
import { MsgTransfer } from "./types/tokenbridge/tx";
import { MsgExecuteGovernanceVAA } from "./types/tokenbridge/tx";
export declare const MissingWalletError: Error;
interface TxClientOptions {
    addr: string;
}
interface SignAndBroadcastOptions {
    fee: StdFee;
    memo?: string;
}
declare const txClient: (wallet: OfflineSigner, { addr: addr }?: TxClientOptions) => Promise<{
    signAndBroadcast: (msgs: EncodeObject[], { fee, memo }?: SignAndBroadcastOptions) => Promise<import("@cosmjs/stargate").BroadcastTxResponse>;
    msgAttestToken: (data: MsgAttestToken) => EncodeObject;
    msgExecuteVAA: (data: MsgExecuteVAA) => EncodeObject;
    msgTransfer: (data: MsgTransfer) => EncodeObject;
    msgExecuteGovernanceVAA: (data: MsgExecuteGovernanceVAA) => EncodeObject;
}>;
interface QueryClientOptions {
    addr: string;
}
declare const queryClient: ({ addr: addr }?: QueryClientOptions) => Promise<Api<unknown>>;
export { txClient, queryClient, };
