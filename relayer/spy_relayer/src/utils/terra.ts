import {
  CHAIN_ID_TERRA2,
  isNativeTerra,
  TerraChainId,
} from "@certusone/wormhole-sdk";

export const LUNA_SYMBOL = "LUNA";
export const LUNA_CLASSIC_SYMBOL = "LUNC";

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
