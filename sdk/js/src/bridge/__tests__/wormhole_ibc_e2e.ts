import { describe, test } from "@jest/globals";
import {
  LCDClient,
  MnemonicKey,
  Msg,
  MsgExecuteContract,
  Wallet,
  isTxError,
} from "@terra-money/terra.js";
import { getEmitterAddressTerra, parseSequenceFromLogTerra } from "../..";
import {
  TERRA2_NODE_URL,
  TERRA_CHAIN_ID,
} from "../../token_bridge/__tests__/utils/consts";
import {
  getSignedVAABySequence,
  waitForTerraExecution,
} from "../../token_bridge/__tests__/utils/helpers";
import { CHAIN_ID_SEI } from "../../utils/consts";

const TERRA2_PRIVATE_KEY_4 =
  "bounce success option birth apple portion aunt rural episode solution hockey pencil lend session cause hedgehog slender journey system canvas decorate razor catch empty";

const lcd = new LCDClient({
  URL: TERRA2_NODE_URL,
  chainID: TERRA_CHAIN_ID,
});
const terraWallet = lcd.wallet(
  new MnemonicKey({ mnemonic: TERRA2_PRIVATE_KEY_4 })
);
const terraWalletAddress = terraWallet.key.accAddress;

const terraBroadcastAndWaitForExecution = async (
  msgs: Msg[],
  wallet: Wallet
) => {
  const tx = await wallet.createAndSignTx({
    msgs,
  });
  const txResult = await lcd.tx.broadcast(tx);
  if (isTxError(txResult)) {
    throw new Error("tx error");
  }
  const txInfo = await waitForTerraExecution(txResult.txhash, lcd);
  if (!txInfo) {
    throw new Error("tx info not found");
  }
  return txInfo;
};

const terraBroadcastTxAndGetSignedVaa = async (
  msgs: Msg[],
  wallet: Wallet,
  emitter: string
) => {
  const txInfo = await terraBroadcastAndWaitForExecution(msgs, wallet);
  const txSequence = parseSequenceFromLogTerra(txInfo);
  if (!txSequence) {
    throw new Error("tx sequence not found");
  }
  return await getSignedVAABySequence(CHAIN_ID_SEI, txSequence, emitter);
};

describe("IBC Watcher Integration Tests", () => {
  test('Send a message from "Sei" (Terra2) via IBC', async () => {
    const postMsg = new MsgExecuteContract(
      terraWalletAddress,
      "terra1436kxs0w2es6xlqpp9rd35e3d0cjnw4sv8j3a7483sgks29jqwgsnyey7t",
      {
        post_message: {
          message: Buffer.from("Hello World").toString("base64"),
          nonce: 1,
        },
      }
    );
    const postedVaa = await terraBroadcastTxAndGetSignedVaa(
      [postMsg],
      terraWallet,
      await getEmitterAddressTerra(terraWalletAddress)
    );
  });
});
