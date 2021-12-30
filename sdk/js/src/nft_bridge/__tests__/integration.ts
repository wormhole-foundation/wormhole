import { BlockTxBroadcastResult, Coins, LCDClient, MnemonicKey, Msg, MsgExecuteContract, StdFee, StdTx, TxInfo, Wallet } from "@terra-money/terra.js";
import axios from "axios";
import Web3 from 'web3';
import { Contract } from 'web3-eth-contract';
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
  TERRA_NODE_URL,
  TERRA_CHAIN_ID,
  TERRA_PRIVATE_KEY,
} from "./consts";
import { getForeignAssetTerra } from "../../token_bridge";
const ERC721 = require("@openzeppelin/contracts/build/contracts/ERC721PresetMinterPauserAutoId.json");

setDefaultWasm("node");

jest.setTimeout(60000);

type Address = string;

const lcd = new LCDClient({
  URL: TERRA_NODE_URL,
  chainID: TERRA_CHAIN_ID,
});
const terraWallet: Wallet = lcd.wallet(new MnemonicKey({
  mnemonic: TERRA_PRIVATE_KEY,
}));

const web3 = new Web3(ETH_NODE_URL);

const provider = new ethers.providers.WebSocketProvider(ETH_NODE_URL);
const signer = new ethers.Wallet(ETH_PRIVATE_KEY, provider);

afterAll(() => {
  provider.destroy();
});

async function deployNFTOnEth(): Promise<Contract> {
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


  await nft.methods.mint(accounts[1]).send({
    from: accounts[1],
    gas: 1000000,
  });
  return nft;
}

async function deployNFTOnTerra(): Promise<Address> {
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
  return address;
}

describe("Integration Tests", () => {
  describe("Ethereum to Terra", () => {
    test("Send Ethereum ERC-721 to Terra and back", (done) => {
      (async () => {
        try {
          const erc721 = await deployNFTOnEth();
          let signedVAA = await waitUntilEthTxObserved(await _transferFromEth(erc721.options.address, 0));
          (await expectReceivedOnTerra(signedVAA)).toBe(false);
          await _redeemOnTerra(signedVAA);
          (await expectReceivedOnTerra(signedVAA)).toBe(true);

          // Check we have the NFT we were expecting
          const terra_addr = await getForeignAssetTerra(TERRA_NFT_BRIDGE_ADDRESS, lcd, CHAIN_ID_ETH,
            hexToUint8Array(
              nativeToHexString(erc721.options.address, CHAIN_ID_ETH) || ""
            ));
          if (!terra_addr) {
            throw new Error("Terra address is null");
          }

          const info: any = await lcd.wasm.contractQuery(terra_addr, { nft_info: { token_id: "0" } });
          expect(info.token_uri).toBe("https://cloudflare-ipfs.com/ipfs/QmeSjSinHpPnmXmspMjwiXyN6zS4E9zccariGR3jxcaWtq/0");

          const ownerInfo: any = await lcd.wasm.contractQuery(terra_addr, { owner_of: { token_id: "0" } });
          expect(ownerInfo.owner).toBe(terraWallet.key.accAddress);

          let ownerEth = await erc721.methods.ownerOf(0).call();
          expect(ownerEth).not.toBe(signer.address);

          // Send back the NFT to ethereum
          signedVAA = await waitUntilTerraTxObserved(await _transferFromTerra(terra_addr, "0"));
          (await expectReceivedOnEth(signedVAA)).toBe(false);
          await _redeemOnEth(signedVAA);
          (await expectReceivedOnEth(signedVAA)).toBe(true);

          ownerEth = await erc721.methods.ownerOf(0).call();
          expect(ownerEth).toBe(signer.address);

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
          // transfer NFT
          const signedVAA = await waitUntilTerraTxObserved(await _transferFromTerra(cw721, "0"));
          (await expectReceivedOnEth(signedVAA)).toBe(false);
          await _redeemOnEth(signedVAA);
          (await expectReceivedOnEth(signedVAA)).toBe(true);
          done();
        } catch (e) {
          console.error(e);
          done(`An error occured while trying to transfer from Ethereum to Solana: ${e}`);
        }
      })();
    });
  })
});

////////////////////////////////////////////////////////////////////////////////
// Utils

async function getGasPrices() {
  return axios
    .get(TERRA_GAS_PRICES_URL)
    .then((result) => result.data);
}

async function estimateTerraFee(gasPrices: Coins.Input, msgs: Msg[]): Promise<StdFee> {
  const feeEstimate = await lcd.tx.estimateFee(
    terraWallet.key.accAddress,
    msgs,
    {
      memo: "localhost",
      feeDenoms: ["uluna"],
      gasPrices,
    }
  );
  return feeEstimate;
}

async function waitUntilTerraTxObserved(txresult: BlockTxBroadcastResult): Promise<Uint8Array> {
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
  return signedVAA;
}

async function waitUntilEthTxObserved(receipt: ethers.ContractReceipt): Promise<Uint8Array> {
  // get the sequence from the logs (needed to fetch the vaa)
  let sequence = parseSequenceFromLogEth(
    receipt,
    ETH_CORE_BRIDGE_ADDRESS
  );
  let emitterAddress = getEmitterAddressEth(ETH_NFT_BRIDGE_ADDRESS);
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
  return signedVAA;
}

async function mint_cw721(contract_address: string, token_id: number, token_uri: any): Promise<void> {
  await terraWallet
    .createAndSignTx({
      msgs: [
        new MsgExecuteContract(
          terraWallet.key.accAddress,
          contract_address,
          {
            mint: {
              token_id: token_id.toString(),
              owner: terraWallet.key.accAddress,
              token_uri: token_uri,
            },
          },
          { uluna: 1000 }
        ),
      ],
      memo: "",
      fee: new StdFee(2000000, {
        uluna: "100000",
      }),
    })
    .then((tx) => lcd.tx.broadcast(tx));
}

async function waitForTerraExecution(txHash: string): Promise<TxInfo> {
  let info: TxInfo | undefined = undefined;
  while (!info) {
    await new Promise((resolve) => setTimeout(resolve, 1000));
    try {
      info = await lcd.tx.txInfo(txHash);
    } catch (e) {
      console.error(e);
    }
  }
  if (info.code !== undefined) {
    // error code
    throw new Error(
      `Tx ${txHash}: error code ${info.code}: ${info.raw_log}`
    );
  }
  return info;
}

async function expectReceivedOnTerra(signedVAA: Uint8Array) {
  return expect(
    await getIsTransferCompletedTerra(
      TERRA_NFT_BRIDGE_ADDRESS,
      signedVAA,
      terraWallet.key.accAddress,
      lcd,
      TERRA_GAS_PRICES_URL
    )
  );
}

async function expectReceivedOnEth(signedVAA: Uint8Array) {
  return expect(
    await getIsTransferCompletedEth(
      ETH_NFT_BRIDGE_ADDRESS,
      provider,
      signedVAA,
    )
  );
}

async function _transferFromEth(erc721: string, token_id: number): Promise<ethers.ContractReceipt> {
  return transferFromEth(
    ETH_NFT_BRIDGE_ADDRESS,
    signer,
    erc721,
    token_id,
    CHAIN_ID_TERRA,
    hexToUint8Array(
      nativeToHexString(terraWallet.key.accAddress, CHAIN_ID_TERRA) || ""
    ));
}

async function _transferFromTerra(terra_addr: string, token_id: string): Promise<BlockTxBroadcastResult> {
  const gasPrices = await getGasPrices();
  const msgs = await transferFromTerra(
    terraWallet.key.accAddress,
    TERRA_NFT_BRIDGE_ADDRESS,
    terra_addr,
    token_id,
    CHAIN_ID_ETH,
    hexToUint8Array(
      nativeToHexString(signer.address, CHAIN_ID_ETH) || ""
    ));
  const tx = await terraWallet.createAndSignTx({
    msgs: msgs,
    memo: "test",
    feeDenoms: ["uluna"],
    gasPrices,
    fee: await estimateTerraFee(gasPrices, msgs)
  });
  return lcd.tx.broadcast(tx);
}

async function _redeemOnEth(signedVAA: Uint8Array): Promise<any> {
  return redeemOnEth(
    ETH_NFT_BRIDGE_ADDRESS,
    signer,
    signedVAA
  );
}

async function _redeemOnTerra(signedVAA: Uint8Array): Promise<BlockTxBroadcastResult> {
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
  return lcd.tx.broadcast(tx);
}
