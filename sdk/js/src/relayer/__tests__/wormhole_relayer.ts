import { afterAll, beforeEach, describe, expect, jest, test} from "@jest/globals";
import { PublicKey } from "@solana/web3.js";
import { ethers } from "ethers";
import { generateRandomString, waitForRelay } from "./utils/utils";
import {getAddressInfo} from "../consts" 
import {getDefaultProvider} from "../main/helpers"
import {
    relayer,
    ethers_contracts,
    tryNativeToUint8Array,
    ChainId,
    CONTRACTS,
    CHAIN_ID_TO_NAME
  } from "../../../";
  import {GovernanceEmitter, MockGuardians} from "../../../src/mock";

  const env = process.env['ENV'];
  if(!env) throw Error("No env specified: tilt or ci or testnet or mainnet");
  const network = env == 'tilt' || env == 'ci' ? "DEVNET" : env == 'testnet' ? "TESTNET" : env == 'mainnet' ? "MAINNET" : undefined;
  if(!network) throw Error(`Invalid env specified: ${env}`);

  const sourceChainId = network == 'DEVNET' ? 2 : 6;
  const targetChainId = network == 'DEVNET' ? 4 : 14;

// Devnet Private Key
const privateKey = process.env['WALLET_KEY'] || "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"
 
const guardianRPC = env == 'tilt' ? "http://localhost:7071" : env == 'ci' ? "http://guardian:7071" : env == "testnet" ? "https://wormhole-v2-testnet-api.certus.one" : env == "mainnet" ? "https://wormhole-v2-mainnet-api.certus.one" : "";

const sourceAddressInfo = getAddressInfo(sourceChainId, network);
const targetAddressInfo = getAddressInfo(targetChainId, network);
const sourceProvider = getDefaultProvider(network, sourceChainId, env=='ci');
const targetProvider = getDefaultProvider(network, targetChainId,  env=='ci');

// signers
const walletSource = new ethers.Wallet(privateKey, sourceProvider);
const walletTarget = new ethers.Wallet(privateKey, targetProvider);

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

const myMap = new Map<ChainId, ethers.providers.Provider>();
myMap.set(sourceChainId, sourceProvider);
myMap.set(targetChainId, targetProvider);
const infoRequestOptionalParams = {sourceChainProvider: sourceProvider, targetChainProviders: myMap};

const GUARDIAN_KEYS = [
  "cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0",
  "c3b2e45c422a1602333a64078aeb42637370b0f48fe385f9cfa6ad54a8e0c47e",
  "9f790d3f08bc4b5cd910d4278f3deb406e57bb5e924906ccd52052bb078ccd47",
  "b20cc49d6f2c82a5e6519015fc18aa3e562867f85f872c58f1277cfbd2a0c8e4",
  "eded5a2fdcb5bbbfa5b07f2a91393813420e7ac30a72fc935b6df36f8294b855",
  "00d39587c3556f289677a837c7f3c0817cb7541ce6e38a243a4bdc761d534c5e",
  "da534d61a8da77b232f3a2cee55c0125e2b3e33a5cd8247f3fe9e72379445c3b",
  "cdbabfc2118eb00bc62c88845f3bbd03cb67a9e18a055101588ca9b36387006c",
  "c83d36423820e7350428dc4abe645cb2904459b7d7128adefe16472fdac397ba",
  "1cbf4e1388b81c9020500fefc83a7a81f707091bb899074db1bfce4537428112",
  "17646a6ba14a541957fc7112cc973c0b3f04fce59484a92c09bb45a0b57eb740",
  "eb94ff04accbfc8195d44b45e7c7da4c6993b2fbbfc4ef166a7675a905df9891",
  "053a6527124b309d914a47f5257a995e9b0ad17f14659f90ed42af5e6e262b6a",
  "3fbf1e46f6da69e62aed5670f279e818889aa7d8f1beb7fd730770fd4f8ea3d7",
  "53b05697596ba04067e40be8100c9194cbae59c90e7870997de57337497172e9",
  "4e95cb2ff3f7d5e963631ad85c28b1b79cb370f21c67cbdd4c2ffb0bf664aa06",
  "01b8c448ce2c1d43cfc5938d3a57086f88e3dc43bb8b08028ecb7a7924f4676f",
  "1db31a6ba3bcd54d2e8a64f8a2415064265d291593450c6eb7e9a6a986bd9400",
  "70d8f1c9534a0ab61a020366b831a494057a289441c07be67e4288c44bc6cd5d",
];
const GUARDIAN_SET_INDEX = 0;


// for signing wormhole messages
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

const GOVERNANCE_EMITTER_ADDRESS = new PublicKey(
  "11111111111111111111111111111115"
);


// for generating governance wormhole messages
const governance = new GovernanceEmitter(
GOVERNANCE_EMITTER_ADDRESS.toBuffer().toString("hex")
);

const guardianIndices = env=='ci'?[0,1]:[0];

const getStatus = async (txHash: string): Promise<string> => {
    console.log(env);
    console.log(sourceChainId);
    console.log(txHash);
  const info = (await relayer.getWormholeRelayerInfo(
      sourceChainId,
      txHash,
      { environment: network, ...infoRequestOptionalParams }
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
    const price = (await relayer.getPrice(sourceChainId, targetChainId, 200000, {environment: network, sourceChainProvider: sourceProvider}));
    expect(price.toString()).toBe("165000000000000000");
  });

  test("Test getPriceMultipleHops in Typescript SDK", async () => {

    const price = (await relayer.getPriceMultipleHops(sourceChainId, [{targetChainId: targetChainId, gasAmount: 200000, optionalParams: {sourceChainProvider: sourceProvider}}, {targetChainId: sourceChainId, gasAmount: 200000, optionalParams: {sourceChainProvider: sourceProvider}}], network));
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
      { environment: network, sourceChainProvider: sourceProvider }
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
      { environment: network, ...infoRequestOptionalParams }
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
      { environment: network, ...infoRequestOptionalParams }
    )) as relayer.DeliveryInfo;
    console.log(relayer.stringifyWormholeRelayerInfo(info));
    const status = info.targetChainStatus.events[0].status;
    expect(status).toBe("Receiver Failure");

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

  test("Executes a receiver failure and then redelivery through SDK", async () => {
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
      { environment: network, ...infoRequestOptionalParams }
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
      network,
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

  // GOVERNANCE TESTS

  test("Governance: Test Registering Chain", async () => {

    const currentAddress = await sourceCoreRelayer.getRegisteredCoreRelayerContract(6);
    console.log(`For Chain ${sourceChainId}, registered chain 6 address: ${currentAddress}`);

    const expectedNewRegisteredAddress = "0x0000000000000000000000001234567890123456789012345678901234567892";

    const timestamp = (await walletSource.provider.getBlock("latest")).timestamp;
    const chain = 6;
    const firstMessage = governance.publishWormholeRelayerRegisterChain(timestamp, chain, expectedNewRegisteredAddress)
    const firstSignedVaa = guardians.addSignatures(firstMessage, guardianIndices);

    let tx = await sourceCoreRelayer.registerCoreRelayerContract(firstSignedVaa, {gasLimit: 500000});
    await tx.wait();

    const newRegisteredAddress = (await sourceCoreRelayer.getRegisteredCoreRelayerContract(6));

    expect(newRegisteredAddress).toBe(expectedNewRegisteredAddress);

    const inverseFirstMessage = governance.publishWormholeRelayerRegisterChain(timestamp, chain, currentAddress)
    const inverseFirstSignedVaa = guardians.addSignatures(inverseFirstMessage, guardianIndices);

    tx = await sourceCoreRelayer.registerCoreRelayerContract(inverseFirstSignedVaa, {gasLimit: 500000});
    await tx.wait();

    const secondRegisteredAddress = (await sourceCoreRelayer.getRegisteredCoreRelayerContract(6));

    expect(secondRegisteredAddress).toBe(currentAddress);
})

test("Governance: Test Setting Default Relay Provider", async () => {

    const currentAddress = await sourceCoreRelayer.getDefaultRelayProvider();
    console.log(`For Chain ${sourceChainId}, default relay provider: ${currentAddress}`);

    const expectedNewDefaultRelayProvider = "0x1234567890123456789012345678901234567892";

    const timestamp = (await walletSource.provider.getBlock("latest")).timestamp;
    const chain = sourceChainId;
    const firstMessage = governance.publishWormholeRelayerSetDefaultRelayProvider(timestamp, chain, expectedNewDefaultRelayProvider);
    const firstSignedVaa = guardians.addSignatures(firstMessage, guardianIndices);

    let tx = await sourceCoreRelayer.setDefaultRelayProvider(firstSignedVaa);
    await tx.wait();

    const newDefaultRelayProvider = (await sourceCoreRelayer.getDefaultRelayProvider());

    expect(newDefaultRelayProvider).toBe(expectedNewDefaultRelayProvider);

    const inverseFirstMessage = governance.publishWormholeRelayerSetDefaultRelayProvider(timestamp, chain, currentAddress)
    const inverseFirstSignedVaa = guardians.addSignatures(inverseFirstMessage, guardianIndices);

    tx = await sourceCoreRelayer.setDefaultRelayProvider(inverseFirstSignedVaa);
    await tx.wait();

    const originalDefaultRelayProvider = (await sourceCoreRelayer.getDefaultRelayProvider());

    expect(originalDefaultRelayProvider).toBe(currentAddress);

});


test("Governance: Test Upgrading Contract", async () => {
  const IMPLEMENTATION_STORAGE_SLOT = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc";

  const getImplementationAddress = () => sourceProvider.getStorageAt(sourceCoreRelayer.address, IMPLEMENTATION_STORAGE_SLOT);

  console.log(`Current Implementation address: ${(await getImplementationAddress())}`);

  const wormholeAddress = CONTRACTS[network][CHAIN_ID_TO_NAME[sourceChainId as ChainId]].core || "";

  const newCoreRelayerImplementationAddress = (await new ethers_contracts.CoreRelayer__factory(walletSource).deploy(wormholeAddress, ethers.utils.getAddress(await sourceCoreRelayer.getDefaultRelayProvider())).then((x)=>x.deployed())).address;

  console.log(`Deployed!`);
  console.log(`New core relayer implementation: ${newCoreRelayerImplementationAddress}`);

  const timestamp = (await walletSource.provider.getBlock("latest")).timestamp;
  const chain = sourceChainId;
  const firstMessage = governance.publishWormholeRelayerUpgradeContract(timestamp, chain, newCoreRelayerImplementationAddress);
  const firstSignedVaa = guardians.addSignatures(firstMessage, guardianIndices);

  let tx = await sourceCoreRelayer.submitContractUpgrade(firstSignedVaa);

  expect(ethers.utils.getAddress((await getImplementationAddress()).substring(26))).toBe(ethers.utils.getAddress(newCoreRelayerImplementationAddress));
});
});
