import { expect } from "chai";
import { ethers } from "ethers";
import { ChainInfo, RELAYER_DEPLOYER_PRIVATE_KEY } from "./helpers/consts";
import { generateRandomString, waitForRelay } from "./helpers/utils";
import {
  init,
  loadChains,
  loadCoreRelayers,
  loadGuardianRpc,
  loadMockIntegrations,
  loadPrivateKey,
} from "../../ts-scripts/relayer/helpers/env";
import {
  relayer,
  ethers_contracts,
  tryNativeToUint8Array,
  Network,
} from "../../../sdk/js/src";
import { MockRelayerIntegration } from "../../ethers-contracts";

const ETHEREUM_ROOT = `${__dirname}/..`;

const env = init();
const chains = loadChains();
const coreRelayers = loadCoreRelayers();
const mockIntegrations = loadMockIntegrations();
let PRIVATE_KEY = RELAYER_DEPLOYER_PRIVATE_KEY;
try {
  PRIVATE_KEY = loadPrivateKey();
} catch {}

const environment: Network =
  env == "testnet" ? "TESTNET" : env == "mainnet" ? "MAINNET" : "DEVNET";

const getWormholeSequenceNumber = (
  rx: ethers.providers.TransactionReceipt,
  wormholeAddress: string
) => {
  return Number(
    rx.logs
      .find(
        (logentry: ethers.providers.Log) => logentry.address == wormholeAddress
      )
      ?.data?.substring(0, 16) || 0
  );
};

describe("Core Relayer Integration Test - Two Chains", () => {
  // signers

  const sourceChain = chains.find(
    (c) => c.chainId == (env == "testnet" ? 6 : 2)
  ) as ChainInfo;
  const targetChain = chains.find(
    (c) => c.chainId == (env == "testnet" ? 14 : 4)
  ) as ChainInfo;

  const providerSource = new ethers.providers.StaticJsonRpcProvider(
    sourceChain.rpc
  );
  const providerTarget = new ethers.providers.StaticJsonRpcProvider(
    targetChain.rpc
  );

  const walletSource = new ethers.Wallet(PRIVATE_KEY, providerSource);
  const walletTarget = new ethers.Wallet(PRIVATE_KEY, providerTarget);

  const sourceCoreRelayerAddress = coreRelayers.find(
    (p) => p.chainId == sourceChain.chainId
  )?.address as string;
  const sourceMockIntegrationAddress = mockIntegrations.find(
    (p) => p.chainId == sourceChain.chainId
  )?.address as string;
  const targetCoreRelayerAddress = coreRelayers.find(
    (p) => p.chainId == targetChain.chainId
  )?.address as string;
  const targetMockIntegrationAddress = mockIntegrations.find(
    (p) => p.chainId == targetChain.chainId
  )?.address as string;

  const sourceCoreRelayer = ethers_contracts.CoreRelayer__factory.connect(
    sourceCoreRelayerAddress,
    walletSource
  );
  const sourceMockIntegration = ethers_contracts.MockRelayerIntegration__factory.connect(
    sourceMockIntegrationAddress,
    walletSource
  );
  const targetCoreRelayer = ethers_contracts.CoreRelayer__factory.connect(
    targetCoreRelayerAddress,
    walletTarget
  );
  const targetMockIntegration = ethers_contracts.MockRelayerIntegration__factory.connect(
    targetMockIntegrationAddress,
    walletTarget
  );

  it("Executes a delivery", async () => {
    const arbitraryPayload = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload}`);
    const value = await sourceCoreRelayer.quoteGas(
      targetChain.chainId,
      500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    console.log(`Quoted gas delivery fee: ${value}`);
    const tx = await sourceMockIntegration.sendMessage(
      arbitraryPayload,
      targetChain.chainId,
      targetMockIntegrationAddress,
      { value, gasLimit: 500000 }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    await waitForRelay();

    console.log("Checking if message was relayed");
    const message = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload}`);
    console.log(`Received message: ${message}`);
    expect(message).to.equal(arbitraryPayload);
  });

  it("Executes a forward", async () => {
    const arbitraryPayload1 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    const arbitraryPayload2 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload1}`);
    const value = await sourceCoreRelayer.quoteGas(
      targetChain.chainId,
      500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    const extraForwardingValue = await targetCoreRelayer.quoteGas(
      sourceChain.chainId,
      800000,
      await targetCoreRelayer.getDefaultRelayProvider()
    );
    console.log(`Quoted gas delivery fee: ${value.add(extraForwardingValue)}`);

    const furtherInstructions: MockRelayerIntegration.FurtherInstructionsStruct = {
      keepSending: true,
      newMessages: [arbitraryPayload2, "0x00"],
      chains: [sourceChain.chainId],
      gasLimits: [500000],
    };
    const tx = await sourceMockIntegration.sendMessagesWithFurtherInstructions(
      [arbitraryPayload1],
      furtherInstructions,
      [targetChain.chainId],
      [value.add(extraForwardingValue)],
      { value: value.add(extraForwardingValue), gasLimit: 500000 }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    await waitForRelay(2);

    console.log("Checking if message was relayed");
    const message1 = await targetMockIntegration.getMessage();
    console.log(
      `Sent message: ${arbitraryPayload1} (expecting ${arbitraryPayload2} from forward)`
    );
    console.log(`Received message on target: ${message1}`);
    expect(message1).to.equal(arbitraryPayload1);

    console.log("Checking if forward message was relayed back");
    const message2 = await sourceMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload2}`);
    console.log(`Received message on source: ${message2}`);
    expect(message2).to.equal(arbitraryPayload2);

    let info: relayer.DeliveryInfo = (await relayer.getWormholeRelayerInfo(
      sourceChain.chainId,
      tx.hash,
      { environment: environment }
    )) as DeliveryInfo;
    let status = info.targetChainStatus.events[0].status;
    expect(status).to.equal("Forward Request Success");
  });

  it("Executes a multidelivery", async () => {
    const arbitraryPayload1 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload1}`);
    const value1 = await sourceCoreRelayer.quoteGas(
      sourceChain.chainId,
      500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    const value2 = await sourceCoreRelayer.quoteGas(
      targetChain.chainId,
      500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    console.log(`Quoted gas delivery fee: ${value1.add(value2)}`);

    const furtherInstructions: MockRelayerIntegration.FurtherInstructionsStruct = {
      keepSending: false,
      newMessages: [],
      chains: [],
      gasLimits: [],
    };
    const tx = await sourceMockIntegration.sendMessagesWithFurtherInstructions(
      [arbitraryPayload1],
      furtherInstructions,
      [sourceChain.chainId, targetChain.chainId],
      [value1, value2],
      { value: value1.add(value2), gasLimit: 500000 }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    await waitForRelay();

    console.log("Checking if first message was relayed");
    const message1 = await sourceMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload1}`);
    console.log(`Received message: ${message1}`);
    expect(message1).to.equal(arbitraryPayload1);

    console.log("Checking if second message was relayed");
    const message2 = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload1}`);
    console.log(`Received message: ${message2}`);
    expect(message2).to.equal(arbitraryPayload1);
  });
  it("Executes a multiforward", async () => {
    const arbitraryPayload1 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    const arbitraryPayload2 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload1}`);
    const value1 = await sourceCoreRelayer.quoteGas(
      targetChain.chainId,
      500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    const value2 = await targetCoreRelayer.quoteGas(
      sourceChain.chainId,
      500000,
      await targetCoreRelayer.getDefaultRelayProvider()
    );
    const value3 = await targetCoreRelayer.quoteGas(
      targetChain.chainId,
      500000,
      await targetCoreRelayer.getDefaultRelayProvider()
    );
    const payment = value1
      .mul(2)
      .add(value2)
      .add(value3)
      .mul(105)
      .div(100)
      .add(1);
    console.log(`Quoted gas delivery fee: ${payment}`);

    const furtherInstructions: MockRelayerIntegration.FurtherInstructionsStruct = {
      keepSending: true,
      newMessages: [arbitraryPayload2, "0x00"],
      chains: [sourceChain.chainId, targetChain.chainId],
      gasLimits: [500000, 500000],
    };
    const tx = await sourceMockIntegration.sendMessagesWithFurtherInstructions(
      [arbitraryPayload1],
      furtherInstructions,
      [targetChain.chainId],
      [payment],
      { value: payment, gasLimit: 800000 }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    await waitForRelay(2);

    console.log("Checking if first forward was relayed");
    const message1 = await sourceMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload2}`);
    console.log(`Received message: ${message1}`);
    expect(message1).to.equal(arbitraryPayload2);

    console.log("Checking if second forward was relayed");
    const message2 = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload2}`);
    console.log(`Received message: ${message2}`);
    expect(message2).to.equal(arbitraryPayload2);
  });

  it("Executes a delivery that results in a forward failure", async () => {
    const arbitraryPayload1 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    const arbitraryPayload2 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload1}`);
    const value = await sourceCoreRelayer.quoteGas(
      targetChain.chainId,
      500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    const notEnoughExtraForwardingValue = await targetCoreRelayer.quoteGas(
      sourceChain.chainId,
      10000,
      await targetCoreRelayer.getDefaultRelayProvider()
    );
    const enoughExtraForwardingValue = await targetCoreRelayer.quoteGas(
      sourceChain.chainId,
      500000,
      await targetCoreRelayer.getDefaultRelayProvider()
    );
    console.log(
      `Quoted gas delivery fee: ${value.add(notEnoughExtraForwardingValue)}`
    );

    const furtherInstructions: MockRelayerIntegration.FurtherInstructionsStruct = {
      keepSending: true,
      newMessages: [arbitraryPayload2, "0x00"],
      chains: [sourceChain.chainId],
      gasLimits: [500000],
    };
    const tx = await sourceMockIntegration.sendMessagesWithFurtherInstructions(
      [arbitraryPayload1],
      furtherInstructions,
      [targetChain.chainId],
      [value.add(notEnoughExtraForwardingValue)],
      { value: value.add(notEnoughExtraForwardingValue), gasLimit: 500000 }
    );

    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    await waitForRelay();

    console.log("Checking if message was relayed (it shouldn't have been!");
    const message1 = await targetMockIntegration.getMessage();
    console.log(
      `Sent message: ${arbitraryPayload1} (expecting ${arbitraryPayload2} from forward)`
    );
    console.log(`Received message on target: ${message1}`);
    expect(message1).to.not.equal(arbitraryPayload1);

    console.log(
      "Checking if forward message was relayed back (it shouldn't have been!)"
    );
    const message2 = await sourceMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload2}`);
    console.log(`Received message on source: ${message2}`);
    expect(message2).to.not.equal(arbitraryPayload2);

    let info: relayer.DeliveryInfo = (await relayer.getWormholeRelayerInfo(
      sourceChain.chainId,
      tx.hash,
      {
        environment: environment,
      }
    )) as DeliveryInfo;
    let status = info.targetChainStatus.events[0].status;
    expect(status).to.equal("Forward Request Failure");
    console.log(relayer.stringifyWormholeRelayerInfo(info));
  });

  it("Tests the Typescript SDK with a Delivery Success", async () => {
    const arbitraryPayload = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload}`);
    const value = await sourceCoreRelayer.quoteGas(
      targetChain.chainId,
      500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    console.log(`Quoted gas delivery fee: ${value}`);
    const tx = await sourceMockIntegration.sendMessage(
      arbitraryPayload,
      targetChain.chainId,
      targetMockIntegrationAddress,
      { value, gasLimit: 500000 }
    );

    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    await waitForRelay();

    const message = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload}`);
    console.log(`Received message: ${message}`);
    expect(message).to.equal(arbitraryPayload);

    console.log("Checking status using SDK");
    const info = (await relayer.getWormholeRelayerInfo(
      sourceChain.chainId,
      tx.hash,
      { environment: environment }
    )) as relayer.DeliveryInfo;
    const status = info.targetChainStatus.events[0].status;
    expect(status).to.equal("Delivery Success");
  });

  
  it("Test Stringify in Typescript SDK", async () => {
    const info = (await relayer.getWormholeRelayerInfo(2, "0xf3b6d47694db4a4e8a28eae14be205c430a00d9b62ab60612e24728d1eeb4a88", {environment: "DEVNET"})) as DeliveryInfo;
    console.log(relayer.stringifyWormholeRelayerInfo(info));
  })

  it("Test getPrice in Typescript SDK", async () => {

    const price = (await relayer.getPrice(sourceChain.chainId, targetChain.chainId, 200000, {environment: environment}));
    console.log(price.toString());
    expect(price).to.not.equal(undefined);
  });

  it("Test getPriceMultipleHops in Typescript SDK", async () => {

    const price = (await relayer.getPriceMultipleHops(sourceChain.chainId, [{targetChain: targetChain.chainId, gasAmount: 200000}, {targetChain: sourceChain.chainId, gasAmount: 200000}], environment));
    console.log(price.toString());
    expect(price).to.not.equal(undefined);
  });

  it("Test the send in Typescript SDK + Cross chain refund", async () => {
    const arbitraryPayload = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload}`);
    const value = await relayer.getPrice(
      sourceChain.chainId,
      targetChain.chainId,
      500000,
      { environment: environment }
    );
    console.log(`Quoted gas delivery fee: ${value}`);
    const startingBalance = await walletSource.getBalance();
    // Dummy target address; doesn't exist
    const tx = await relayer.send(
      sourceChain.chainId,
      targetChain.chainId,
      "0xa114d06ee4a140da5b6c0175f7886b7753d0343a",
      walletSource,
      Buffer.from("hi!"),
      value,
      { environment: environment }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");
    const endingBalance = await walletSource.getBalance();

    await waitForRelay();

    console.log("Checking status using SDK");
    const info = (await relayer.getWormholeRelayerInfo(
      sourceChain.chainId,
      tx.hash,
      { environment: environment }
    )) as DeliveryInfo;
    console.log(relayer.stringifyWormholeRelayerInfo(info));
    const status = info.targetChainStatus.events[0].status;
    expect(status).to.equal("Delivery Success");

    console.log(`Quoted gas delivery fee: ${value}`);
    console.log(
      `Cost (including gas) ${startingBalance.sub(endingBalance).toString()}`
    );

    await waitForRelay();
    const newEndingBalance = await walletSource.getBalance();
    console.log(`Refund: ${newEndingBalance.sub(endingBalance).toString()}`);
    console.log(
      `As a percentage of original value: ${newEndingBalance
        .sub(endingBalance)
        .mul(100)
        .div(value)
        .toString()}%`
    );
  });

  it("Tests the Typescript SDK with a Delivery Failure", async () => {
    const arbitraryPayload = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload}`);
    const valueNotEnough = await sourceCoreRelayer.quoteGas(
      targetChain.chainId,
      10000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    const value = await sourceCoreRelayer.quoteGas(
      targetChain.chainId,
      500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    console.log(`Quoted gas delivery fee: ${value}`);
    const tx = await sourceMockIntegration.sendMessage(
      arbitraryPayload,
      targetChain.chainId,
      targetMockIntegrationAddress,
      { value: valueNotEnough, gasLimit: 500000 }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    await waitForRelay();

    const message = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload}`);
    console.log(`Received message: ${message}`);
    expect(message).to.not.equal(arbitraryPayload);

    console.log("Checking status using SDK");
    const info = (await relayer.getWormholeRelayerInfo(
      sourceChain.chainId as ChainId,
      tx.hash,
      { environment: environment }
    )) as relayer.DeliveryInfo;
    const status = info.targetChainStatus.events[0].status;
    expect(status).to.equal("Receiver Failure");
    console.log(relayer.stringifyWormholeRelayerInfo(info));
  });

  it("Tests a receiver failure and then redelivery", async () => {
    const arbitraryPayload = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload}`);
    const valueNotEnough = await sourceCoreRelayer.quoteGas(
      targetChain.chainId,
      10000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    const value = await sourceCoreRelayer.quoteGas(
      targetChain.chainId,
      500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    console.log(`Quoted gas delivery fee: ${value}`);
    const tx = await sourceMockIntegration.sendMessage(
      arbitraryPayload,
      targetChain.chainId,
      targetMockIntegrationAddress,
      { value: valueNotEnough, gasLimit: 500000 }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    await waitForRelay();

    const message = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload}`);
    console.log(`Received message: ${message}`);
    expect(message).to.not.equal(arbitraryPayload);

    console.log("Checking status using SDK");
    const info = (await relayer.getWormholeRelayerInfo(
      sourceChain.chainId,
      tx.hash,
      { environment: environment }
    )) as DeliveryInfo;
    const status = info.targetChainStatus.events[0].status;
    expect(status).to.equal("Receiver Failure");

    console.log("Redelivering message");
    const redeliveryReceipt = await relayer.resend(
      walletSource,
      sourceChain.chainId as ChainId,
      targetChain.chainId as ChainId,
      "DEVNET",
      relayer.createVaaKey(
        sourceChain.chainId,
        Buffer.from(
          tryNativeToUint8Array(sourceCoreRelayerAddress, "ethereum")
        ),
        info.sourceDeliverySequenceNumber
      ),
      value,
      0,
      await sourceCoreRelayer.getDefaultRelayProvider(),
      [loadGuardianRpc()],
      true,
      {
        value: value,
        gasLimit: 500000,
      }
    );

    console.log("redelivery tx:", redeliveryReceipt.hash);

    await waitForRelay();

    console.log("Checking if message was relayed after redelivery");
    const message2 = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload}`);
    console.log(`Received message: ${message2}`);
    expect(message2).to.equal(arbitraryPayload);

    //TODO check for redelivery event
  });
});
