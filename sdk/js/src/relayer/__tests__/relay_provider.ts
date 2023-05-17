import { afterAll, beforeEach, describe, expect, jest, test} from "@jest/globals";
import { ethers } from "ethers";
import { RelayProvider__factory } from "../../ethers-contracts"
import {getAddressInfo} from "../consts" 
import {getDefaultProvider} from "../main/helpers"


const env = process.env['ENV'];
if(!env) throw Error("No env specified: tilt or ci or testnet or mainnet");
const network = env == 'tilt' || env == 'ci' ? "DEVNET" : env == 'testnet' ? "TESTNET" : env == 'mainnet' ? "MAINNET" : undefined;
if(!network) throw Error(`Invalid env specified: ${env}`);

const sourceChainId = network == 'DEVNET' ? 2 : 6;
const targetChainId = network == 'DEVNET' ? 4 : 14;

// Devnet Private Key
const privateKey = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"

describe("Relay Provider Test", () => {

  
  const addressInfo = getAddressInfo(sourceChainId, network);
  const provider = getDefaultProvider(network, sourceChainId, env=="ci");

  // signers
  const oracleDeployer = new ethers.Wallet(privateKey, provider);
  const relayProviderAddress = addressInfo.mockRelayProviderAddress;
  if(!relayProviderAddress) throw Error("No relay provider address");
  const relayProvider = RelayProvider__factory.connect(relayProviderAddress, oracleDeployer);


  describe("Read Prices Correctly", () => {
    test("readPrices", async () => {
      const tokenPrice = ethers.BigNumber.from("100000");
      const gasPrice = ethers.BigNumber.from("300000000000")
      
      const tokenPriceReturned = await relayProvider.nativeCurrencyPrice(targetChainId);
      const gasPriceReturned = await relayProvider.gasPrice(targetChainId);

      expect(tokenPriceReturned.toString()).toBe(tokenPrice.toString());
      expect(gasPriceReturned.toString()).toBe(gasPrice.toString());

    });
  });
});
