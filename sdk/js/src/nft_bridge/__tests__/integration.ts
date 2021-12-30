import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { describe, expect, jest, test } from "@jest/globals";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import {
  MsgInstantiateContract,
} from "@terra-money/terra.js";
import { ethers } from "ethers";
import {
  CHAIN_ID_ETH,
  CHAIN_ID_TERRA,
  getEmitterAddressEth,
  getEmitterAddressTerra,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogEth,
  parseSequenceFromLogTerra,
} from "../..";
import { estimateTerraFee, getGasPrices, lcd, mint_cw721, terraWallet, waitForTerraExecution, web3 } from './utils';
import {
  redeemOnEth,
  redeemOnTerra,
  transferFromEth,
  transferFromTerra,
  getIsTransferCompletedEth,
  getIsTransferCompletedTerra,
} from "../";
import getSignedVAAWithRetry from "../../rpc/getSignedVAAWithRetry";
import { setDefaultWasm } from "../../solana/wasm";
import {
  ETH_CORE_BRIDGE_ADDRESS,
  ETH_NODE_URL,
  ETH_PRIVATE_KEY,
  ETH_NFT_BRIDGE_ADDRESS,
  TERRA_GAS_PRICES_URL,
  TERRA_NFT_BRIDGE_ADDRESS,
  WORMHOLE_RPC_HOSTS,
  TERRA_CW721_CODE_ID,
} from "./consts";
const ERC721 = require("@openzeppelin/contracts/build/contracts/ERC721PresetMinterPauserAutoId.json");

setDefaultWasm("node");

jest.setTimeout(60000);

async function deployNFTOnEth(): Promise<string> {
  const accounts = await web3.eth.getAccounts();
  const nftContract = new web3.eth.Contract(ERC721.abi);
  let nft = await nftContract.deploy({
    data: ERC721.bytecode,
    arguments: [
      "Not an APE 🐒",
      "APE🐒",
      "https://cloudflare-ipfs.com/ipfs/QmeSjSinHpPnmXmspMjwiXyN6zS4E9zccariGR3jxcaWtq/"
    ]
  }).send({
    from: accounts[1],
    gas: 5000000,
  });


  // const nft = new web3.eth.Contract(ERC721.abi, "0x5b9b42d6e4B2e4Bf8d42Eba32D46918e10899B66");
  await nft.methods.mint(accounts[1]).send({
    from: accounts[1],
    gas: 1000000,
  });
  await nft.methods.mint(accounts[1]).send({
    from: accounts[1],
    gas: 1000000,
  });
  // const nftAddress = (nft as any)._address as string;
  return nft.options.address;
}

async function deployNFTOnTerra(): Promise<string> {
  var address: any;
  await terraWallet
    .createAndSignTx({
      msgs: [
        new MsgInstantiateContract(
          terraWallet.key.accAddress,
          terraWallet.key.accAddress,
          TERRA_CW721_CODE_ID,
          {
            name: "INTEGRATION TEST NFT",
            symbol: "INT",
            minter: terraWallet.key.accAddress,
          }
        ),
      ],
      memo: "",
    })
    .then((tx) => lcd.tx.broadcast(tx))
    .then((rs) => {
      const match = /"contract_address","value":"([^"]+)/gm.exec(rs.raw_log);
      if (match) {
        address = match[1];
      }
    });
  await mint_cw721(address, 0, 'https://ixmfkhnh2o4keek2457f2v2iw47cugsx23eynlcfpstxihsay7nq.arweave.net/RdhVHafTuKIRWud-XVdItz4qGlfWyYasRXyndB5Ax9s/');
  await mint_cw721(address, 1, 'https://portal.neondistrict.io/api/getNft/158456327500392944014123206890');
  return address;
}

describe("Integration Tests", () => {
  describe("Ethereum to Terra", () => {
    test("Send Ethereum ERC-721 to Terra", (done) => {
      (async () => {
        try {
          const erc721 = await deployNFTOnEth();
          // create a signer for Eth
          const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          // transfer NFT
          const receipt = await transferFromEth(
            ETH_NFT_BRIDGE_ADDRESS,
            signer,
            erc721,
            0,
            CHAIN_ID_TERRA,
            hexToUint8Array(
              nativeToHexString(terraWallet.key.accAddress, CHAIN_ID_TERRA) || ""
            ));
          // get the sequence from the logs (needed to fetch the vaa)
          const sequence = parseSequenceFromLogEth(
            receipt,
            ETH_CORE_BRIDGE_ADDRESS
          );
          const emitterAddress = getEmitterAddressEth(ETH_NFT_BRIDGE_ADDRESS);
          // poll until the guardian(s) witness and sign the vaa
          const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_ETH,
            emitterAddress,
            sequence,
            {
              transport: NodeHttpTransport(),
            }
          );
          expect(
            await getIsTransferCompletedTerra(
              TERRA_NFT_BRIDGE_ADDRESS,
              signedVAA,
              terraWallet.key.accAddress,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(false);
          const msg = await redeemOnTerra(
            TERRA_NFT_BRIDGE_ADDRESS,
            terraWallet.key.accAddress,
            signedVAA
          );
          const gasPrices = await getGasPrices();
          const tx = await terraWallet.createAndSignTx({
            msgs: [msg],
            memo: "localhost",
            feeDenoms: ["uluna"],
            gasPrices,
            fee: await estimateTerraFee(gasPrices, [msg]),
          });
          await lcd.tx.broadcast(tx);
          expect(
            await getIsTransferCompletedTerra(
              TERRA_NFT_BRIDGE_ADDRESS,
              signedVAA,
              terraWallet.key.accAddress,
              lcd,
              TERRA_GAS_PRICES_URL
            )
          ).toBe(true);
          provider.destroy();
          done();
        } catch (e) {
          console.error(e);
          done(`An error occured while trying to transfer from Ethereum to Solana: ${e}`);
        }
      })();
    });
    test("Send Terra CW721 to Ethereum", (done) => {
      (async () => {
        try {
          const cw721 = await deployNFTOnTerra();
          // create a signer for Eth
          const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
          const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);
          // transfer NFT
          const msgs = await transferFromTerra(
            terraWallet.key.accAddress,
            TERRA_NFT_BRIDGE_ADDRESS,
            cw721,
            "0",
            CHAIN_ID_ETH,
            hexToUint8Array(
              nativeToHexString(await signer.getAddress(), CHAIN_ID_ETH) || ""
            ));
          const gasPrices = await getGasPrices();
          let tx = await terraWallet.createAndSignTx({
            msgs: msgs,
            memo: "test",
            feeDenoms: ["uluna"],
            gasPrices,
            fee: await estimateTerraFee(gasPrices, msgs)
          });
          const txresult = await lcd.tx.broadcast(tx);
          // get the sequence from the logs (needed to fetch the vaa)
          const info = await waitForTerraExecution(txresult.txhash);
          const sequence = parseSequenceFromLogTerra(info);
          const emitterAddress = await getEmitterAddressTerra(TERRA_NFT_BRIDGE_ADDRESS);
          // poll until the guardian(s) witness and sign the vaa
          const { vaaBytes: signedVAA } = await getSignedVAAWithRetry(
            WORMHOLE_RPC_HOSTS,
            CHAIN_ID_TERRA,
            emitterAddress,
            sequence,
            {
              transport: NodeHttpTransport(),
            }
          );
          expect(
            await getIsTransferCompletedEth(
              ETH_NFT_BRIDGE_ADDRESS,
              provider,
              signedVAA,
            )
          ).toBe(false);
          await redeemOnEth(
            ETH_NFT_BRIDGE_ADDRESS,
            signer,
            signedVAA
          );
          expect(
            await getIsTransferCompletedEth(
              ETH_NFT_BRIDGE_ADDRESS,
              provider,
              signedVAA,
            )
          ).toBe(true);
          provider.destroy();
          done();
        } catch (e) {
          console.error(e);
          done(`An error occured while trying to transfer from Ethereum to Solana: ${e}`);
        }
      })();
    });
  })
});
