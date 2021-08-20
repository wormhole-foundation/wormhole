import {
  attestFromEth as attestEthTx,
  attestFromSolana as attestSolanaTx,
  attestFromTerra as attestTerraTx,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  parseSequenceFromLogTerra,
} from "@certusone/wormhole-sdk";
import Wallet from "@project-serum/sol-wallet-adapter";
import { Connection } from "@solana/web3.js";
import { ConnectedWallet as TerraConnectedWallet } from "@terra-money/wallet-provider";
import { ethers } from "ethers";
import {
  ETH_BRIDGE_ADDRESS,
  ETH_TOKEN_BRIDGE_ADDRESS,
  TERRA_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
} from "./consts";
import { getSignedVAAWithRetry } from "./getSignedVAAWithRetry";
import { signSendConfirmAndGet } from "./solana";
import { waitForTerraExecution } from "./terra";

export async function attestFromEth(
  signer: ethers.Signer | undefined,
  tokenAddress: string
) {
  if (!signer) return;
  const receipt = await attestEthTx(
    ETH_TOKEN_BRIDGE_ADDRESS,
    signer,
    tokenAddress
  );
  const sequence = parseSequenceFromLogEth(receipt, ETH_BRIDGE_ADDRESS);
  const emitterAddress = getEmitterAddressEth(ETH_TOKEN_BRIDGE_ADDRESS);
  const { vaaBytes } = await getSignedVAAWithRetry(
    CHAIN_ID_ETH,
    emitterAddress,
    sequence
  );
  return vaaBytes;
}

export async function attestFromSolana(
  wallet: Wallet | undefined,
  payerAddress: string | undefined, //TODO: we may not need this since we have wallet
  mintAddress: string
) {
  if (!wallet || !wallet.publicKey || !payerAddress) return;
  // TODO: share connection in context?
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const transaction = await attestSolanaTx(
    connection,
    SOL_BRIDGE_ADDRESS,
    SOL_TOKEN_BRIDGE_ADDRESS,
    payerAddress,
    mintAddress
  );
  const info = await signSendConfirmAndGet(wallet, connection, transaction);
  if (!info) {
    throw new Error("An error occurred while fetching the transaction info");
  }
  const sequence = parseSequenceFromLogSolana(info);
  const emitterAddress = await getEmitterAddressSolana(
    SOL_TOKEN_BRIDGE_ADDRESS
  );
  const { vaaBytes } = await getSignedVAAWithRetry(
    CHAIN_ID_SOLANA,
    emitterAddress,
    sequence
  );
  return vaaBytes;
}

export async function attestFromTerra(
  wallet: TerraConnectedWallet | undefined,
  asset: string | undefined
) {
  if (!wallet || !asset) return;
  const infoMaybe = await attestTerraTx(
    TERRA_TOKEN_BRIDGE_ADDRESS,
    wallet,
    asset
  );
  const info = await waitForTerraExecution(wallet, infoMaybe);
  const sequence = parseSequenceFromLogTerra(info);
  const emitterAddress = await getEmitterAddressTerra(
    TERRA_TOKEN_BRIDGE_ADDRESS
  );
  const result = await getSignedVAAWithRetry(
    CHAIN_ID_TERRA,
    emitterAddress,
    sequence
  );
  return result && result.vaaBytes;
}
