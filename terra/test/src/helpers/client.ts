import {
  BlockTxBroadcastResult,
  Int,
  LCDClient,
  MnemonicKey,
  Msg,
  Wallet,
} from "@terra-money/terra.js";

export const GAS_PRICE = 0.2; // uusd

export async function makeProviderAndWallet(): Promise<[LCDClient, Wallet]> {
  // provider
  const client = new LCDClient({
    URL: "http://localhost:1317",
    chainID: "localterra",
    gasAdjustment: "2",
    gasPrices: {
      uusd: GAS_PRICE,
    },
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

  return client.tx.broadcastBlock(tx);
}

export async function transactWithoutMemo(
  client: LCDClient,
  wallet: Wallet,
  msgs: Msg[]
): Promise<BlockTxBroadcastResult> {
  return transact(client, wallet, msgs, "");
}

export async function getNativeBalance(
  client: LCDClient,
  address: string,
  denom: string
): Promise<Int> {
  const [balance] = await client.bank.balance(address);
  const coin = balance.get(denom);
  if (coin === undefined) {
    return new Int(0);
  }
  return new Int(coin.amount);
}
