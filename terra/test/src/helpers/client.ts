import {
  BlockTxBroadcastResult,
  LCDClient,
  MnemonicKey,
  Msg,
  Wallet,
} from "@terra-money/terra.js";

export async function makeProviderAndWallet(): Promise<[LCDClient, Wallet]> {
  // provider
  const client = new LCDClient({
    URL: "http://localhost:1317",
    chainID: "localterra",
  });

  // wallet
  const mnemonic =
    "notice oak worry limit wrap speak medal online prefer cluster roof addict wrist behave treat actual wasp year salad speed social layer crew genius";

  const wallet = client.wallet(
    new MnemonicKey({
      mnemonic,
    })
  );
  await wallet.sequence();

  return [client, wallet];
}

export async function transact(
  client: LCDClient,
  wallet: Wallet,
  msgs: Msg[],
  memo: string
): Promise<BlockTxBroadcastResult> {
  const tx = await wallet.createAndSignTx({
    msgs: msgs,
    memo: memo,
  });

  return client.tx.broadcast(tx);
}

export async function transactWithoutMemo(
  client: LCDClient,
  wallet: Wallet,
  msgs: Msg[]
): Promise<BlockTxBroadcastResult> {
  return transact(client, wallet, msgs, "");
}
