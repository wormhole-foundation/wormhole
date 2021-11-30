import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
} from "@certusone/wormhole-sdk";
import getSignedVAAWithRetry from "@certusone/wormhole-sdk/lib/cjs/rpc/getSignedVAAWithRetry";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import {
  getNFTBridgeAddressForChain,
  getTokenBridgeAddressForChain,
  WORMHOLE_RPC_HOSTS,
} from "../consts";

export async function getSignedVAABySequence(
  chainId: ChainId,
  sequence: string,
  isNftTransfer: boolean
): Promise<Uint8Array> {
  //Note, if handed a sequence which doesn't exist or was skipped for consensus this will retry until the timeout.
  const contractAddress = isNftTransfer
    ? getNFTBridgeAddressForChain(chainId)
    : getTokenBridgeAddressForChain(chainId);
  const emitterAddress = await nativeAddressToEmitterAddress(
    chainId,
    contractAddress
  );
  console.log("about to do signed vaa with retry");
  const { vaaBytes } = await getSignedVAAWithRetry(
    WORMHOLE_RPC_HOSTS,
    chainId,
    emitterAddress,
    sequence,
    {
      transport: NodeHttpTransport(), //This should only be needed when running in node.
    },
    1000, //retryTimeout
    1000 //Maximum retry attempts
  );

  return vaaBytes;
}

async function nativeAddressToEmitterAddress(
  chainId: ChainId,
  address: string
): Promise<string> {
  if (chainId === CHAIN_ID_SOLANA) {
    return await getEmitterAddressSolana(address);
  } else if (chainId === CHAIN_ID_TERRA) {
    return await getEmitterAddressTerra(address);
  } else {
    return getEmitterAddressEth(address); //Not a mistake, this one is synchronous.
  }
}
