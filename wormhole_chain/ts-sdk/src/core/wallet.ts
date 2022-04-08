import {
  DirectSecp256k1HdWallet,
  DirectSecp256k1HdWalletOptions,
} from "@cosmjs/proto-signing";
import { ADDRESS_PREFIX, OPERATOR_PREFIX } from "./consts";

export async function getWallet(
  mnemonic: string,
  options?: DirectSecp256k1HdWalletOptions
): Promise<DirectSecp256k1HdWallet> {
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
    ...options,
    prefix: ADDRESS_PREFIX,
  });
  return wallet;
}

export async function getOperatorWallet(
  mnemonic: string,
  options?: DirectSecp256k1HdWalletOptions
): Promise<DirectSecp256k1HdWallet> {
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
    ...options,
    prefix: OPERATOR_PREFIX,
  });
  return wallet;
}

export async function getAddress(
  wallet: DirectSecp256k1HdWallet
): Promise<string> {
  //There are actually up to 5 accounts in a cosmos wallet. I believe this returns the first wallet.
  const [{ address }] = await wallet.getAccounts();

  return address;
}
