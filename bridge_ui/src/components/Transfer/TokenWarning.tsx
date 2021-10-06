import {
  ChainId,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { getAddress } from "@ethersproject/address";
import { Alert } from "@material-ui/lab";
import { useMemo } from "react";
import {
  ETH_TOKENS_THAT_CAN_BE_SWAPPED_ON_SOLANA,
  ETH_TOKENS_THAT_EXIST_ELSEWHERE,
  SOLANA_TOKENS_THAT_EXIST_ELSEWHERE,
} from "../../utils/consts";

export default function TokenWarning({
  sourceChain,
  tokenAddress,
  symbol,
}: {
  sourceChain: ChainId;
  tokenAddress: string | undefined;
  symbol: string | undefined;
}) {
  const tokenConflictingNativeWarning = useMemo(
    () =>
      tokenAddress &&
      ((sourceChain === CHAIN_ID_SOLANA &&
        SOLANA_TOKENS_THAT_EXIST_ELSEWHERE.includes(tokenAddress)) ||
        (sourceChain === CHAIN_ID_ETH &&
          ETH_TOKENS_THAT_EXIST_ELSEWHERE.includes(getAddress(tokenAddress))))
        ? `Bridging ${
            symbol ? symbol : "the token"
          } via Wormhole will not produce native ${
            symbol ? symbol : "assets"
          }. It will produce a wrapped version which might have no liquidity or utility on the target chain.`
        : undefined,
    [sourceChain, tokenAddress, symbol]
  );
  return tokenConflictingNativeWarning ? (
    <Alert severity="warning">{tokenConflictingNativeWarning}</Alert>
  ) : sourceChain === CHAIN_ID_ETH &&
    tokenAddress &&
    getAddress(tokenAddress) ===
      getAddress("0xae7ab96520de3a18e5e111b5eaab095312d7fe84") ? ( // stETH (Lido)
    <Alert severity="warning">
      Lido stETH rewards can only be received on Ethereum. Use the value
      accruing wrapper token wstETH instead.
    </Alert>
  ) : sourceChain === CHAIN_ID_ETH &&
    tokenAddress &&
    ETH_TOKENS_THAT_CAN_BE_SWAPPED_ON_SOLANA.includes(
      getAddress(tokenAddress)
    ) ? (
    //TODO: will this be accurate with Terra support?
    <Alert severity="info">
      Bridging {symbol ? symbol : "the token"} via Wormhole will not produce
      native {symbol ? symbol : "assets"}. It will produce a wrapped version
      which can be swapped using a stable swap protocol.
    </Alert>
  ) : null;
}
