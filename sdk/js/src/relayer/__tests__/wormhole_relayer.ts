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
    Network,
  } from "../../../";
  import {GovernanceEmitter, MockGuardians} from "../../../src/mock";
import { AddressInfo } from "net";

  const network: Network = getNetwork();
  const ci: boolean = isCI();
  
  const sourceChain = network == 'DEVNET' ? "ethereum" : "avalanche";
  const targetChain = network == 'DEVNET' ? "bsc" : "celo";

  type TestChain = {
    chainId: ChainId,
    name: ChainName,
    provider: ethers.providers.Provider,
    wallet: ethers.Wallet,
    wormholeRelayerAddress: string,
    mockIntegrationAddress: string,
    wormholeRelayer: ethers_contracts.WormholeRelayer,
    mockIntegration: ethers_contracts.MockRelayerIntegration
  }

  const createTestChain = (name: ChainName) => {
    const provider = getDefaultProvider(network, name, ci);
    const addressInfo = getAddressInfo(name, network);
    if(!addressInfo.wormholeRelayerAddress) throw Error(`No core relayer address for ${name}`);
    if(!addressInfo.mockIntegrationAddress) throw Error(`No mock relayer integration address for ${name}`);
    const wallet = new ethers.Wallet(PRIVATE_KEY, provider);
    const wormholeRelayer = ethers_contracts.WormholeRelayer__factory.connect(
      addressInfo.wormholeRelayerAddress,
      wallet
    );
    const mockIntegration = ethers_contracts.MockRelayerIntegration__factory.connect(
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
      mockIntegration
    }
    return result;
  }

  const source = createTestChain(sourceChain);
  const target = createTestChain(targetChain);

const myMap = new Map<ChainName, ethers.providers.Provider>();
myMap.set(sourceChain, source.provider);
myMap.set(targetChain, target.provider);
const optionalParams = {environment: network, sourceChainProvider: source.provider, targetChainProviders: myMap};

// for signing wormhole messages
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);


// for generating governance wormhole messages
const governance = new GovernanceEmitter(
GOVERNANCE_EMITTER_ADDRESS
);

const guardianIndices = ci?[0,1]:[0];

const REASONABLE_GAS_LIMIT = 500000;
const TOO_LOW_GAS_LIMIT = 10000;
const REASONABLE_GAS_LIMIT_FORWARDS = 900000;

const getStatus = async (txHash: string, _sourceChain?: ChainName): Promise<string> => {
  const info = (await relayer.getWormholeRelayerInfo(
      _sourceChain || sourceChain,
      txHash,
      {environment: network, targetChainProviders: myMap, sourceChainProvider: myMap.get(_sourceChain || sourceChain)}
    )) as relayer.DeliveryInfo;
  return  info.targetChainStatus.events[0].status;
}

const testSend = async (payload: string, sendToSourceChain?: boolean, notEnoughValue?: boolean): Promise<string> => {
    const value = await relayer.getPrice(sourceChain, sendToSourceChain ? sourceChain : targetChain, notEnoughValue ? TOO_LOW_GAS_LIMIT : REASONABLE_GAS_LIMIT, optionalParams);
    console.log(`Quoted gas delivery fee: ${value}`);
    const tx = await source.mockIntegration.sendMessage(
      payload,
      sendToSourceChain ? source.chainId : target.chainId,
      notEnoughValue ? TOO_LOW_GAS_LIMIT : REASONABLE_GAS_LIMIT,
      0,
      { value, gasLimit: REASONABLE_GAS_LIMIT }
    );
    console.log("Sent delivery request!");
    await tx.wait();
    console.log("Message confirmed!");
  
    return tx.hash;
}

const testForward = async (payload1: string, payload2: string, notEnoughExtraForwardingValue?: boolean): Promise<string> => {
    const valueNeededOnTargetChain = await relayer.getPrice(targetChain, sourceChain, notEnoughExtraForwardingValue ? TOO_LOW_GAS_LIMIT : REASONABLE_GAS_LIMIT, optionalParams);
    const value = await relayer.getPrice(sourceChain, targetChain, REASONABLE_GAS_LIMIT_FORWARDS, {receiverValue: valueNeededOnTargetChain, ...optionalParams});
    console.log(`Quoted gas delivery fee: ${value}`);

    const tx = await source.mockIntegration["sendMessageWithForwardedResponse(bytes,bytes,uint16,uint32,uint128)"](
      payload1,
      payload2,
      target.chainId,
      REASONABLE_GAS_LIMIT_FORWARDS,
      valueNeededOnTargetChain,
      { value: value, gasLimit: REASONABLE_GAS_LIMIT }
    );
    console.log("Sent delivery request!");
    await tx.wait();
    console.log("Message confirmed!");

    return tx.hash
}

describe("Wormhole Relayer Tests", () => {

  test("Executes a Delivery Success", async () => {
    const arbitraryPayload = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload}`);
    
    const txHash = await testSend(arbitraryPayload);

    await waitForRelay();

    console.log("Checking status using SDK");
    const status = await getStatus(txHash);
    expect(status).toBe("Delivery Success");

    console.log("Checking if message was relayed");
    const message = await target.mockIntegration.getMessage();
    expect(message).toBe(arbitraryPayload);


  });

  test("Executes a Forward Request Success", async () => {
    const arbitraryPayload1 = getArbitraryBytes32()
    const arbitraryPayload2 = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload1}, expecting ${arbitraryPayload2} to be forwarded`);
    
    const txHash = await testForward(arbitraryPayload1, arbitraryPayload2);

    await waitForRelay(2);


    console.log("Checking status using SDK");
    const status = await getStatus(txHash);
    expect(status).toBe("Forward Request Success");

    console.log("Checking if message was relayed");
    const message1 = await target.mockIntegration.getMessage();
    expect(message1).toBe(arbitraryPayload1);

    console.log("Checking if forward message was relayed back");
    const message2 = await source.mockIntegration.getMessage();
    expect(message2).toBe(arbitraryPayload2);

  
  });

  test("Executes multiple forwards", async () => {
    const arbitraryPayload1 = getArbitraryBytes32()
    const arbitraryPayload2 = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload1}, expecting ${arbitraryPayload2} to be forwarded`);
    const valueNeededOnTargetChain1 = await relayer.getPrice(targetChain, sourceChain, REASONABLE_GAS_LIMIT, optionalParams);
    const valueNeededOnTargetChain2 = await relayer.getPrice(targetChain, targetChain, REASONABLE_GAS_LIMIT, optionalParams);

    const value = await relayer.getPrice(sourceChain, targetChain, REASONABLE_GAS_LIMIT_FORWARDS, {receiverValue: valueNeededOnTargetChain1.add(valueNeededOnTargetChain2), ...optionalParams});
    console.log(`Quoted gas delivery fee: ${value}`);


    const tx = await source.mockIntegration.sendMessageWithMultiForwardedResponse(
      arbitraryPayload1,
      arbitraryPayload2,
      target.chainId,
      REASONABLE_GAS_LIMIT_FORWARDS,
      valueNeededOnTargetChain1.add(valueNeededOnTargetChain2),
      { value: value, gasLimit: REASONABLE_GAS_LIMIT }
    );
    console.log("Sent delivery request!");
    await tx.wait();
    console.log("Message confirmed!");

    await waitForRelay(2);

    const status = await getStatus(tx.hash);
    console.log(`Status of forward: ${status}`)

    console.log("Checking if first forward was relayed");
    const message1 = await source.mockIntegration.getMessage();
    expect(message1).toBe(arbitraryPayload2);

    console.log("Checking if second forward was relayed");
    const message2 = await target.mockIntegration.getMessage();
    expect(message2).toBe(arbitraryPayload2);
  });

  test("Executes a Forward Request Failure", async () => {
    const arbitraryPayload1 = getArbitraryBytes32()
    const arbitraryPayload2 = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload1}, expecting ${arbitraryPayload2} to be forwarded (but should fail)`);
   
    const txHash = await testForward(arbitraryPayload1, arbitraryPayload2, true);

    await waitForRelay();

    const status = await getStatus(txHash);
    expect(status).toBe("Forward Request Failure");

    console.log("Checking if message was relayed (it shouldn't have been!");
    const message1 = await target.mockIntegration.getMessage();
    expect(message1).not.toBe(arbitraryPayload1);

    console.log(
      "Checking if forward message was relayed back (it shouldn't have been!)"
    );
    const message2 = await source.mockIntegration.getMessage();
    expect(message2).not.toBe(arbitraryPayload2);


  });


  test("Test getPrice in Typescript SDK", async () => {
    const price = (await relayer.getPrice(sourceChain, targetChain, 200000, optionalParams));
    expect(price.toString()).toBe("165000000000000000");
  });

  test("Executes a delivery with a Cross Chain Refund", async () => {
    const arbitraryPayload = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload}`);
    const value = await relayer.getPrice(
      sourceChain,
      targetChain,
      REASONABLE_GAS_LIMIT,
      optionalParams
    );
    console.log(`Quoted gas delivery fee: ${value}`);
    const startingBalance = await source.wallet.getBalance();

    const tx = await relayer.sendToEvm(
      source.wallet,
      sourceChain,
      targetChain,
      target.wormholeRelayerAddress, // This is an address that exists but doesn't implement the IWormhole interface, so should result in Receiver Failure
      Buffer.from("hi!"),
      REASONABLE_GAS_LIMIT,
      {value, gasLimit: REASONABLE_GAS_LIMIT},
      optionalParams,
    );
    console.log("Sent delivery request!");
    await tx.wait();
    console.log("Message confirmed!");
    const endingBalance = await source.wallet.getBalance();

    await waitForRelay();

    console.log("Checking status using SDK");
    const status = await getStatus(tx.hash);
    expect(status).toBe("Receiver Failure");

    const info = (await relayer.getWormholeRelayerInfo(sourceChain, tx.hash, optionalParams)) as relayer.DeliveryInfo;

    await waitForRelay();

    const newEndingBalance = await source.wallet.getBalance();

    console.log("Checking status of refund using SDK");
    console.log(relayer.stringifyWormholeRelayerInfo(info));
    const statusOfRefund = await getStatus(info.targetChainStatus.events[0].transactionHash || "", targetChain);
    expect(statusOfRefund).toBe("Delivery Success"); 

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

    const message = await target.mockIntegration.getMessage();
    expect(message).not.toBe(arbitraryPayload);

    const status = await getStatus(txHash);
    expect(status).toBe("Receiver Failure");
  });

  test("Executes a receiver failure and then redelivery through SDK", async () => {
    const arbitraryPayload = getArbitraryBytes32()
    console.log(`Sent message: ${arbitraryPayload}`);
    
    const txHash = await testSend(arbitraryPayload, false, true);

    await waitForRelay();

    const message = await target.mockIntegration.getMessage();
    expect(message).not.toBe(arbitraryPayload);

    console.log("Checking status using SDK");
    const status = await getStatus(txHash);
    expect(status).toBe("Receiver Failure");

    const value = await relayer.getPrice(sourceChain, targetChain, REASONABLE_GAS_LIMIT, optionalParams);

    const info = (await relayer.getWormholeRelayerInfo(sourceChain, txHash, optionalParams)) as relayer.DeliveryInfo;

    console.log("Redelivering message");
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
      true
    );

    console.log("redelivery tx:", redeliveryReceipt.hash);

    await waitForRelay();

    console.log("Checking if message was relayed after redelivery");
    const message2 = await target.mockIntegration.getMessage();
    expect(message2).toBe(arbitraryPayload);

    //Can extend this to look for redelivery event
  });

  // GOVERNANCE TESTS

  test("Governance: Test Registering Chain", async () => {

    const currentAddress = await source.wormholeRelayer.getRegisteredWormholeRelayerContract(6);
    console.log(`For Chain ${source.chainId}, registered chain 6 address: ${currentAddress}`);

    const expectedNewRegisteredAddress = "0x0000000000000000000000001234567890123456789012345678901234567892";

    const timestamp = (await source.wallet.provider.getBlock("latest")).timestamp;
    const chain = 6;
    const firstMessage = governance.publishWormholeRelayerRegisterChain(timestamp, chain, expectedNewRegisteredAddress)
    const firstSignedVaa = guardians.addSignatures(firstMessage, guardianIndices);

    let tx = await source.wormholeRelayer.registerWormholeRelayerContract(firstSignedVaa, {gasLimit: REASONABLE_GAS_LIMIT});
    await tx.wait();

    const newRegisteredAddress = (await source.wormholeRelayer.getRegisteredWormholeRelayerContract(6));

    expect(newRegisteredAddress).toBe(expectedNewRegisteredAddress);

    const inverseFirstMessage = governance.publishWormholeRelayerRegisterChain(timestamp, chain, currentAddress)
    const inverseFirstSignedVaa = guardians.addSignatures(inverseFirstMessage, guardianIndices);

    tx = await source.wormholeRelayer.registerWormholeRelayerContract(inverseFirstSignedVaa, {gasLimit: REASONABLE_GAS_LIMIT});
    await tx.wait();

    const secondRegisteredAddress = (await source.wormholeRelayer.getRegisteredWormholeRelayerContract(6));

    expect(secondRegisteredAddress).toBe(currentAddress);
})

test("Governance: Test Setting Default Relay Provider", async () => {

    const currentAddress = await source.wormholeRelayer.getDefaultDeliveryProvider();
    console.log(`For Chain ${source.chainId}, default relay provider: ${currentAddress}`);

    const expectedNewDefaultDeliveryProvider = "0x1234567890123456789012345678901234567892";

    const timestamp = (await source.wallet.provider.getBlock("latest")).timestamp;
    const chain = source.chainId;
    const firstMessage = governance.publishWormholeRelayerSetDefaultRelayProvider(timestamp, chain, expectedNewDefaultDeliveryProvider);
    const firstSignedVaa = guardians.addSignatures(firstMessage, guardianIndices);

    let tx = await source.wormholeRelayer.setDefaultDeliveryProvider(firstSignedVaa);
    await tx.wait();

    const newDefaultDeliveryProvider = (await source.wormholeRelayer.getDefaultDeliveryProvider());

    expect(newDefaultDeliveryProvider).toBe(expectedNewDefaultDeliveryProvider);

    const inverseFirstMessage = governance.publishWormholeRelayerSetDefaultRelayProvider(timestamp, chain, currentAddress)
    const inverseFirstSignedVaa = guardians.addSignatures(inverseFirstMessage, guardianIndices);

    tx = await source.wormholeRelayer.setDefaultDeliveryProvider(inverseFirstSignedVaa);
    await tx.wait();

    const originalDefaultDeliveryProvider = (await source.wormholeRelayer.getDefaultDeliveryProvider());

    expect(originalDefaultDeliveryProvider).toBe(currentAddress);

});


test("Governance: Test Upgrading Contract", async () => {
  const IMPLEMENTATION_STORAGE_SLOT = "0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc";

  const getImplementationAddress = () => source.provider.getStorageAt(source.wormholeRelayer.address, IMPLEMENTATION_STORAGE_SLOT);

  console.log(`Current Implementation address: ${(await getImplementationAddress())}`);

  const wormholeAddress = CONTRACTS[network][sourceChain].core || "";

  const newWormholeRelayerImplementationAddress = (await new ethers_contracts.WormholeRelayer__factory(source.wallet).deploy(wormholeAddress).then((x)=>x.deployed())).address;

  console.log(`Deployed!`);
  console.log(`New core relayer implementation: ${newWormholeRelayerImplementationAddress}`);

  const timestamp = (await source.wallet.provider.getBlock("latest")).timestamp;
  const chain = source.chainId;
  const firstMessage = governance.publishWormholeRelayerUpgradeContract(timestamp, chain, newWormholeRelayerImplementationAddress);
  const firstSignedVaa = guardians.addSignatures(firstMessage, guardianIndices);

  let tx = await source.wormholeRelayer.submitContractUpgrade(firstSignedVaa);

  expect(ethers.utils.getAddress((await getImplementationAddress()).substring(26))).toBe(ethers.utils.getAddress(newWormholeRelayerImplementationAddress));
});
});
