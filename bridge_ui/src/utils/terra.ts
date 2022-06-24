import {
  canonicalAddress,
  CHAIN_ID_TERRA2,
  isNativeDenom,
  isNativeTerra,
  TerraChainId,
} from "@certusone/wormhole-sdk";
import { formatUnits } from "@ethersproject/units";
import { LCDClient, isTxError } from "@terra-money/terra.js";
import { ConnectedWallet, TxResult } from "@terra-money/wallet-provider";
import axios from "axios";
import {
  getTerraGasPricesUrl,
  getTerraConfig,
  getTokenBridgeAddressForChain,
} from "./consts";

export const NATIVE_TERRA_DECIMALS = 6;
export const LUNA_SYMBOL = "LUNA";
export const LUNA_CLASSIC_SYMBOL = "LUNC";

export const getNativeTerraIcon = (symbol: string) =>
  symbol === LUNA_SYMBOL
    ? `https://assets.terra.money/icon/svg/LUNA.png`
    : symbol === LUNA_CLASSIC_SYMBOL
    ? `https://assets.terra.money/icon/svg/LUNC.svg`
    : `https://assets.terra.money/icon/60/${symbol.slice(
        0,
        symbol.length - 1
      )}.png`;

export const formatNativeDenom = (
  denom: string,
  chainId: TerraChainId
): string => {
  const unit = denom.slice(1).toUpperCase();
  const isValidTerra = isNativeTerra(denom);
  return denom === "uluna"
    ? chainId === CHAIN_ID_TERRA2
      ? LUNA_SYMBOL
      : LUNA_CLASSIC_SYMBOL
    : isValidTerra
    ? unit.slice(0, 2) + "TC"
    : "";
};

export const formatTerraNativeBalance = (balance = ""): string =>
  formatUnits(balance, 6);

export async function waitForTerraExecution(
  transaction: TxResult,
  chainId: TerraChainId
) {
  const lcd = new LCDClient(getTerraConfig(chainId));
  let info;
  while (!info) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    try {
      info = await lcd.tx.txInfo(transaction.result.txhash);
    } catch (e) {
      console.error(e);
    }
  }
  if (isTxError(info)) {
    throw new Error(
      `Tx ${transaction.result.txhash}: error code ${info.code}: ${info.raw_log}`
    );
  }
  return info;
}

export const isValidTerraAddress = (address: string, chainId: TerraChainId) => {
  if (isNativeDenom(address)) {
    return true;
  }
  try {
    const startsWithTerra = address && address.startsWith("terra");
    const isParseable = canonicalAddress(address);
    const isLengthOk =
      isParseable.length === (chainId === CHAIN_ID_TERRA2 ? 32 : 20);
    return !!(startsWithTerra && isParseable && isLengthOk);
  } catch (error) {
    return false;
  }
};

export async function postWithFees(
  wallet: ConnectedWallet,
  msgs: any[],
  memo: string,
  feeDenoms: string[],
  chainId: TerraChainId
) {
  // don't try/catch, let errors propagate
  const lcd = new LCDClient(getTerraConfig(chainId));
  //Thus, we are going to pull it directly from the current FCD.
  const gasPrices = await axios
    .get(getTerraGasPricesUrl(chainId))
    .then((result) => result.data);

  const account = await lcd.auth.accountInfo(wallet.walletAddress);

  const feeEstimate = await lcd.tx.estimateFee(
    [
      {
        sequenceNumber: account.getSequenceNumber(),
        publicKey: account.getPublicKey(),
      },
    ],
    {
      msgs: [...msgs],
      memo,
      feeDenoms,
      gasPrices,
    }
  );

  const result = await wallet.post({
    msgs: [...msgs],
    memo,
    feeDenoms,
    gasPrices,
    fee: feeEstimate,
    // @ts-ignore, https://github.com/terra-money/terra.js/pull/295 (adding isClassic property)
    isClassic: lcd.config.isClassic,
  });

  return result;
}

export interface ExternalIdResponse {
  token_id: {
    Bank?: { denom: string };
    Contract?: {
      NativeCW20?: {
        contract_address: string;
      };
      ForeignToken?: {
        chain_id: string;
        foreign_address: string;
      };
    };
  };
}

// returns the TokenId corresponding to the ExternalTokenId
// see cosmwasm token_addresses.rs
export const queryExternalId = async (externalTokenId: string) => {
  const lcd = new LCDClient(getTerraConfig(CHAIN_ID_TERRA2));
  try {
    const response = await lcd.wasm.contractQuery<ExternalIdResponse>(
      getTokenBridgeAddressForChain(CHAIN_ID_TERRA2),
      {
        external_id: {
          external_id: Buffer.from(externalTokenId, "hex").toString("base64"),
        },
      }
    );
    return (
      // response depends on the token type
      response.token_id.Bank?.denom ||
      response.token_id.Contract?.NativeCW20?.contract_address ||
      response.token_id.Contract?.ForeignToken?.foreign_address
    );
  } catch {
    return null;
  }
};
