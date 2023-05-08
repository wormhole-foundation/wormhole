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
import { ChainId } from "@certusone/wormhole-sdk";

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
    )) as relayer.DeliveryInfo;
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
    )) as relayer.DeliveryInfo;
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
    const addressMap = new Map<ChainId, string>();
    addressMap.set(6, "0x6bBaF11913b3Ebb383fEee962B07Cd9a048F7029");
    addressMap.set(4, "0x44C34D7e0CEAc3B9255ACafFb3dC061D2d90fe20")
    addressMap.set(5, "0x44C34D7e0CEAc3B9255ACafFb3dC061D2d90fe20")
    const blockRangeMap = new Map<ChainId, [ethers.providers.BlockTag, ethers.providers.BlockTag]>();
    blockRangeMap.set(4, [29527513, 29527515])

    const info = (await relayer.getWormholeRelayerInfo(6, "0xf6c47da953a7d8a6d4438ad89ba5295bb240392f3742c0b91aea0dced75d3a35", {environment: "TESTNET", coreRelayerAddresses: addressMap, targetChainBlockRanges: blockRangeMap })) as relayer.DeliveryInfo;
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
    )) as relayer.DeliveryInfo;
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
    )) as relayer.DeliveryInfo;
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


  // The following tests are meant to test governance actions
  // These may not pass if, e.g. the governance action has already been performed

  /*
  it("Test governance actions", async () => {
    
    //  worm generate registration -c avalanche -a 0x1357924680135792468013579246801357924680 -m "CoreRelayer" -g cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    console.log(`For Chain 2, registered chain 6 address: ${(await sourceCoreRelayer.registeredCoreRelayerContract(6))}`);
    let tx = await sourceCoreRelayer.registerCoreRelayerContract(Buffer.from("010000000001006cf46ab00b7f80e3d629479f94d07f93fd845f0522f8581408bd450246923aef117d4beebd4a6a4e4d5a914b51dfab5d905632e91ffba7400c94a127a56dbbd600000000001db52662000100000000000000000000000000000000000000000000000000000000000000045773bcbab8bbd90420000000000000000000000000000000000000000000436f726552656c6179657201000000031234567890123456789012345678901234567890123456789012345678901234", "hex"), {gasLimit: 500000});
    await tx.wait();
    console.log(`Now for Chain 2, registered chain 6 address: ${(await sourceCoreRelayer.registeredCoreRelayerContract(6))}`);

    // worm generate set-default-relay-provider -c ethereum -f 0x9876543210987654321098765432109876543210 -g cfb12303a19cde580bb4dd771639b0d26bc68353645571a8cff516ab2ee113a0
    console.log(`For Chain 2, default relay provider address: ${(await sourceCoreRelayer.getDefaultRelayProvider())}`);
    tx = await sourceCoreRelayer.setDefaultRelayProvider(Buffer.from("01000000000100296b7c9504a82a59e4e6f4a1b00c26bb1667d5f0431fc39983bf3700062727531ad1f61d9d70da935f0298f383eab2d456d02facca2a498f5686e88c9790390f0000000000597e029300010000000000000000000000000000000000000000000000000000000000000004c74027772636e5bb20000000000000000000000000000000000000000000436f726552656c617965720300023141592631415926314159263141592631415926314159263141592631415926", "hex"), {gasLimit: 500000});
    await tx.wait();
    console.log(`Now for Chain 2, default relay provider address: ${(await sourceCoreRelayer.getDefaultRelayProvider())}`);
  });*/
  /*
  it("Test governance upgrade", async () => {

    // Note: This test as it is should fail because the destination address doesn't have 'initialize' implemented
    // However, it does revert the error corresponding to the above, indicating that it got to that point in the logic

    
    console.log(`For chain 2, Let's upgrade`);
    const tx = await sourceCoreRelayer.submitContractUpgrade(Buffer.from("010000000001009a7973a4b74a9638e1d7caaf76e7adcbed7c7e33de532cf81b65376afb798215744ab1b5a67a48d779bef004140e4481a5bc9b636ce6ae26ebec4948101ff65e00000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000217d6a200000000000000000000000000000000000000000000436f726552656c617965720200020000000000000000000000001ef9e15c3bbf0555860b5009b51722027134d53a", "hex"), {gasLimit: 500000});
    const rx = await tx.wait();
    console.log("Logs:");
    console.log(rx.logs);
    

  });*/
});
