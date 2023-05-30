import { afterAll, beforeEach, describe, expect, jest, test} from "@jest/globals";
import { ethers } from "ethers";
import { DeliveryProvider__factory } from "../../ethers-contracts"
import {getAddressInfo} from "../consts" 
import {getDefaultProvider} from "../main/helpers"
import {CHAINS, ChainId, ChainName, Network} from "../../../"
import {getNetwork, PRIVATE_KEY, isCI} from "./utils/utils";


const network: Network = getNetwork();
const ci: boolean = isCI();

const sourceChain = network == 'DEVNET' ? "ethereum" : "avalanche";
const targetChain = network == 'DEVNET' ? "bsc" : "celo";

const sourceChainId = CHAINS[sourceChain];
const targetChainId = CHAINS[targetChain];

describe("Relay Provider Test", () => {

  
  const addressInfo = getAddressInfo(sourceChain, network);
  const provider = getDefaultProvider(network, sourceChain, ci);

  // signers
  const oracleDeployer = new ethers.Wallet(PRIVATE_KEY, provider);
  const deliveryProviderAddress = addressInfo.mockDeliveryProviderAddress;
  if(!deliveryProviderAddress) throw Error("No relay provider address");
  const deliveryProvider = DeliveryProvider__factory.connect(deliveryProviderAddress, oracleDeployer);


  describe("Read Prices Correctly", () => {
    test("readPrices", async () => {
      const tokenPrice = ethers.BigNumber.from("100000");
      const gasPrice = ethers.utils.parseUnits("300", "gwei");
      
      const tokenPriceReturned = await deliveryProvider.nativeCurrencyPrice(targetChainId);
      const gasPriceReturned = await deliveryProvider.gasPrice(targetChainId);

      expect(tokenPriceReturned.toString()).toBe(tokenPrice.toString());
      expect(gasPriceReturned.toString()).toBe(gasPrice.toString());

    });
  });
});
