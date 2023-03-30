import { expect } from "chai";
import { ethers } from "ethers";
import { RelayProvider__factory } from "../../sdk/src";
import {
  ORACLE_DEPLOYER_PRIVATE_KEY,
  ChainInfo
} from "./helpers/consts";
import { init, loadChains, loadRelayProviders } from "../ts-scripts/helpers/env";

const ETHEREUM_ROOT = `${__dirname}/..`;

init()
const chains = loadChains();
const relayProviders = loadRelayProviders();

describe("Relay Provider Test", () => {
  const chain = chains[0]
  const provider = new ethers.providers.StaticJsonRpcProvider(chain.rpc);
  

  // signers
  const oracleDeployer = new ethers.Wallet(ORACLE_DEPLOYER_PRIVATE_KEY, provider);
  const relayProviderAddress = relayProviders.find((p)=>(p.chainId==chain.chainId))?.address as string

  const relayProviderAbiPath = `${ETHEREUM_ROOT}/build/RelayProvider.sol/RelayProvider.json`;
  const relayProvider = RelayProvider__factory.connect(relayProviderAddress, oracleDeployer);


  describe("Read Prices Correctly", () => {
    it("readPrices", async () => {
      const tokenPrice = ethers.BigNumber.from("100000");
      const gasPrice = ethers.BigNumber.from("300000000000")
      
      const tokenPriceReturned = await relayProvider.nativeCurrencyPrice(chain.chainId);
      const gasPriceReturned = await relayProvider.gasPrice(chain.chainId);

      expect(tokenPriceReturned.toString()).to.equal(tokenPrice.toString());
      expect(gasPriceReturned.toString()).to.equal(gasPrice.toString());
      // TODO: check getter
    });
  });
});
