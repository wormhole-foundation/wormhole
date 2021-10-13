import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  WSOL_ADDRESS,
} from "@certusone/wormhole-sdk";
import { getAddress } from "@ethersproject/address";
import { Alert } from "@material-ui/lab";
import { useMemo } from "react";
import {
  ETH_TOKENS_THAT_CAN_BE_SWAPPED_ON_SOLANA,
  ETH_TOKENS_THAT_EXIST_ELSEWHERE,
  SOLANA_TOKENS_THAT_EXIST_ELSEWHERE,
  WBNB_ADDRESS,
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
  const marketsWarning = useMemo(() => {
    let show = false;
    if (sourceChain === CHAIN_ID_SOLANA && tokenAddress === WSOL_ADDRESS) {
      show = true;
    } else if (sourceChain === CHAIN_ID_BSC && tokenAddress === WBNB_ADDRESS) {
      show = true;
    }
    if (show) {
      return `As of 10/13/2021, markets have not been established for ${
        symbol ? "Wormhole-wrapped " + symbol : "this token"
      }. Please verify this token will be useful on the target chain.`;
    } else {
      return null;
    }
  }, [sourceChain, tokenAddress, symbol]);

  return tokenConflictingNativeWarning ? (
    <Alert severity="warning" variant="outlined">
      {tokenConflictingNativeWarning}
    </Alert>
  ) : marketsWarning ? (
    <Alert severity="warning" variant="outlined">
      {marketsWarning}
    </Alert>
  ) : sourceChain === CHAIN_ID_ETH &&
    tokenAddress &&
    getAddress(tokenAddress) ===
      getAddress("0xae7ab96520de3a18e5e111b5eaab095312d7fe84") ? ( // stETH (Lido)
    <Alert severity="warning" variant="outlined">
      Lido stETH rewards can only be received on Ethereum. Use the value
      accruing wrapper token wstETH instead.
    </Alert>
  ) : sourceChain === CHAIN_ID_ETH &&
    tokenAddress &&
    ETH_TOKENS_THAT_CAN_BE_SWAPPED_ON_SOLANA.includes(
      getAddress(tokenAddress)
    ) ? (
    //TODO: will this be accurate with Terra support?
    <Alert severity="info" variant="outlined">
      Bridging {symbol ? symbol : "the token"} via Wormhole will not produce
      native {symbol ? symbol : "assets"}. It will produce a wrapped version
      which can be swapped using a stable swap protocol.
    </Alert>
  ) : null;
}
