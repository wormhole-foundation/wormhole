import { Typography } from "@material-ui/core";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import ToggleConnectedButton from "./ToggleConnectedButton";

const EthereumSignerKey = () => {
  const { connect, disconnect, signerAddress, providerError } =
    useEthereumProvider();
  return (
    <>
      <ToggleConnectedButton
        connect={connect}
        disconnect={disconnect}
        connected={!!signerAddress}
        pk={signerAddress || ""}
      />
      {providerError ? (
        <Typography variant="body2" color="error">
          {providerError}
        </Typography>
      ) : null}
    </>
  );
};

export default EthereumSignerKey;
