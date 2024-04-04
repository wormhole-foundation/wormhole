import {
  DirectSecp256k1HdWallet,
  DirectSecp256k1HdWalletOptions,
  OfflineSigner,
} from "@cosmjs/proto-signing";
import { ADDRESS_PREFIX, OPERATOR_PREFIX } from "./consts";

export async function getWallet(
  mnemonic: string,
  options?: DirectSecp256k1HdWalletOptions
): Promise<DirectSecp256k1HdWallet> {
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
    prefix: ADDRESS_PREFIX,
  });
  return wallet;
}

export async function getOperatorWallet(
  mnemonic: string,
  options?: DirectSecp256k1HdWalletOptions
): Promise<DirectSecp256k1HdWallet> {
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
    prefix: OPERATOR_PREFIX,
  });
  return wallet;
}

export async function getAddress(wallet: OfflineSigner): Promise<string> {
  //There are actually up to 5 accounts in a cosmos wallet. I believe this returns the first wallet.
  const [{ address }] = await wallet.getAccounts();

  return address;
}
