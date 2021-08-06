import { Button, Tooltip, Typography } from "@material-ui/core";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";

const EthereumSignerKey = () => {
  const { connect, disconnect, signerAddress, providerError } =
    useEthereumProvider();
  return (
    <>
      {signerAddress ? (
        <>
          <Tooltip title={signerAddress}>
            <Typography>
              {signerAddress.substring(0, 6)}...
              {signerAddress.substr(signerAddress.length - 4)}
            </Typography>
          </Tooltip>
          <Button
            color="secondary"
            variant="contained"
            size="small"
            onClick={disconnect}
            style={{ width: "100%", textTransform: "none" }}
          >
            Disconnect
          </Button>
        </>
      ) : (
        <Button
          color="primary"
          variant="contained"
          size="small"
          onClick={connect}
          style={{ width: "100%", textTransform: "none" }}
        >
          Connect
        </Button>
      )}
      {providerError ? (
        <Typography variant="body2" color="error">
          {providerError}
        </Typography>
      ) : null}
    </>
  );
};

export default EthereumSignerKey;
