import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import ToggleConnectedButton from "./ToggleConnectedButton";

const SolanaWalletKey = () => {
  const { connect, disconnect, connected, wallet } = useSolanaWallet();
  const pk = wallet?.publicKey?.toString() || "";
  return (
    <ToggleConnectedButton
      connect={connect}
      disconnect={disconnect}
      connected={connected}
      pk={pk}
    />
  );
};

export default SolanaWalletKey;
