import { Migrator__factory } from "@certusone/wormhole-sdk";
import {
  Button,
  Container,
  makeStyles,
  Paper,
  TextField,
  Typography,
} from "@material-ui/core";
import { ethers } from "ethers";
import { useState } from "react";
import EthereumSignerKey from "../components/EthereumSignerKey";
import LogWatcher from "../components/LogWatcher";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import { useLogger } from "../contexts/Logger";

const useStyles = makeStyles(() => ({
  rootContainer: {},
  mainPaper: {
    "& > *": {
      margin: "1rem",
    },
    padding: "2rem",
  },
  divider: {
    margin: "2rem",
  },
  spacer: {
    height: "1rem",
  },
}));

function DeployNewEthereum() {
  const classes = useStyles();
  const { signer, provider } = useEthereumProvider();
  const { log } = useLogger();

  const [migratorAddress, setMigratorAddress] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [fromAddress, setFromAddress] = useState<string | null>(null);
  const [toAddress, setToAddress] = useState<string | null>(null);

  const errorMessage =
    error ||
    (!provider && "Wallet not connected") ||
    (!fromAddress && "No 'from' address") ||
    (!toAddress && "No 'to' address");

  const deployPool = async () => {
    if (fromAddress && toAddress) {
      const contractInterface = Migrator__factory.createInterface();
      const bytecode = Migrator__factory.bytecode;
      const factory = new ethers.ContractFactory(
        contractInterface,
        bytecode,
        signer
      );
      const contract = await factory.deploy(fromAddress, toAddress);
      contract.deployed().then(
        (result) => {
          log("Successfully deployed contract at " + result.address);
          setMigratorAddress(result.address);
        },
        (error) => {
          log("Failed to deploy the contract");
          setError((error && error.toString()) || "Unable to create the pool.");
        }
      );
    } else {
    }
  };

  return (
    <>
      <Container maxWidth="md" className={classes.rootContainer}>
        <Paper className={classes.mainPaper}>
          <Typography variant="h6">
            Create a new Ethereum Liquidity Pool
          </Typography>
          <EthereumSignerKey />
          <TextField
            value={fromAddress}
            onChange={(event) => setFromAddress(event.target.value)}
            label={"From Token"}
            fullWidth
            style={{ display: "block" }}
          />
          <TextField
            value={toAddress}
            onChange={(event) => setToAddress(event.target.value)}
            label={"To Token"}
            fullWidth
            style={{ display: "block" }}
          />
          <Button disabled={!!errorMessage} onClick={deployPool}>
            Create
          </Button>
          {errorMessage && <Typography>{errorMessage}</Typography>}
          {migratorAddress !== null && (
            <>
              <Typography>Successfully created a new pool at:</Typography>
              <Typography variant="h5">{migratorAddress}</Typography>
              <Typography>
                You may now populate the pool from the Ethereum pool management
                page.
              </Typography>
            </>
          )}
        </Paper>
        <LogWatcher />
      </Container>
    </>
  );
}

export default DeployNewEthereum;
