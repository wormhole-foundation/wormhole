import {
  Container,
  makeStyles,
  Typography,
  Paper,
  Button,
} from "@material-ui/core";
import { useEffect } from "react";
import SolanaWalletKey from "../components/SolanaWalletKey";
import { useLogger } from "../contexts/Logger";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";

const useStyles = makeStyles(() => ({}));

function LogWatcher() {
  const { logs, clear, log } = useLogger();

  useEffect(() => {
    log("Instantiated the logger.");
  }, []);

  return (
    <Paper style={{ padding: "1rem", maxHeight: "600px", overflow: "auto" }}>
      <Typography variant="h5">Logs</Typography>
      {logs.map((x) => (
        <Typography>{x}</Typography>
      ))}
      <Button
        onClick={clear}
        variant="contained"
        color="primary"
        style={{ marginTop: "2rem" }}
      >
        Clear All Logs
      </Button>
    </Paper>
  );
}

export default LogWatcher;
