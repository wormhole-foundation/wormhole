import { AptosClient, AptosAccount, FaucetClient } from "aptos";

const NODE_URL = "https://127.0.0.1:8080/v1";
const FAUCET_URL = "https://127.0.0.1:8081";

(async () => {
  const client = new AptosClient(NODE_URL);
  const faucetClient = new FaucetClient(NODE_URL, FAUCET_URL);
})();