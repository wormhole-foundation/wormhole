import { afterAll, beforeEach, describe, expect, jest, test} from "@jest/globals";
import { ethers } from "ethers";
import { getNetwork, isCI, generateRandomString, waitForRelay, PRIVATE_KEY, getGuardianRPC, GUARDIAN_KEYS, GUARDIAN_SET_INDEX, GOVERNANCE_EMITTER_ADDRESS, getArbitraryBytes32} from "./utils/utils";
import {getAddressInfo} from "../consts" 
import {getDefaultProvider} from "../main/helpers"
import {
    relayer,
    ethers_contracts,
    tryNativeToUint8Array,
    ChainId,
    CHAINS,
    CONTRACTS,
    CHAIN_ID_TO_NAME,
    ChainName,
    Network
  } from "../../../";
  import {GovernanceEmitter, MockGuardians} from "../../../src/mock";

  const network: Network = getNetwork();
  const ci: boolean = isCI();
  
  const sourceChain = network == 'DEVNET' ? "ethereum" : "avalanche";
  const targetChain = network == 'DEVNET' ? "bsc" : "celo";
  
  const sourceChainId = CHAINS[sourceChain];
  const targetChainId = CHAINS[targetChain];

const sourceAddressInfo = getAddressInfo(sourceChain, network);
const targetAddressInfo = getAddressInfo(targetChain, network);
const sourceProvider = getDefaultProvider(network, sourceChain, ci);
const targetProvider = getDefaultProvider(network, targetChain, ci);

// signers
const walletSource = new ethers.Wallet(PRIVATE_KEY, sourceProvider);
const walletTarget = new ethers.Wallet(PRIVATE_KEY, targetProvider);

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

const myMap = new Map<ChainName, ethers.providers.Provider>();
myMap.set(sourceChain, sourceProvider);
myMap.set(targetChain, targetProvider);
const optionalParams = {environment: network, sourceChainProvider: sourceProvider, targetChainProviders: myMap};

// for signing wormhole messages
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);


// for generating governance wormhole messages
const governance = new GovernanceEmitter(
GOVERNANCE_EMITTER_ADDRESS
);

const guardianIndices = ci?[0,1]:[0];

const REASONABLE_GAS_LIMIT = 500000;
const TOO_LOW_GAS_LIMIT = 10000;
const REASONABLE_GAS_LIMIT_FORWARDS = 800000;

const getStatus = async (txHash: string): Promise<string> => {
  const info = (await relayer.getWormholeRelayerInfo(
      sourceChain,
      txHash,
      optionalParams
    )) as relayer.DeliveryInfo;
  return  info.targetChainStatus.events[0].status;
}

const testSend = async (payload: string, sendToSourceChain?: boolean, notEnoughValue?: boolean): Promise<string> => {
    const value = await relayer.getPrice(sourceChain, sendToSourceChain ? sourceChain : targetChain, notEnoughValue ? TOO_LOW_GAS_LIMIT : REASONABLE_GAS_LIMIT);
    console.log(`Quoted gas delivery fee: ${value}`);
    const tx = await sourceMockIntegration.sendMessage(
      payload,
      sendToSourceChain ? sourceChainId : targetChainId,
     sendToSourceChain ? sourceMockIntegrationAddress : targetMockIntegrationAddress,
      { value, gasLimit: REASONABLE_GAS_LIMIT }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    return tx.hash;
}

const testForward = async (payload1: string, payload2: string, notEnoughExtraForwardingValue?: boolean): Promise<string> => {
    const value = await relayer.getPriceMultipleHops(sourceChain, [{targetChain: targetChain, gasAmount: REASONABLE_GAS_LIMIT_FORWARDS, optionalParams: optionalParams}, {targetChain: sourceChain, gasAmount: REASONABLE_GAS_LIMIT, optionalParams: optionalParams}, network]);
    console.log(`Quoted gas delivery fee: ${value}`);

    const furtherInstructions: ethers_contracts.MockRelayerIntegration.FurtherInstructionsStruct = {
      keepSending: true,
      newMessages: [payload2, "0x00"],
      chains: [sourceChainId],
      gasLimits: [REASONABLE_GAS_LIMIT],
    };
    const tx = await sourceMockIntegration.sendMessagesWithFurtherInstructions(
      [payload1],
      furtherInstructions,
      [targetChainId],
      [relayer.getPrice(targetChain, sourceChain, REASONABLE_GAS_LIMIT, optionalParams)],
      { value: value, gasLimit: REASONABLE_GAS_LIMIT }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    return tx.hash
}

describe("Wormhole Relayer Tests", () => {

  test("Executes a Delivery Success", async () => {
    const arbitraryPayload = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload}`);
    
    const txHash = await testSend(arbitraryPayload);

    await waitForRelay();

    console.log("Checking if message was relayed");
    const message = await targetMockIntegration.getMessage();
    expect(message).toBe(arbitraryPayload);

    console.log("Checking status using SDK");
    const status = await getStatus(txHash);
    expect(status).toBe("Delivery Success");

  });

  test("Executes a Forward Request Success", async () => {
    const arbitraryPayload1 = getArbitraryBytes32()
    const arbitraryPayload2 = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload1}, expecting ${arbitraryPayload2} to be forwarded`);
    
    const txHash = await testForward(arbitraryPayload1, arbitraryPayload2);

    await waitForRelay(2);

    console.log("Checking if message was relayed");
    const message1 = await targetMockIntegration.getMessage();
    expect(message1).toBe(arbitraryPayload1);

    console.log("Checking if forward message was relayed back");
    const message2 = await sourceMockIntegration.getMessage();
    expect(message2).toBe(arbitraryPayload2);

    console.log("Checking status using SDK");
    const status = await getStatus(txHash);
    expect(status).toBe("Forward Request Success");
  
  });

  test("Executes multiple forwards", async () => {
    const arbitraryPayload1 = getArbitraryBytes32()
    const arbitraryPayload2 = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload1}, expecting ${arbitraryPayload2} to be forwarded`);
    const value1 = await relayer.getPrice(sourceChain, targetChain, REASONABLE_GAS_LIMIT, optionalParams);
    const value2 = await relayer.getPrice(targetChain, sourceChain, REASONABLE_GAS_LIMIT, optionalParams)
    const value3 = await relayer.getPrice(targetChain, targetChain, REASONABLE_GAS_LIMIT, optionalParams)
    // Have enough value on the target chain to fund both forwards
    const payment = value1
      .add((value2
      .add(value3)
      .mul(105) // Apply asset conversion buffer in reverse
      .div(100)
      .add(1)));
    console.log(`Quoted gas delivery fee: ${payment}`);

    const furtherInstructions: ethers_contracts.MockRelayerIntegration.FurtherInstructionsStruct = {
      keepSending: true,
      newMessages: [arbitraryPayload2, "0x00"],
      chains: [sourceChainId, targetChainId],
      gasLimits: [REASONABLE_GAS_LIMIT, REASONABLE_GAS_LIMIT],
    };
    const tx = await sourceMockIntegration.sendMessagesWithFurtherInstructions(
      [arbitraryPayload1],
      furtherInstructions,
      [targetChainId],
      [payment],
      { value: payment, gasLimit: REASONABLE_GAS_LIMIT_FORWARDS }
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");

    await waitForRelay(2);

    console.log("Checking if first forward was relayed");
    const message1 = await sourceMockIntegration.getMessage();
    expect(message1).toBe(arbitraryPayload2);

    console.log("Checking if second forward was relayed");
    const message2 = await targetMockIntegration.getMessage();
    expect(message2).toBe(arbitraryPayload2);
  });

  test("Executes a Forward Request Failure", async () => {
    const arbitraryPayload1 = getArbitraryBytes32()
    const arbitraryPayload2 = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload1}, expecting ${arbitraryPayload2} to be forwarded (but should fail)`);
   
    const txHash = await testForward(arbitraryPayload1, arbitraryPayload2, true);

    await waitForRelay();

    console.log("Checking if message was relayed (it shouldn't have been!");
    const message1 = await targetMockIntegration.getMessage();
    expect(message1).not.toBe(arbitraryPayload1);

    console.log(
      "Checking if forward message was relayed back (it shouldn't have been!)"
    );
    const message2 = await sourceMockIntegration.getMessage();
    expect(message2).not.toBe(arbitraryPayload2);

    const status = await getStatus(txHash);
    expect(status).toBe("Forward Request Failure");

  });


  test("Test getPrice in Typescript SDK", async () => {
    const price = (await relayer.getPrice(sourceChain, targetChain, 200000, optionalParams));
    expect(price.toString()).toBe("165000000000000000");
  });

  test("Test getPriceMultipleHops in Typescript SDK", async () => {
    const price = (await relayer.getPriceMultipleHops(sourceChain, [{targetChain: targetChain, gasAmount: 200000, optionalParams}, {targetChain: sourceChain, gasAmount: 200000, optionalParams}], network));
    expect(price.toString()).toBe("338250000000000000");
  });

  test("Executes a delivery with a Cross Chain Refund + Reads Status from SDK", async () => {
    const arbitraryPayload = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload}`);
    const value = await relayer.getPrice(
      sourceChain,
      targetChain,
      REASONABLE_GAS_LIMIT,
      { environment: network, sourceChainProvider: sourceProvider }
    );
    console.log(`Quoted gas delivery fee: ${value}`);
    const startingBalance = await walletSource.getBalance();

    const tx = await relayer.send(
      sourceChain,
      targetChain,
      targetCoreRelayerAddress, // This is an address that exists but doesn't implement the IWormhole interface, so should result in Receiver Failure
      walletSource,
      Buffer.from("hi!"),
      value,
      optionalParams
    );
    console.log("Sent delivery request!");
    const rx = await tx.wait();
    console.log("Message confirmed!");
    const endingBalance = await walletSource.getBalance();

    await waitForRelay();

    console.log("Checking status using SDK");
    const status = await getStatus(tx.hash);
    expect(status).toBe("Receiver Failure");

    const info = (await relayer.getWormholeRelayerInfo(sourceChain, tx.hash, optionalParams)) as relayer.DeliveryInfo;

    await waitForRelay();

    const newEndingBalance = await walletSource.getBalance();

    console.log("Checking status of refund using SDK");
    const statusOfRefund = await getStatus(info.targetChainStatus.events[0].transactionHash || "");
    expect(statusOfRefund).toBe("Receiver Failure"); // This is what the status is set to in the codepath where refunds are sent

    console.log(`Quoted gas delivery fee: ${value}`);
    console.log(
      `Cost (including gas) ${startingBalance.sub(endingBalance).toString()}`
    );
    const refund = newEndingBalance.sub(endingBalance);
    console.log(`Refund: ${refund.toString()}`);
    console.log(
      `As a percentage of original value: ${newEndingBalance
        .sub(endingBalance)
        .mul(100)
        .div(value)
        .toString()}%`
    );
    console.log("Confirming refund is nonzero");
    expect(refund.gt(0)).toBe(true);
  });

  test("Executes a Receiver Failure", async () => {
    const arbitraryPayload = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload}`);
    
    const txHash = await testSend(arbitraryPayload, false, true);

    await waitForRelay();

    const message = await targetMockIntegration.getMessage();
    expect(message).not.toBe(arbitraryPayload);

    const status = await getStatus(txHash);
    expect(status).toBe("Receiver Failure");
  });

  test("Executes a receiver failure and then redelivery through SDK", async () => {
    const arbitraryPayload = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload}`);
    
    const txHash = await testSend(arbitraryPayload, false, true);

    await waitForRelay();

    const message = await targetMockIntegration.getMessage();
    expect(message).not.toBe(arbitraryPayload);

    console.log("Checking status using SDK");
    const status = await getStatus(txHash);
    expect(status).toBe("Receiver Failure");

    const value = await relayer.getPrice(sourceChain, targetChain, REASONABLE_GAS_LIMIT, optionalParams);

    const info = (await relayer.getWormholeRelayerInfo(sourceChain, txHash, optionalParams)) as relayer.DeliveryInfo;

    console.log("Redelivering message");
    const redeliveryReceipt = await relayer.resend(
      walletSource,
      sourceChain,
      targetChain,
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
      [getGuardianRPC(network, ci)],
      true,
      {
        value: value,
        gasLimit: REASONABLE_GAS_LIMIT,
      }
    );

    console.log("redelivery tx:", redeliveryReceipt.hash);

    await waitForRelay();

    console.log("Checking if message was relayed after redelivery");
    const message2 = await targetMockIntegration.getMessage();
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

    let tx = await sourceCoreRelayer.registerCoreRelayerContract(firstSignedVaa, {gasLimit: REASONABLE_GAS_LIMIT});
    await tx.wait();

    const newRegisteredAddress = (await sourceCoreRelayer.getRegisteredCoreRelayerContract(6));

    expect(newRegisteredAddress).toBe(expectedNewRegisteredAddress);

    const inverseFirstMessage = governance.publishWormholeRelayerRegisterChain(timestamp, chain, currentAddress)
    const inverseFirstSignedVaa = guardians.addSignatures(inverseFirstMessage, guardianIndices);

    tx = await sourceCoreRelayer.registerCoreRelayerContract(inverseFirstSignedVaa, {gasLimit: REASONABLE_GAS_LIMIT});
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

  const wormholeAddress = CONTRACTS[network][sourceChain].core || "";

  const newCoreRelayerImplementationAddress = (await new ethers_contracts.CoreRelayer__factory(walletSource).deploy(wormholeAddress).then((x)=>x.deployed())).address;

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
