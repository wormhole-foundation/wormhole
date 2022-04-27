import { parseUnits } from "@ethersproject/units";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, Keypair, PublicKey, Transaction } from "@solana/web3.js";
import { LCDClient, MnemonicKey, TxInfo } from "@terra-money/terra.js";
import { ethers } from "ethers";
import {
  approveEth,
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  getForeignAssetSolana,
  getSignedVAAWithRetry,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogEth,
  transferFromEth,
} from "../..";
import {
  ETH_CORE_BRIDGE_ADDRESS,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  ETH_TOKEN_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOLANA_PRIVATE_KEY,
  SOLANA_TOKEN_BRIDGE_ADDRESS,
  TERRA_CHAIN_ID,
  TERRA_HOST,
  TERRA_NODE_URL,
  TERRA_PRIVATE_KEY,
  TEST_ERC20,
  WORMHOLE_RPC_HOSTS,
} from "./consts";

export async function transferFromEthToSolana(): Promise<string> {
  // create a keypair for Solana
  const connection = new Connection(SOLANA_HOST, "confirmed");
  const keypair = Keypair.fromSecretKey(SOLANA_PRIVATE_KEY);
  // determine destination address - an associated token account
  const solanaMintKey = new PublicKey(
    (await getForeignAssetSolana(
      connection,
      SOLANA_TOKEN_BRIDGE_ADDRESS,
      CHAIN_ID_ETH,
      hexToUint8Array(nativeToHexString(TEST_ERC20, CHAIN_ID_ETH) || "")
    )) || ""
  );
  const recipient = await Token.getAssociatedTokenAddress(
    ASSOCIATED_TOKEN_PROGRAM_ID,
    TOKEN_PROGRAM_ID,
    solanaMintKey,
    keypair.publicKey
  );
  // create the associated token account if it doesn't exist
  const associatedAddressInfo = await connection.getAccountInfo(recipient);
  if (!associatedAddressInfo) {
    const transaction = new Transaction().add(
      await Token.createAssociatedTokenAccountInstruction(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        solanaMintKey,
        recipient,
        keypair.publicKey, // owner
        keypair.publicKey // payer
      )
    );
    const { blockhash } = await connection.getRecentBlockhash();
    transaction.recentBlockhash = blockhash;
    transaction.feePayer = keypair.publicKey;
    // sign, send, and confirm transaction
    transaction.partialSign(keypair);
    const txid = await connection.sendRawTransaction(transaction.serialize());
    await connection.confirmTransaction(txid);
  }
  // create a signer for Eth
  const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
  const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
  const amount = parseUnits("1", 18);
  // approve the bridge to spend tokens
  await approveEth(ETH_TOKEN_BRIDGE_ADDRESS, TEST_ERC20, signer, amount);
  // transfer tokens
  const receipt = await transferFromEth(
    ETH_TOKEN_BRIDGE_ADDRESS,
    signer,
    TEST_ERC20,
    amount,
    CHAIN_ID_SOLANA,
    hexToUint8Array(
      nativeToHexString(recipient.toString(), CHAIN_ID_SOLANA) || ""
    )
  );
  // get the sequence from the logs (needed to fetch the vaa)
  const sequence = await parseSequenceFromLogEth(
    receipt,
    ETH_CORE_BRIDGE_ADDRESS
  );
  provider.destroy();
  return sequence;
}

export async function waitForTerraExecution(
  transaction: string
): Promise<TxInfo | undefined> {
  const lcd = new LCDClient(TERRA_HOST);
  let done: boolean = false;
  let info;
  while (!done) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    try {
      info = await lcd.tx.txInfo(transaction);
      if (info) {
        done = true;
      }
    } catch (e) {
      console.error(e);
    }
  }
  if (info && info.code !== 0) {
    // error code
    throw new Error(
      `Tx ${transaction}: error code ${info.code}: ${info.raw_log}`
    );
  }
  return info;
}

export async function getSignedVAABySequence(
  chainId: ChainId,
  sequence: string,
  emitterAddress: string
): Promise<Uint8Array> {
  //Note, if handed a sequence which doesn't exist or was skipped for consensus this will retry until the timeout.
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

export async function queryBalanceOnTerra(asset: string): Promise<number> {
  const lcd = new LCDClient({
    URL: TERRA_NODE_URL,
    chainID: TERRA_CHAIN_ID,
  });
  const mk = new MnemonicKey({
    mnemonic: TERRA_PRIVATE_KEY,
  });
  const wallet = lcd.wallet(mk);

  let balance: number = NaN;
  try {
    let coins: any;
    let pagnation: any;
    [coins, pagnation] = await lcd.bank.balance(wallet.key.accAddress);
    console.log("wallet query returned: %o", coins);
    if (coins) {
      let coin = coins.get(asset);
      if (coin) {
        balance = parseInt(coin.toData().amount);
      } else {
        console.error(
          "failed to query coin balance, coin [" +
            asset +
            "] is not in the wallet, coins: %o",
          coins
        );
      }
    } else {
      console.error("failed to query coin balance!");
    }
  } catch (e) {
    console.error("failed to query coin balance: %o", e);
  }

  return balance;
}
