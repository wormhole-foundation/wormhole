import { AppBar, Button, Divider, Typography } from "@material-ui/core";
import { useCallback, useState } from "react";
import { default as DeployNewEthereum } from "./DeployNewEthereum";
import MigrateEthereum from "./MigrateEthereum";
import Main from "./Main";
import { CLUSTER } from "../utils/consts";

const ETH = "Interact with an existing Ethereum pool";
const NEW_ETH = "Create a New Ethereum Pool";
const SOL = "Manage Solana Liquidity pools.";

function Home() {
  const [displayedView, setDisplayedView] = useState<string | null>(null);

  const setEth = useCallback(() => {
    setDisplayedView(ETH);
  }, []);

  const setNewEth = useCallback(() => {
    setDisplayedView(NEW_ETH);
  }, []);

  const setSol = useCallback(() => {
    setDisplayedView(SOL);
  }, []);

  const clear = useCallback(() => {
    setDisplayedView(null);
  }, []);

  const backHeader = (
    <>
      <div style={{ padding: ".5rem", textAlign: "center" }}>
        <Typography variant="h5">{displayedView}</Typography>
        <Button onClick={clear} variant="contained" color="default">
          Back
        </Button>
      </div>
      <Divider />
    </>
  );

  const content =
    displayedView === null ? (
      <div style={{ textAlign: "center", padding: "1rem" }}>
        <Typography variant="h5">
          Which action would you like to perform?
        </Typography>
        <div style={{ margin: "2rem" }}>
          <Button
            style={{ margin: ".5rem" }}
            variant="contained"
            onClick={setEth}
          >
            {ETH}
          </Button>
          <Button
            style={{ margin: ".5rem" }}
            variant="contained"
            onClick={setNewEth}
          >
            {NEW_ETH}
          </Button>
          <Button
            style={{ margin: ".5rem" }}
            variant="contained"
            onClick={setSol}
          >
            {SOL}
          </Button>
        </div>
      </div>
    ) : displayedView === ETH ? (
      <>
        {backHeader}
        <MigrateEthereum />
      </>
    ) : displayedView === NEW_ETH ? (
      <>
        {backHeader}
        <DeployNewEthereum />
      </>
    ) : displayedView === SOL ? (
      <>
        {backHeader}
        <Main />
      </>
    ) : null;

  return (
    <>
      {CLUSTER === "mainnet" ? null : (
        <AppBar position="static" color="secondary">
          <Typography style={{ textAlign: "center" }}>
            Caution! You are using the {CLUSTER} build of this app.
          </Typography>
        </AppBar>
      )}
      {content}
    </>
  );
}

export default Home;
