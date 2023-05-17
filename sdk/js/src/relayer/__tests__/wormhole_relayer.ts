import { afterAll, beforeEach, describe, expect, jest, test} from "@jest/globals";
import { ethers } from "ethers";
import { generateRandomString, waitForRelay } from "./utils/utils";
import {getAddressInfo, getRPC} from "../consts" 
import {
    relayer,
    ethers_contracts,
    tryNativeToUint8Array,
    ChainId
  } from "../../../";

  const env = process.env['ENV'];
  if(!env) throw Error("No env specified: tilt or ci or testnet or mainnet");
  const network = env == 'tilt' || env == 'ci' ? "DEVNET" : env == 'testnet' ? "TESTNET" : env == 'mainnet' ? "MAINNET" : undefined;
  if(!network) throw Error(`Invalid env specified: ${env}`);

  const sourceChainId = network == 'DEVNET' ? 2 : 6;
  const targetChainId = network == 'DEVNET' ? 4 : 14;

// Devnet Private Key
const privateKey = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"

const guardianRPC = "http://localhost:7071"

const sourceAddressInfo = getAddressInfo(sourceChainId, network);
const targetAddressInfo = getAddressInfo(targetChainId, network);
const sourceRpc = getRPC(sourceChainId, network, env=='ci');
const targetRpc = getRPC(targetChainId, network, env=='ci');

// signers
const walletSource = new ethers.Wallet(privateKey, new ethers.providers.JsonRpcProvider(sourceRpc));
const walletTarget = new ethers.Wallet(privateKey, new ethers.providers.JsonRpcProvider(targetRpc));

const sourceCoreRelayerAddress = sourceAddressInfo.coreRelayerAddress;
const sourceMockIntegrationAddress = sourceAddressInfo.mockIntegrationAddress;
const targetCoreRelayerAddress = targetAddressInfo.coreRelayerAddress;
const targetMockIntegrationAddress = targetAddressInfo.mockIntegrationAddress;

if(!sourceCoreRelayerAddress) throw Error("No source core relayer address");
if(!targetCoreRelayerAddress) throw Error("No source core relayer address");
if(!sourceMockIntegrationAddress) throw Error("No source mock integration address");
if(!targetMockIntegrationAddress) throw Error("No source mock integration address");

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

const getStatus = async (txHash: string): Promise<string> => {
    console.log(env);
    console.log(sourceChainId);
    console.log(txHash);
  const info = (await relayer.getWormholeRelayerInfo(
      sourceChainId,
      txHash,
      { environment: network }
    )) as relayer.DeliveryInfo;
  return  info.targetChainStatus.events[0].status;
}

const testSend = async (payload: string, sendToSourceChain?: boolean, notEnoughValue?: boolean): Promise<string> => {
  const value = await sourceCoreRelayer.quoteGas(
      sendToSourceChain ? sourceChainId : targetChainId,
      notEnoughValue ? 10000 : 500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    console.log(`Quoted gas delivery fee: ${value}`);
    const tx = await sourceMockIntegration.sendMessage(
      payload,
      sendToSourceChain ? sourceChainId : targetChainId,
     sendToSourceChain ? sourceMockIntegrationAddress : targetMockIntegrationAddress,
      { value, gasLimit: 500000 }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    return tx.hash;
}

const testForward = async (payload1: string, payload2: string, notEnoughExtraForwardingValue?: boolean): Promise<string> => {
  const value = await sourceCoreRelayer.quoteGas(
      targetChainId,
      500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    const extraForwardingValue = await targetCoreRelayer.quoteGas(
      sourceChainId,
      notEnoughExtraForwardingValue ? 10000 : 800000,
      await targetCoreRelayer.getDefaultRelayProvider()
    );
    console.log(`Quoted gas delivery fee: ${value.add(extraForwardingValue)}`);

    const furtherInstructions: ethers_contracts.MockRelayerIntegration.FurtherInstructionsStruct = {
      keepSending: true,
      newMessages: [payload2, "0x00"],
      chains: [sourceChainId],
      gasLimits: [500000],
    };
    const tx = await sourceMockIntegration.sendMessagesWithFurtherInstructions(
      [payload1],
      furtherInstructions,
      [targetChainId],
      [value.add(extraForwardingValue)],
      { value: value.add(extraForwardingValue), gasLimit: 500000 }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    return tx.hash
}

describe("Wormhole Relayer Tests", () => {

  test("Executes a Delivery Success", async () => {
    const arbitraryPayload = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload}`);
    
    await testSend(arbitraryPayload);

    await waitForRelay();

    console.log("Checking if message was relayed");
    const message = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload}`);
    console.log(`Received message: ${message}`);
    expect(message).toBe(arbitraryPayload);
  });

  test("Reads Delivery Success status through SDK", async () => {
    const arbitraryPayload = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload}`);
    
    const txHash = await testSend(arbitraryPayload);

    await waitForRelay();

    const message = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload}`);
    console.log(`Received message: ${message}`);
    expect(message).toBe(arbitraryPayload);

    console.log("Checking status using SDK");

    const status = await getStatus(txHash);
    
    expect(status).toBe("Delivery Success");
  });


  test("Executes a forward", async () => {
    const arbitraryPayload1 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    const arbitraryPayload2 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload1}`);
    
    const txHash = await testForward(arbitraryPayload1, arbitraryPayload2);

    await waitForRelay(2);

    console.log("Checking if message was relayed");
    const message1 = await targetMockIntegration.getMessage();
    console.log(
      `Sent message: ${arbitraryPayload1} (expecting ${arbitraryPayload2} from forward)`
    );
    console.log(`Received message on target: ${message1}`);
    expect(message1).toBe(arbitraryPayload1);

    console.log("Checking if forward message was relayed back");
    const message2 = await sourceMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload2}`);
    console.log(`Received message on source: ${message2}`);
    expect(message2).toBe(arbitraryPayload2);

  });

  test("Reads Forward Request Success status through SDK", async () => {
    const arbitraryPayload1 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    const arbitraryPayload2 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload1}`);
    
    const txHash = await testForward(arbitraryPayload1, arbitraryPayload2);

    await waitForRelay(2);

    console.log("Checking if message was relayed");
    const message1 = await targetMockIntegration.getMessage();
    console.log(
      `Sent message: ${arbitraryPayload1} (expecting ${arbitraryPayload2} from forward)`
    );
    console.log(`Received message on target: ${message1}`);
    expect(message1).toBe(arbitraryPayload1);

    console.log("Checking if forward message was relayed back");
    const message2 = await sourceMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload2}`);
    console.log(`Received message on source: ${message2}`);
    expect(message2).toBe(arbitraryPayload2);

    const status = await getStatus(txHash);
    expect(status).toBe("Forward Request Success");
  });

  test("Executes a multidelivery", async () => {
    const arbitraryPayload1 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    const arbitraryPayload2 = ethers.utils.hexlify(
        ethers.utils.toUtf8Bytes(generateRandomString(32))
      );
    console.log(`Sent message: ${arbitraryPayload1}`);
   
    const txHash1 = await testSend(arbitraryPayload1, true);
    const txHash2 = await testSend(arbitraryPayload2);

    await waitForRelay();

    console.log("Checking if first message was relayed");
    const message1 = await sourceMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload1}`);
    console.log(`Received message: ${message1}`);
    expect(message1).toBe(arbitraryPayload1);

    console.log("Checking if second message was relayed");
    const message2 = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload2}`);
    console.log(`Received message: ${message2}`);
    expect(message2).toBe(arbitraryPayload2);
  });

  test("Executes a multiforward", async () => {
    const arbitraryPayload1 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    const arbitraryPayload2 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload1}`);
    const value1 = await sourceCoreRelayer.quoteGas(
      targetChainId,
      500000,
      await sourceCoreRelayer.getDefaultRelayProvider()
    );
    const value2 = await targetCoreRelayer.quoteGas(
      sourceChainId,
      500000,
      await targetCoreRelayer.getDefaultRelayProvider()
    );
    const value3 = await targetCoreRelayer.quoteGas(
      targetChainId,
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

    const furtherInstructions: ethers_contracts.MockRelayerIntegration.FurtherInstructionsStruct = {
      keepSending: true,
      newMessages: [arbitraryPayload2, "0x00"],
      chains: [sourceChainId, targetChainId],
      gasLimits: [500000, 500000],
    };
    const tx = await sourceMockIntegration.sendMessagesWithFurtherInstructions(
      [arbitraryPayload1],
      furtherInstructions,
      [targetChainId],
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
    expect(message1).toBe(arbitraryPayload2);

    console.log("Checking if second forward was relayed");
    const message2 = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload2}`);
    console.log(`Received message: ${message2}`);
    expect(message2).toBe(arbitraryPayload2);
  });

  test("Executes a Forward Request Failure", async () => {
    const arbitraryPayload1 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    const arbitraryPayload2 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload1}`);
   
    const txHash = await testForward(arbitraryPayload1, arbitraryPayload2, true);

    await waitForRelay();

    console.log("Checking if message was relayed (it shouldn't have been!");
    const message1 = await targetMockIntegration.getMessage();
    console.log(
      `Sent message: ${arbitraryPayload1} (expecting ${arbitraryPayload2} from forward)`
    );
    console.log(`Received message on target: ${message1}`);
    expect(message1).not.toBe(arbitraryPayload1);

    console.log(
      "Checking if forward message was relayed back (it shouldn't have been!)"
    );
    const message2 = await sourceMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload2}`);
    console.log(`Received message on source: ${message2}`);
    expect(message2).not.toBe(arbitraryPayload2);

  });

  test("Reads Forward Request Failure status through SDK", async () => {
    const arbitraryPayload1 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    const arbitraryPayload2 = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload1}`);
   
    const txHash = await testForward(arbitraryPayload1, arbitraryPayload2, true);

    await waitForRelay();

    console.log("Checking if message was relayed (it shouldn't have been!");
    const message1 = await targetMockIntegration.getMessage();
    console.log(
      `Sent message: ${arbitraryPayload1} (expecting ${arbitraryPayload2} from forward)`
    );
    console.log(`Received message on target: ${message1}`);
    expect(message1).not.toBe(arbitraryPayload1);

    console.log(
      "Checking if forward message was relayed back (it shouldn't have been!)"
    );
    const message2 = await sourceMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload2}`);
    console.log(`Received message on source: ${message2}`);
    expect(message2).not.toBe(arbitraryPayload2);

    const status = await getStatus(txHash);
    expect(status).toBe("Forward Request Failure");
  });
  

  test("Test getPrice in Typescript SDK", async () => {
    const price = (await relayer.getPrice(sourceChainId, targetChainId, 200000, {environment: network}));
    expect(price.toString()).toBe("165000000000000000");
  });

  test("Test getPriceMultipleHops in Typescript SDK", async () => {

    const price = (await relayer.getPriceMultipleHops(sourceChainId, [{targetChainId: targetChainId, gasAmount: 200000}, {targetChainId: sourceChainId, gasAmount: 200000}], network));
    expect(price.toString()).toBe("338250000000000000");
  });

  test("Executes a delivery with a Cross Chain Refund + Reads Status from SDK", async () => {
    const arbitraryPayload = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload}`);
    const value = await relayer.getPrice(
      sourceChainId,
      targetChainId,
      500000,
      { environment: network }
    );
    console.log(`Quoted gas delivery fee: ${value}`);
    const startingBalance = await walletSource.getBalance();
    // Dummy target address; doesn't exist
    const tx = await relayer.send(
      sourceChainId,
      targetChainId,
      targetCoreRelayerAddress, // This is an address that exists but doesn't implement the IWormhole interface
      walletSource,
      Buffer.from("hi!"),
      value,
      { environment: network }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");
    const endingBalance = await walletSource.getBalance();

    await waitForRelay();

    console.log("Checking status using SDK");
    const info = (await relayer.getWormholeRelayerInfo(
      sourceChainId,
      tx.hash,
      { environment: network }
    )) as relayer.DeliveryInfo;
    console.log(relayer.stringifyWormholeRelayerInfo(info));
    const status = info.targetChainStatus.events[0].status;
    expect(status).toBe("Delivery Success");

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

  test("Executes a Receiver Failure", async () => {
    const arbitraryPayload = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload}`);
    
    const txHash = await testSend(arbitraryPayload, false, true);

    await waitForRelay();

    const message = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload}`);
    console.log(`Received message: ${message}`);
    expect(message).not.toBe(arbitraryPayload);
  });

  test("Reads Receiver Failure status through SDK", async () => {
    const arbitraryPayload = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload}`);
    
    const txHash = await testSend(arbitraryPayload, false, true);

    await waitForRelay();

    const message = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload}`);
    console.log(`Received message: ${message}`);
    expect(message).not.toBe(arbitraryPayload);

    const status = await getStatus(txHash);
    expect(status).toBe("Receiver Failure");
  });

  test("Executes a receiver failure and then redelivery", async () => {
    const arbitraryPayload = ethers.utils.hexlify(
      ethers.utils.toUtf8Bytes(generateRandomString(32))
    );
    console.log(`Sent message: ${arbitraryPayload}`);
    
    const txHash = await testSend(arbitraryPayload, false, true);

    await waitForRelay();

    const message = await targetMockIntegration.getMessage();
    console.log(`Sent message: ${arbitraryPayload}`);
    console.log(`Received message: ${message}`);
    expect(message).not.toBe(arbitraryPayload);

    console.log("Checking status using SDK");
    const info = (await relayer.getWormholeRelayerInfo(
      sourceChainId,
      txHash,
      { environment: network }
    )) as relayer.DeliveryInfo;
    const status = info.targetChainStatus.events[0].status;
    expect(status).toBe("Receiver Failure");

    const value = await sourceCoreRelayer.quoteGas(
        targetChainId,
        500000,
        await sourceCoreRelayer.getDefaultRelayProvider()
      );

    console.log("Redelivering message");
    const redeliveryReceipt = await relayer.resend(
      walletSource,
      sourceChainId as ChainId,
      targetChainId as ChainId,
      "DEVNET",
      relayer.createVaaKey(
        sourceChainId,
        Buffer.from(
          tryNativeToUint8Array(sourceCoreRelayerAddress, "ethereum")
        ),
        info.sourceDeliverySequenceNumber
      ),
      value,
      0,
      await sourceCoreRelayer.getDefaultRelayProvider(),
      [guardianRPC],
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
    expect(message2).toBe(arbitraryPayload);

    //Can extend this to look for redelivery event
  });
});
