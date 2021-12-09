import { ChainId } from "@certusone/wormhole-sdk";
import getSignedVAAWithRetry from "@certusone/wormhole-sdk/lib/cjs/rpc/getSignedVAAWithRetry";
import { create } from "domain";
import { ALL_CHAINS, WORMHOLE_RPC_HOSTS } from "./consts";
import { attest } from "./core/attestation";
import { createWrapped } from "./core/createWrapped";
import { getSignedVAABySequence } from "./core/guardianQuery";
import { transferTokens } from "./core/transfer";
import { redeem } from "./core/redeem";

export async function fullAttestation(
  originChain: ChainId,
  originAsset: string
) {
  const otherChainIds = ALL_CHAINS.filter((x) => x !== originChain);
  const sequence = await attest(originChain, originAsset);
  console.log("attest transaction completed " + sequence);
  const signedVaa = await getSignedVAABySequence(originChain, sequence, false);

  console.log("have attestation VAA");
  const promises = [];
  otherChainIds.forEach((chain) =>
    promises.push(
      createWrapped(chain, signedVaa).then(() =>
        console.log("attested on " + chain)
      )
    )
  );

  return await Promise.all(promises);
}

export async function basicTransfer(
  sourceChain: ChainId,
  amount: string,
  targetChain: ChainId,
  sourceAddress: string,
  recipientAddress: string,
  isNativeAsset: boolean,
  assetAddress?: string,
  decimals?: number
) {
  const sequence = await transferTokens(
    sourceChain,
    amount,
    targetChain,
    sourceAddress,
    recipientAddress,
    isNativeAsset,
    assetAddress,
    decimals
  );
  const signedVAA = await getSignedVAABySequence(sourceChain, sequence, false);
  return await redeem(targetChain, signedVAA, false);
}
