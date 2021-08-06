import { Button, Tooltip, Typography } from "@material-ui/core";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";

const SolanaWalletKey = () => {
  const { connect, disconnect, connected, wallet } = useSolanaWallet();
  const pk = wallet?.publicKey?.toString() || "";
  return (
    <>
      {connected ? (
        <>
          <Tooltip title={pk}>
            <Typography>
              {pk.substring(0, 3)}...{pk.substr(pk.length - 3)}
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
    </>
  );
};

export default SolanaWalletKey;
