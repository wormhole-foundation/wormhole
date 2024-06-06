import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { describe, expect, test } from "@jest/globals";
import { ContractReceipt, ethers } from "ethers";
import {
  CHAINS,
  CONTRACTS,
  ChainId,
  ChainName,
  Network,
  ethers_relayer_contracts,
  relayer,
  tryNativeToUint8Array,
} from "../../../";
import { GovernanceEmitter, MockGuardians } from "../../../src/mock";
import { Implementation__factory } from "../../ethers-contracts";
import { getAddressInfo } from "../consts";
import { manualDelivery } from "../relayer";
import { getDefaultProvider } from "../relayer/helpers";
import { packEVMExecutionInfoV1 } from "../structs";
import {
  GOVERNANCE_EMITTER_ADDRESS,
  GUARDIAN_KEYS,
  GUARDIAN_SET_INDEX,
  PRIVATE_KEY,
  getArbitraryBytes32,
  getGuardianRPC,
  getNetwork,
  isCI,
  waitForRelay,
} from "./utils/utils";

const network: Network = getNetwork();
const ci: boolean = isCI();

const sourceChain = network == "DEVNET" ? "ethereum" : "celo";
const targetChain = network == "DEVNET" ? "bsc" : "avalanche";

const testIfDevnet = () => (network == "DEVNET" ? test : test.skip);
const testIfNotDevnet = () => (network != "DEVNET" ? test : test.skip);

type TestChain = {
  chainId: ChainId;
  name: ChainName;
  provider: ethers.providers.StaticJsonRpcProvider;
  wallet: ethers.Wallet;
  wormholeRelayerAddress: string;
  mockIntegrationAddress: string;
  wormholeRelayer: ethers_relayer_contracts.WormholeRelayer;
  mockIntegration: ethers_relayer_contracts.MockRelayerIntegration;
};

const createTestChain = (name: ChainName) => {
  const provider = getDefaultProvider(network, name, ci);
  const addressInfo = getAddressInfo(name, network);
  if (process.env.DEV) {
    // Via ir is off -> different wormhole relayer address
    addressInfo.wormholeRelayerAddress =
      "0xcC680D088586c09c3E0E099a676FA4b6e42467b4";
  }
  if (network == "MAINNET")
    addressInfo.mockIntegrationAddress =
      "0xa507Ff8D183D2BEcc9Ff9F82DFeF4b074e1d0E05";
  if (network == "MAINNET")
    addressInfo.mockDeliveryProviderAddress =
      "0x7A0a53847776f7e94Cc35742971aCb2217b0Db81";

  if (!addressInfo.wormholeRelayerAddress)
    throw Error(`No core relayer address for ${name}`);
  if (!addressInfo.mockIntegrationAddress)
    throw Error(`No mock relayer integration address for ${name}`);
  const wallet = new ethers.Wallet(PRIVATE_KEY, provider);
  const wormholeRelayer =
    ethers_relayer_contracts.WormholeRelayer__factory.connect(
      addressInfo.wormholeRelayerAddress,
      wallet
    );
  const mockIntegration =
    ethers_relayer_contracts.MockRelayerIntegration__factory.connect(
      addressInfo.mockIntegrationAddress,
      wallet
    );
  const result: TestChain = {
    chainId: CHAINS[name],
    name,
    provider,
    wallet,
    wormholeRelayerAddress: addressInfo.wormholeRelayerAddress,
    mockIntegrationAddress: addressInfo.mockIntegrationAddress,
    wormholeRelayer,
    mockIntegration,
  };
  return result;
};

const source = createTestChain(sourceChain);
const target = createTestChain(targetChain);

const myMap = new Map<ChainName, ethers.providers.Provider>();
myMap.set(sourceChain, source.provider);
myMap.set(targetChain, target.provider);
const optionalParams = {
  environment: network,
  sourceChainProvider: source.provider,
  targetChainProviders: myMap,
  wormholeRelayerAddress: source.wormholeRelayerAddress,
};
const optionalParamsTarget = {
  environment: network,
  sourceChainProvider: target.provider,
  targetChainProviders: myMap,
  wormholeRelayerAddress: target.wormholeRelayerAddress,
};

// for signing wormhole messages
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

// for generating governance wormhole messages
const governance = new GovernanceEmitter(GOVERNANCE_EMITTER_ADDRESS);

const guardianIndices = process.env.NUM_GUARDIANS
  ? [...Array(parseInt(process.env.NUM_GUARDIANS)).keys()]
  : ci
  ? [0, 1]
  : [0];

const REASONABLE_GAS_LIMIT = 500000;
const TOO_LOW_GAS_LIMIT = 10000;

const wormholeRelayerAddresses = new Map<ChainName, string>();
wormholeRelayerAddresses.set(sourceChain, source.wormholeRelayerAddress);
wormholeRelayerAddresses.set(targetChain, target.wormholeRelayerAddress);

const getStatus = async (
  txHash: string,
  _sourceChain?: ChainName,
  index?: number
): Promise<string> => {
  const info = (await relayer.getWormholeRelayerInfo(
    _sourceChain || sourceChain,
    txHash,
    {
      environment: network,
      targetChainProviders: myMap,
      sourceChainProvider: myMap.get(_sourceChain || sourceChain),
      wormholeRelayerAddresses,
    }
  )) as relayer.DeliveryInfo;
  return info.targetChainStatus.events[index ? index : 0].status;
};

const testSend = async (
  payload: string,
  sendToSourceChain?: boolean,
  notEnoughValue?: boolean
): Promise<ContractReceipt> => {
  const value = await relayer.getPrice(
    sourceChain,
    sendToSourceChain ? sourceChain : targetChain,
    notEnoughValue ? TOO_LOW_GAS_LIMIT : REASONABLE_GAS_LIMIT,
    optionalParams
  );
  !ci && console.log(`Quoted gas delivery fee: ${value}`);
  const tx = await source.mockIntegration.sendMessage(
    payload,
    sendToSourceChain ? source.chainId : target.chainId,
    notEnoughValue ? TOO_LOW_GAS_LIMIT : REASONABLE_GAS_LIMIT,
    0,
    { value, gasLimit: REASONABLE_GAS_LIMIT }
  );
  !ci && console.log(`Sent delivery request! Transaction hash ${tx.hash}`);
  await tx.wait();
  !ci && console.log("Message confirmed!");

  return tx.wait();
};

describe("Wormhole Relayer Tests", () => {
  test("Executes a Delivery Success", async () => {
    const arbitraryPayload = getArbitraryBytes32();
    !ci && console.log(`Sent message: ${arbitraryPayload}`);

    const rx = await testSend(arbitraryPayload);

    await waitForRelay();

    !ci && console.log("Checking if message was relayed");
    const message = await target.mockIntegration.getMessage();
    expect(message).toBe(arbitraryPayload);
  });

  test("Executes a Delivery Success With Additional VAAs", async () => {
    const arbitraryPayload = getArbitraryBytes32();
    !ci && console.log(`Sent message: ${arbitraryPayload}`);

    const wormhole = Implementation__factory.connect(
      CONTRACTS[network][sourceChain].core || "",
      source.wallet
    );
    const deliverySeq = await wormhole.nextSequence(source.wallet.address);
    const msgTx = await wormhole.publishMessage(0, arbitraryPayload, 200);
    await msgTx.wait();

    const value = await relayer.getPrice(
      sourceChain,
      targetChain,
      REASONABLE_GAS_LIMIT * 2,
      optionalParams
    );
    !ci && console.log(`Quoted gas delivery fee: ${value}`);

    const tx = await source.mockIntegration.sendMessageWithAdditionalVaas(
      [],
      target.chainId,
      REASONABLE_GAS_LIMIT * 2,
      0,
      [
        relayer.createVaaKey(
          source.chainId,
          Buffer.from(tryNativeToUint8Array(source.wallet.address, "ethereum")),
          deliverySeq
        ),
      ],
      { value }
    );

    !ci && console.log(`Sent tx hash: ${tx.hash}`);

    const rx = await tx.wait();

    await waitForRelay();

    !ci && console.log("Checking if message was relayed");
    const message = (await target.mockIntegration.getDeliveryData())
      .additionalVaas[0];
    const parsedMessage = await wormhole.parseVM(message);
    expect(parsedMessage.payload).toBe(arbitraryPayload);
  });

  testIfNotDevnet()(
    "Executes a Delivery Success with manual delivery",
    async () => {
      const arbitraryPayload = getArbitraryBytes32();
      !ci && console.log(`Sent message: ${arbitraryPayload}`);

      const deliverySeq = await Implementation__factory.connect(
        CONTRACTS[network][sourceChain].core || "",
        source.provider
      ).nextSequence(source.wormholeRelayerAddress);

      const rx = await testSend(arbitraryPayload, false, true);

      await waitForRelay();

      // confirm that the message was not relayed successfully
      {
        const message = await target.mockIntegration.getMessage();
        expect(message).not.toBe(arbitraryPayload);
      }
      const [value, refundPerGasUnused] = await relayer.getPriceAndRefundInfo(
        sourceChain,
        targetChain,
        REASONABLE_GAS_LIMIT,
        optionalParams
      );

      const priceInfo = await manualDelivery(
        sourceChain,
        rx.transactionHash,
        { wormholeRelayerAddresses, ...optionalParams },
        true,
        {
          newExecutionInfo: Buffer.from(
            packEVMExecutionInfoV1({
              gasLimit: ethers.BigNumber.from(REASONABLE_GAS_LIMIT),
              targetChainRefundPerGasUnused:
                ethers.BigNumber.from(refundPerGasUnused),
            }).substring(2),
            "hex"
          ),
          newReceiverValue: ethers.BigNumber.from(0),
          redeliveryHash: Buffer.from(
            ethers.utils.keccak256("0x1234").substring(2),
            "hex"
          ), // fake a redelivery
        }
      );

      !ci &&
        console.log(
          `Price: ${priceInfo.quote} of ${priceInfo.targetChain} wei`
        );

      const deliveryRx = await manualDelivery(
        sourceChain,
        rx.transactionHash,
        { wormholeRelayerAddresses, ...optionalParams },
        false,
        {
          newExecutionInfo: Buffer.from(
            packEVMExecutionInfoV1({
              gasLimit: ethers.BigNumber.from(REASONABLE_GAS_LIMIT),
              targetChainRefundPerGasUnused:
                ethers.BigNumber.from(refundPerGasUnused),
            }).substring(2),
            "hex"
          ),
          newReceiverValue: ethers.BigNumber.from(0),
          redeliveryHash: Buffer.from(
            ethers.utils.keccak256("0x1234").substring(2),
            "hex"
          ), // fake a redelivery
        },
        target.wallet
      );
      !ci && console.log("Manual delivery tx hash", deliveryRx.txHash);

      !ci && console.log("Checking if message was relayed");
      const message = await target.mockIntegration.getMessage();
      expect(message).toBe(arbitraryPayload);
    }
  );

  testIfDevnet()("Test getPrice in Typescript SDK", async () => {
    const price = await relayer.getPrice(
      sourceChain,
      targetChain,
      200000,
      optionalParams
    );
    expect(price.toString()).toBe("165000000000000000");
  });

  test("Executes a delivery with a Cross Chain Refund", async () => {
    const arbitraryPayload = getArbitraryBytes32();
    !ci && console.log(`Sent message: ${arbitraryPayload}`);
    const value = await relayer.getPrice(
      sourceChain,
      targetChain,
      REASONABLE_GAS_LIMIT,
      optionalParams
    );
    !ci && console.log(`Quoted gas delivery fee: ${value}`);
    const startingBalance = await source.wallet.getBalance();

    const tx = await relayer.sendToEvm(
      source.wallet,
      sourceChain,
      targetChain,
      target.wormholeRelayerAddress, // This is an address that exists but doesn't implement the IWormhole interface, so should result in Receiver Failure
      Buffer.from("hi!"),
      REASONABLE_GAS_LIMIT,
      { value, gasLimit: REASONABLE_GAS_LIMIT },
      optionalParams
    );
    !ci && console.log("Sent delivery request!");
    await tx.wait();
    !ci && console.log("Message confirmed!");
    const endingBalance = await source.wallet.getBalance();

    await source.provider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`

    await waitForRelay();

    const info = (await relayer.getWormholeRelayerInfo(sourceChain, tx.hash, {
      wormholeRelayerAddresses,
      ...optionalParams,
    })) as relayer.DeliveryInfo;

    await target.provider.send("anvil_mine", ["0x40"]); // 64 blocks should get the above block to `finalized`

    await waitForRelay();

    const newEndingBalance = await source.wallet.getBalance();

    !ci && console.log(`Quoted gas delivery fee: ${value}`);
    !ci &&
      console.log(
        `Cost (including gas) ${startingBalance.sub(endingBalance).toString()}`
      );
    const refund = newEndingBalance.sub(endingBalance);
    !ci && console.log(`Refund: ${refund.toString()}`);
    !ci &&
      console.log(
        `As a percentage of original value: ${newEndingBalance
          .sub(endingBalance)
          .mul(100)
          .div(value)
          .toString()}%`
      );
    !ci && console.log("Confirming refund is nonzero");
    expect(refund.gt(0)).toBe(true);
  });

  test("Executes a Receiver Failure", async () => {
    const arbitraryPayload = getArbitraryBytes32();
    !ci && console.log(`Sent message: ${arbitraryPayload}`);

    const rx = await testSend(arbitraryPayload, false, true);

    await waitForRelay();

    const message = await target.mockIntegration.getMessage();
    expect(message).not.toBe(arbitraryPayload);
  });

  test("Executes a receiver failure and then redelivery through SDK", async () => {
    const arbitraryPayload = getArbitraryBytes32();
    !ci && console.log(`Sent message: ${arbitraryPayload}`);

    const rx = await testSend(arbitraryPayload, false, true);

    await waitForRelay();

    const message = await target.mockIntegration.getMessage();
    expect(message).not.toBe(arbitraryPayload);

    const value = await relayer.getPrice(
      sourceChain,
      targetChain,
      REASONABLE_GAS_LIMIT,
      optionalParams
    );

    const info = (await relayer.getWormholeRelayerInfo(
      sourceChain,
      rx.transactionHash,
      { wormholeRelayerAddresses, ...optionalParams }
    )) as relayer.DeliveryInfo;

    !ci && console.log("Redelivering message");
    const redeliveryReceipt = await relayer.resend(
      source.wallet,
      sourceChain,
      targetChain,
      network,
      relayer.createVaaKey(
        source.chainId,
        Buffer.from(
          tryNativeToUint8Array(source.wormholeRelayerAddress, "ethereum")
        ),
        info.sourceDeliverySequenceNumber
      ),
      REASONABLE_GAS_LIMIT,
      0,
      await source.wormholeRelayer.getDefaultDeliveryProvider(),
      [getGuardianRPC(network, ci)],
      {
        value: value,
        gasLimit: REASONABLE_GAS_LIMIT,
      },
      { transport: NodeHttpTransport() },
      { wormholeRelayerAddress: source.wormholeRelayerAddress }
    );

    !ci && console.log("redelivery tx:", redeliveryReceipt.hash);

    await redeliveryReceipt.wait();

    await waitForRelay();

    !ci && console.log("Checking if message was relayed after redelivery");
    const message2 = await target.mockIntegration.getMessage();
    expect(message2).toBe(arbitraryPayload);

    //Can extend this to look for redelivery event
  });

  // GOVERNANCE TESTS

  testIfDevnet()("Governance: Test Registering Chain", async () => {
    const chain = 24;

    const currentAddress =
      await source.wormholeRelayer.getRegisteredWormholeRelayerContract(chain);
    !ci &&
      console.log(
        `For Chain ${source.chainId}, registered chain ${chain} address: ${currentAddress}`
      );

    const expectedNewRegisteredAddress =
      "0x0000000000000000000000001234567890123456789012345678901234567892";

    const timestamp = (await source.wallet.provider.getBlock("latest"))
      .timestamp;

    const firstMessage = governance.publishWormholeRelayerRegisterChain(
      timestamp,
      chain,
      expectedNewRegisteredAddress
    );
    const firstSignedVaa = guardians.addSignatures(
      firstMessage,
      guardianIndices
    );

    let tx = await source.wormholeRelayer.registerWormholeRelayerContract(
      firstSignedVaa,
      { gasLimit: REASONABLE_GAS_LIMIT }
    );
    await tx.wait();

    const newRegisteredAddress =
      await source.wormholeRelayer.getRegisteredWormholeRelayerContract(chain);

    expect(newRegisteredAddress).toBe(expectedNewRegisteredAddress);
  });

  testIfDevnet()(
    "Governance: Test Setting Default Relay Provider",
    async () => {
      const currentAddress =
        await source.wormholeRelayer.getDefaultDeliveryProvider();
      !ci &&
        console.log(
          `For Chain ${source.chainId}, default relay provider: ${currentAddress}`
        );

      const expectedNewDefaultDeliveryProvider =
        "0x1234567890123456789012345678901234567892";

      const timestamp = (await source.wallet.provider.getBlock("latest"))
        .timestamp;
      const chain = source.chainId;
      const firstMessage =
        governance.publishWormholeRelayerSetDefaultDeliveryProvider(
          timestamp,
          chain,
          expectedNewDefaultDeliveryProvider
        );
      const firstSignedVaa = guardians.addSignatures(
        firstMessage,
        guardianIndices
      );

      let tx = await source.wormholeRelayer.setDefaultDeliveryProvider(
        firstSignedVaa
      );
      await tx.wait();

      const newDefaultDeliveryProvider =
        await source.wormholeRelayer.getDefaultDeliveryProvider();

      expect(newDefaultDeliveryProvider).toBe(
        expectedNewDefaultDeliveryProvider
      );

      const inverseFirstMessage =
        governance.publishWormholeRelayerSetDefaultDeliveryProvider(
          timestamp,
          chain,
          currentAddress
        );
      const inverseFirstSignedVaa = guardians.addSignatures(
        inverseFirstMessage,
        guardianIndices
      );

      tx = await source.wormholeRelayer.setDefaultDeliveryProvider(
        inverseFirstSignedVaa
      );
      await tx.wait();

      const originalDefaultDeliveryProvider =
        await source.wormholeRelayer.getDefaultDeliveryProvider();

      expect(originalDefaultDeliveryProvider).toBe(currentAddress);
    }
  );

  testIfDevnet()("Governance: Test Upgrading Contract", async () => {
    const IMPLEMENTATION_STORAGE_SLOT =
      "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc";

    const getImplementationAddress = () =>
      source.provider.getStorageAt(
        source.wormholeRelayer.address,
        IMPLEMENTATION_STORAGE_SLOT
      );

    !ci &&
      console.log(
        `Current Implementation address: ${await getImplementationAddress()}`
      );

    const wormholeAddress = CONTRACTS[network][sourceChain].core || "";

    const newWormholeRelayerImplementationAddress = (
      await new ethers_relayer_contracts.WormholeRelayer__factory(source.wallet)
        .deploy(wormholeAddress)
        .then((x) => x.deployed())
    ).address;

    !ci && console.log(`Deployed!`);
    !ci &&
      console.log(
        `New core relayer implementation: ${newWormholeRelayerImplementationAddress}`
      );

    const timestamp = (await source.wallet.provider.getBlock("latest"))
      .timestamp;
    const chain = source.chainId;
    const firstMessage = governance.publishWormholeRelayerUpgradeContract(
      timestamp,
      chain,
      newWormholeRelayerImplementationAddress
    );
    const firstSignedVaa = guardians.addSignatures(
      firstMessage,
      guardianIndices
    );

    let tx = await source.wormholeRelayer.submitContractUpgrade(firstSignedVaa);
    await tx.wait();

    expect(
      ethers.utils.getAddress((await getImplementationAddress()).substring(26))
    ).toBe(ethers.utils.getAddress(newWormholeRelayerImplementationAddress));
  });

  testIfNotDevnet()("Checks the status of a message", async () => {
    const txHash =
      "0xa75e4100240e9b498a48fa29de32c9e62ec241bf4071a3c93fde0df5de53c507";
    const mySourceChain: ChainName = "celo";
    const environment: Network = "TESTNET";

    const info = await relayer.getWormholeRelayerInfo(mySourceChain, txHash, {
      environment,
    });
    !ci && console.log(info.stringified);
  });

  testIfNotDevnet()("Tests custom manual delivery", async () => {
    const txHash =
      "0xc57d12cc789e4e9fa50d496cea62c2a0f11a7557c8adf42b3420e0585ba1f911";
    const mySourceChain: ChainName = "arbitrum";
    const targetProvider = undefined;
    const environment: Network = "TESTNET";

    const info = await relayer.getWormholeRelayerInfo(mySourceChain, txHash, {
      environment,
    });
    !ci && console.log(info.stringified);

    const priceInfo = await manualDelivery(
      mySourceChain,
      txHash,
      { environment },
      true
    );
    !ci && console.log(`Price info: ${JSON.stringify(priceInfo)}`);

    const signer = new ethers.Wallet(
      PRIVATE_KEY,
      targetProvider
        ? new ethers.providers.JsonRpcProvider(targetProvider)
        : getDefaultProvider(environment, priceInfo.targetChain)
    );

    !ci &&
      console.log(
        `Price: ${ethers.utils.formatEther(priceInfo.quote)} of ${
          priceInfo.targetChain
        } currency`
      );
    const balance = await signer.getBalance();
    !ci &&
      console.log(
        `My balance: ${ethers.utils.formatEther(balance)} of ${
          priceInfo.targetChain
        } currency`
      );

    const deliveryRx = await manualDelivery(
      mySourceChain,
      txHash,
      { environment },
      false,
      undefined,
      signer
    );
    !ci && console.log("Manual delivery tx hash", deliveryRx.txHash);
  });
});

function sleep(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(() => r(), ms));
}
