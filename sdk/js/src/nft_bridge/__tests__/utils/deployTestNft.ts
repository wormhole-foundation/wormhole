import { ethers } from "ethers";
import Web3 from "web3";
import {
  NFTImplementation,
  NFTImplementation__factory,
} from "../../../ethers-contracts";
const ERC721 = require("@openzeppelin/contracts/build/contracts/ERC721PresetMinterPauserAutoId.json");

export async function deployTestNftOnEthereum(
  web3: Web3,
  signer: ethers.Wallet,
  name: string,
  symbol: string,
  uri: string,
  howMany: number
): Promise<NFTImplementation> {
  const accounts = await web3.eth.getAccounts();
  const nftContract = new web3.eth.Contract(ERC721.abi);
  let nft = await nftContract
    .deploy({
      data: ERC721.bytecode,
      arguments: [name, symbol, uri],
    })
    .send({
      from: accounts[1],
      gas: 5000000,
    });

  // The eth contracts mints tokens with sequential ids, so in order to get to a
  // specific id, we need to mint multiple nfts. We need this to test that
  // foreign ids on terra get converted to the decimal stringified form of the
  // original id.
  for (var i = 0; i < howMany; i++) {
    await nft.methods.mint(accounts[1]).send({
      from: accounts[1],
      gas: 1000000,
    });
  }

  return NFTImplementation__factory.connect(nft.options.address, signer);
}
