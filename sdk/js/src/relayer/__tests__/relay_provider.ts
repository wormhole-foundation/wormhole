import { afterAll, beforeEach, describe, expect, jest, test} from "@jest/globals";
import { ethers } from "ethers";
import { RelayProvider__factory } from "../../ethers-contracts"
import {getAddressInfo, getRPC} from "../consts" 


const env = "DEVNET";
const sourceChainId = 2;
const targetChainId = 4;

// Devnet Private Key
const privateKey = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"

describe("Relay Provider Test", () => {

  
  const addressInfo = getAddressInfo(sourceChainId, env);
  const rpc = getRPC(sourceChainId, env);

  // signers
  const oracleDeployer = new ethers.Wallet(privateKey, new ethers.providers.JsonRpcProvider(rpc));
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
