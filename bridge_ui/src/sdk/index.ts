import {
  AccountMeta,
  PublicKey,
  TransactionInstruction,
} from "@solana/web3.js";
import {
  GrpcWebImpl,
  PublicrpcClientImpl,
} from "../proto/publicrpc/v1/publicrpc";
import { ChainId } from "../utils/consts";

// begin from clients\solana\main.ts
export function ixFromRust(data: any): TransactionInstruction {
  let keys: Array<AccountMeta> = data.accounts.map(accountMetaFromRust);
  return new TransactionInstruction({
    programId: new PublicKey(data.program_id),
    data: Buffer.from(data.data),
    keys: keys,
  });
}

function accountMetaFromRust(meta: any): AccountMeta {
  return {
    pubkey: new PublicKey(meta.pubkey),
    isSigner: meta.is_signer,
    isWritable: meta.is_writable,
  };
}
// end from clients\solana\main.ts

export async function getSignedVAA(
  emitterChain: ChainId,
  emitterAddress: string,
  sequence: string
) {
  const rpc = new GrpcWebImpl("http://localhost:8080", {});
  const api = new PublicrpcClientImpl(rpc);
  // TODO: potential infinite loop, support cancellation?
  let result;
  while (!result) {
    console.log("wait 1 second");
    await new Promise((resolve) => setTimeout(resolve, 1000));
    console.log("check for signed vaa", emitterChain, emitterAddress, sequence);
    try {
      result = await api.GetSignedVAA({
        messageId: {
          emitterChain,
          emitterAddress,
          sequence,
        },
      });
      console.log(result);
    } catch (e) {
      console.log(e);
    }
  }
  return result;
}
