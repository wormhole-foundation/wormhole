import {
  ChainId,
  parseNFTPayload,
  parseTransferPayload,
} from "@certusone/wormhole-sdk";
import { BigNumber } from "@ethersproject/bignumber";
import { formatUnits } from "@ethersproject/units";
import { getSignedVAABySequence } from "./core/guardianQuery";
import { redeem } from "./core/redeem";

/*
The intent behind this module is to represent a backend process which waits for the guardian network to produce a signedVAA
for the bridged tokens, and then submits the VAA to the target chain. This allows the end user to pay fees only in currency from
the source chain, rather than on both chains.

*/
export async function relay(
  sourceChain: ChainId,
  sequence: string,
  isNftTransfer: boolean
) {
  //The transaction has already been submitted by the client,
  //so the relayer first needs to wait for the guardian network to
  //reach consensus and emit the signedVAA.
  const vaaBytes = await getSignedVAABySequence(
    sourceChain, //Emitter address is always the bridge contract address on the source chain
    sequence,
    isNftTransfer
  );

  //The VAA is in the generic format of the Wormhole Core bridge. The VAA payload contains the information needed to redeem the tokens.
  const transferInformation = await parsePayload(
    await parseVaa(vaaBytes),
    isNftTransfer
  );
  //If the relayer is unwilling to submit VAAs at a potential monetary loss, it should first assess if this will be a profitable action.
  const shouldAttempt = await processFee(transferInformation, isNftTransfer);

  if (shouldAttempt) {
    try {
      await redeem(transferInformation.targetChain, vaaBytes, isNftTransfer);
    } catch (e) {
      //Because VAAs are broadcasted publicly, there is a possibility that the VAA
      //will be redeemed by a different relayer. This case should be detected separately from
      //other errors, as it is a do-not-retry scenario.
      //This error will be deterministic, but dependent upon the implementation
      //of the specific wallet provider used. As such, the detection of this error
      //will need to be implemented for each provider separately.
      if (isAlreadyRedeemedError(e as any)) {
      } else {
        throw e;
      }
    }
  }
}

//This function converts the raw VAA into a useable javascript object.
async function parseVaa(bytes: Uint8Array) {
  //parse_vaa is based on wasm
  const { parse_vaa } = await import(
    "@certusone/wormhole-sdk/lib/cjs/solana/core/bridge"
  );
  return parse_vaa(bytes);
}

//This takes the generic parsedVAA format from the Core Bridge, and parses the payload content into the
//protocol specific information of the Token & NFT bridges.
//Note: there are an unlimited variety of VAA formats, and it should not be assumed that a random VAA is one of these
//two types.
async function parsePayload(parsedVaa: any, isNftTransfer: boolean) {
  const buffered = Buffer.from(new Uint8Array(parsedVaa.payload));
  return isNftTransfer
    ? parseNFTPayload(buffered)
    : parseTransferPayload(buffered);
}

//This is a toy function for the purpose of determining a VAA's profitability.
async function processFee(transferInformation: any, isNftTransfer: boolean) {
  if (isNftTransfer) {
    //NFTs are always relayed at a loss, because there is no fee field on the VAA.
    return true;
  }

  const targetAssetDecimals = 8; //This will have to be pulled from either the chain or third party provider
  const targetAssetUnitPrice = 100; //Accurate price quotes are important for determining profitability.
  const feeValue =
    parseFloat(
      formatUnits(
        BigNumber.from((transferInformation.fee || BigInt(0)) as bigint),
        targetAssetDecimals
      )
    ) * targetAssetUnitPrice;

  const estimatedCurrencyFees = 0.01;
  const estimatedCurrencyPrice = -1;
  const transactionCost = estimatedCurrencyFees * estimatedCurrencyPrice;

  return feeValue > transactionCost;
}

const ALREADY_REDEEMED = "Already Redeemed";
function isAlreadyRedeemedError(e: Error) {
  return e?.message === ALREADY_REDEEMED;
}
