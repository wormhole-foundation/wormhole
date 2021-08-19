import { useTerraWallet } from "../contexts/TerraWalletContext";
import ToggleConnectedButton from "./ToggleConnectedButton";

const TerraWalletKey = () => {
  const { connect, disconnect, connected, wallet } = useTerraWallet();
  const pk =
    (wallet &&
      wallet.wallets &&
      wallet.wallets.length > 0 &&
      wallet.wallets[0].terraAddress) ||
    "";
  return (
    <ToggleConnectedButton
      connect={connect}
      disconnect={disconnect}
      connected={connected}
      pk={pk}
    />
  );
};

export default TerraWalletKey;
