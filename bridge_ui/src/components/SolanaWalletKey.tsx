import { Typography } from "@material-ui/core";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";

const SolanaWalletKey = () => {
  const { wallet } = useSolanaWallet();
  const pk = wallet?.publicKey?.toString();
  if (!pk) return null;
  return (
    <Typography>
      {pk.substring(0, 3)}...{pk.substr(pk.length - 3)}
    </Typography>
  );
};

export default SolanaWalletKey;
