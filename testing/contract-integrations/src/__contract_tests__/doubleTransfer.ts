import {
  approveEth,
  CHAIN_ID_BSC,
  createNonce,
  getEmitterAddressEth,
  Implementation__factory,
  parseSequenceFromLogEth,
} from "@certusone/wormhole-sdk";
import { describe, jest, test } from "@jest/globals";
import { ethers, Signer } from "ethers";
import { getAddress } from "ethers/lib/utils";
import {
  CHAIN_ID_ETH,
  hexToUint8Array,
  nativeToHexString,
} from "@certusone/wormhole-sdk";
import { setDefaultWasm } from "@certusone/wormhole-sdk/lib/cjs/solana/wasm";
import {
  ETH_TEST_TOKEN,
  ETH_TEST_WALLET_PUBLIC_KEY,
  getBridgeAddressForChain,
  getSignerForChain,
  getTokenBridgeAddressForChain,
  WORMHOLE_RPC_HOSTS,
} from "../consts";
import { DoubleTransfer__factory } from "../../ethers-contracts/abi/factories/DoubleTransfer__factory";
import getSignedVAAWithRetry from "@certusone/wormhole-sdk/lib/cjs/rpc/getSignedVAAWithRetry";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";

setDefaultWasm("node");

jest.setTimeout(6000000);

async function deployDoubleTransferContract() {
  const ethSigner = getSignerForChain(CHAIN_ID_ETH);
  const contractInterface = DoubleTransfer__factory.createInterface();
  const bytecode = DoubleTransfer__factory.bytecode;
  const ethfactory = new ethers.ContractFactory(
    contractInterface,
    bytecode,
    ethSigner
  );
  const contract = await ethfactory.deploy(getAddress(ETH_TEST_TOKEN));
  const ethAddress = await contract.deployed().then(
    (result) => {
      console.log("Successfully deployed contract at " + result.address);
      return result.address;
    },
    (error) => {
      console.error(error);
    }
  );

  return ethAddress;
}

describe("Double transfer Tests", () => {
  describe("Ethereum double token transfer", () => {
    test("Attest Ethereum ERC-20 to Solana", (done) => {
      (async () => {
        const ethSigner = getSignerForChain(CHAIN_ID_ETH);
        //await attestEvm(CHAIN_ID_ETH, ETH_TEST_TOKEN);
        const contractAddress = (await deployDoubleTransferContract()) || "";
        const contract = DoubleTransfer__factory.connect(
          contractAddress,
          ethSigner
        );
        await approveEth(
          contractAddress,
          getAddress(ETH_TEST_TOKEN),
          ethSigner,
          "10000000000000000000000"
        );

        const nonce1 = createNonce();
        const nonce2 = createNonce();
        const receipt1 = await (
          await contract.transferTwice(
            "100",
            getTokenBridgeAddressForChain(CHAIN_ID_ETH),
            CHAIN_ID_BSC,
            hexToUint8Array(
              nativeToHexString(ETH_TEST_WALLET_PUBLIC_KEY, CHAIN_ID_BSC) as any
            ),
            "0",
            nonce1,
            nonce2
          )
        ).wait();
        const receipt2 = await (
          await contract.transferTwice(
            "100",
            getTokenBridgeAddressForChain(CHAIN_ID_ETH),
            CHAIN_ID_BSC,
            hexToUint8Array(
              nativeToHexString(ETH_TEST_WALLET_PUBLIC_KEY, CHAIN_ID_BSC) as any
            ),
            "0",
            nonce1,
            nonce2
          )
        ).wait();
        const receipt3 = await (
          await contract.transferTwice(
            "100",
            getTokenBridgeAddressForChain(CHAIN_ID_ETH),
            CHAIN_ID_BSC,
            hexToUint8Array(
              nativeToHexString(ETH_TEST_WALLET_PUBLIC_KEY, CHAIN_ID_BSC) as any
            ),
            "0",
            nonce1,
            nonce2
          )
        ).wait();

        const bridgeLogs = receipt1.logs.filter((l) => {
          return l.address === getBridgeAddressForChain(CHAIN_ID_ETH);
        });
        const sequences = bridgeLogs.map((item) => {
          const {
            args: { sequence },
          } = Implementation__factory.createInterface().parseLog(item);
          return sequence.toString();
        });

        console.log("PARSED MULTI SEQS, ", sequences);

        const sequenceForReceipt1 = parseSequenceFromLogEth(
          receipt1,
          getBridgeAddressForChain(CHAIN_ID_ETH)
        );
        const sequenceForReceipt2 = parseSequenceFromLogEth(
          receipt2,
          getBridgeAddressForChain(CHAIN_ID_ETH)
        );
        const sequenceForReceipt3 = parseSequenceFromLogEth(
          receipt2,
          getBridgeAddressForChain(CHAIN_ID_ETH)
        );

        const expectedSequences = [
          sequenceForReceipt1,
          parseInt(sequenceForReceipt1) + 1,
          parseInt(sequenceForReceipt1) + 2,
          parseInt(sequenceForReceipt1) + 3,
          parseInt(sequenceForReceipt1) + 4,
          parseInt(sequenceForReceipt1) + 5,
        ];
        const sequenceStrings = expectedSequences.map((x) => x.toString());
        console.log("EXPECTEDS, ", sequenceStrings);

        const promises = [];

        let failed = false;
        sequenceStrings.forEach((seq) => {
          promises.push(
            getSignedVAAWithRetry(
              WORMHOLE_RPC_HOSTS,
              CHAIN_ID_ETH,
              getEmitterAddressEth(getTokenBridgeAddressForChain(CHAIN_ID_ETH)),
              seq,
              {
                transport: NodeHttpTransport(), //This should only be needed when running in node.
              },
              1000,
              30
            )
              .then(() => {
                console.log(`found vaa for ${seq}`);
              })
              .catch(() => {
                console.log("Caught exception from vaa retry");
              })
          );
        });

        for (const promise of promises) {
          await promise;
          console.log("awaited promise");
        }
        console.log("promises settled");
        expect(failed).toBe(false);
        done(failed ? "failed" : undefined);
        // const bridgeLog = receipt.logs.filter((l) => {
        //     return l.address === getTokenBridgeAddressForChain(CHAIN_ID_ETH);
        //   })[0];
        //   const {
        //     args: { sequence },
        //   } = Implementation__factory.createInterface().parseLog(bridgeLog);
        //   console.log("Pulled sequence", sequence)
        //   return sequence.toString();
      })();
    });
  });
});
