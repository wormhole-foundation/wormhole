import { ChainId } from "@certusone/wormhole-sdk";
import { Alert } from "@material-ui/lab";
import useTokenBlacklistWarning from "../../hooks/useTokenBlacklistWarning";

export default function TokenBlacklistWarning({
  sourceChain,
  tokenAddress,
  symbol,
}: {
  sourceChain: ChainId;
  tokenAddress: string | undefined;
  symbol: string | undefined;
}) {
  const tokenBlacklistWarning = useTokenBlacklistWarning(
    sourceChain,
    tokenAddress,
    symbol
  );
  return tokenBlacklistWarning ? (
    <Alert severity="warning">{tokenBlacklistWarning}</Alert>
  ) : null;
}
