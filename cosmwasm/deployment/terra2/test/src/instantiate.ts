import {
  LCDClient,
  MsgInstantiateContract,
  MsgStoreCode,
  Wallet,
} from "@terra-money/terra.js";
import { readFileSync } from "fs";

import { transactWithoutMemo } from "./helpers/client";

export async function storeCode(
  terra: LCDClient,
  wallet: Wallet,
  wasm: string
): Promise<number> {
  const contract_bytes = readFileSync(wasm);
  const store_code = new MsgStoreCode(
    wallet.key.accAddress,
    contract_bytes.toString("base64")
  );
  const receipt = await transactWithoutMemo(terra, wallet, [store_code]);

  // @ts-ignore
  const ci = /"code_id","value":"([^"]+)/gm.exec(receipt.raw_log)[1];
  return parseInt(ci);
}

export async function deploy(
  terra: LCDClient,
  wallet: Wallet,
  wasm: string,
  instantiateMsg: Object,
  label: string
): Promise<string> {
  const codeId = await storeCode(terra, wallet, wasm);

  const msgs = [
    new MsgInstantiateContract(
      wallet.key.accAddress,
      wallet.key.accAddress,
      codeId,
      instantiateMsg,
      undefined,
      label
    ),
  ];
  const receipt = await transactWithoutMemo(terra, wallet, msgs);

  // @ts-ignore
  return /"_contract_address","value":"([^"]+)/gm.exec(receipt.raw_log)[1];
}

export async function deployWithCodeID(
  terra: LCDClient,
  wallet: Wallet,
  instantiateMsg: Object,
  label: string,
  codeId: number
): Promise<string> {
  const msgs = [
    new MsgInstantiateContract(
      wallet.key.accAddress,
      wallet.key.accAddress,
      codeId,
      instantiateMsg,
      undefined,
      label
    ),
  ];
  const receipt = await transactWithoutMemo(terra, wallet, msgs);

  // @ts-ignore
  return /"_contract_address","value":"([^"]+)/gm.exec(receipt.raw_log)[1];
}
