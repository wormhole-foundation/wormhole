import { isNativeTerra } from "@certusone/wormhole-sdk";

// inspired by https://github.com/terra-money/station/blob/dca7de43958ce075c6e46605622203b9859b0e14/src/lib/utils/format.ts#L38
export const formatNativeDenom = (denom = ""): string => {
  const unit = denom.slice(1).toUpperCase();
  const isValidTerra = isNativeTerra(denom);
  return denom === "uluna"
    ? "Luna"
    : isValidTerra
    ? unit.slice(0, 2) + "T"
    : "";
};
