import { Typography } from "@material-ui/core";
import { useTerraWallet } from "../contexts/TerraWalletContext";
import ToggleConnectedButton from "./ToggleConnectedButton";

const TerraWalletKey = () => {
  const { connect, disconnect, connected, wallet, providerError } =
    useTerraWallet();
  const pk =
    (wallet &&
      wallet.wallets &&
      wallet.wallets.length > 0 &&
      wallet.wallets[0].terraAddress) ||
    "";
  return (
    <>
      <ToggleConnectedButton
        connect={connect}
        disconnect={disconnect}
        connected={connected}
        pk={pk}
      />
      {providerError ? (
        <Typography variant="body2" color="error">
          {providerError}
        </Typography>
      ) : null}
    </>
  );
};

export default TerraWalletKey;
