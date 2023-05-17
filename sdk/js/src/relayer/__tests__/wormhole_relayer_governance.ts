import { afterAll, beforeEach, describe, expect, jest, test} from "@jest/globals";
import { PublicKey } from "@solana/web3.js";

import { ethers } from "ethers";
import {getAddressInfo} from "../consts" 
import {getDefaultProvider} from "../main/helpers"
import {
    relayer,
    ethers_contracts,
    tryNativeToUint8Array,
    ChainId
  } from "../../../";

  import {GovernanceEmitter, MockGuardians} from "../../../src/mock";
import { error } from "console";

const env = process.env['ENV'];
if(!env) throw Error("No env specified: tilt or ci or testnet or mainnet");
const network = env == 'tilt' || env == 'ci' ? "DEVNET" : env == 'testnet' ? "TESTNET" : env == 'mainnet' ? "MAINNET" : undefined;
if(!network) throw Error(`Invalid env specified: ${env}`);
const sourceChainId = network == 'DEVNET' ? 2 : 6;
const targetChainId = network == 'DEVNET' ? 4 : 14;

// Devnet Private Key
const privateKey = "4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"

const sourceAddressInfo = getAddressInfo(sourceChainId, network);
const sourceProvider = getDefaultProvider(network, sourceChainId, env == 'ci');

// signers
const walletSource = new ethers.Wallet(privateKey, sourceProvider);

const sourceCoreRelayerAddress = sourceAddressInfo.coreRelayerAddress;

if(!sourceCoreRelayerAddress) throw Error("No source core relayer address");

const sourceCoreRelayer = ethers_contracts.CoreRelayer__factory.connect(
  sourceCoreRelayerAddress,
  walletSource
);


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


describe("Wormhole Relayer Governance Action Tests", () => {

    test("Test Registering Chain", async () => {

        const currentAddress = await sourceCoreRelayer.getRegisteredCoreRelayerContract(6);
        console.log(`For Chain 2, registered chain 6 address: ${currentAddress}`);

        const expectedNewRegisteredAddress = "0x0000000000000000000000001234567890123456789012345678901234567892";

        const timestamp = (await walletSource.provider.getBlock("latest")).timestamp;
        const chain = 6;
        const firstMessage = governance.publishWormholeRelayerRegisterChain(timestamp, chain, expectedNewRegisteredAddress)
        const firstSignedVaa = guardians.addSignatures(firstMessage, [0]);

        let tx = await sourceCoreRelayer.registerCoreRelayerContract(firstSignedVaa, {gasLimit: 500000});
        await tx.wait();

        const newRegisteredAddress = (await sourceCoreRelayer.getRegisteredCoreRelayerContract(6));

        expect(newRegisteredAddress).toBe(expectedNewRegisteredAddress);

        const inverseFirstMessage = governance.publishWormholeRelayerRegisterChain(timestamp, chain, currentAddress)
        const inverseFirstSignedVaa = guardians.addSignatures(inverseFirstMessage, [0]);

        tx = await sourceCoreRelayer.registerCoreRelayerContract(inverseFirstSignedVaa, {gasLimit: 500000});
        await tx.wait();

        const secondRegisteredAddress = (await sourceCoreRelayer.getRegisteredCoreRelayerContract(6));

        expect(secondRegisteredAddress).toBe(currentAddress);
    })

    test("Test Setting Default Relay Provider", async () => {

        const currentAddress = await sourceCoreRelayer.getDefaultRelayProvider();
        console.log(`For Chain 2, default relay provider: ${currentAddress}`);

        const expectedNewDefaultRelayProvider = "0x1234567890123456789012345678901234567892";

        const timestamp = (await walletSource.provider.getBlock("latest")).timestamp;
        const chain = 2;
        const firstMessage = governance.publishWormholeRelayerSetDefaultRelayProvider(timestamp, chain, expectedNewDefaultRelayProvider);
        const firstSignedVaa = guardians.addSignatures(firstMessage, [0]);

        let tx = await sourceCoreRelayer.setDefaultRelayProvider(firstSignedVaa, {gasLimit: 500000});
        await tx.wait();

        const newDefaultRelayProvider = (await sourceCoreRelayer.getDefaultRelayProvider());

        expect(newDefaultRelayProvider).toBe(expectedNewDefaultRelayProvider);

        const inverseFirstMessage = governance.publishWormholeRelayerSetDefaultRelayProvider(timestamp, chain, currentAddress)
        const inverseFirstSignedVaa = guardians.addSignatures(inverseFirstMessage, [0]);

        tx = await sourceCoreRelayer.setDefaultRelayProvider(inverseFirstSignedVaa, {gasLimit: 500000});
        await tx.wait();

        const originalDefaultRelayProvider = (await sourceCoreRelayer.getDefaultRelayProvider());

        expect(originalDefaultRelayProvider).toBe(currentAddress);

    });

    /*
    test("Test Upgrading Contract", async () => {
      const defaultRelayProvider = await sourceCoreRelayer.getDefaultRelayProvider();

      const newCoreRelayer = await new ethers_contracts.CoreRelayer__factory(walletSource).deploy(sourceAddressInfo., "0x2468013579246801357924680135792468013579")
      const currentAddress = await sourceCoreRelayer.getDefaultRelayProvider();
      console.log(`For Chain 2, default relay provider: ${currentAddress}`);

      const expectedNewDefaultRelayProvider = "0x1234567890123456789012345678901234567892";

      const timestamp = (await walletSource.provider.getBlock("latest")).timestamp;
      const chain = 2;
      const firstMessage = governance.publishWormholeRelayerSetDefaultRelayProvider(timestamp, chain, expectedNewDefaultRelayProvider);
      const firstSignedVaa = guardians.addSignatures(firstMessage, [0]);

      let tx = await sourceCoreRelayer.setDefaultRelayProvider(firstSignedVaa, {gasLimit: 500000});
      await tx.wait();

      const newDefaultRelayProvider = (await sourceCoreRelayer.getDefaultRelayProvider());

      expect(newDefaultRelayProvider).toBe(expectedNewDefaultRelayProvider);

      const inverseFirstMessage = governance.publishWormholeRelayerSetDefaultRelayProvider(timestamp, chain, currentAddress)
      const inverseFirstSignedVaa = guardians.addSignatures(inverseFirstMessage, [0]);

      tx = await sourceCoreRelayer.setDefaultRelayProvider(inverseFirstSignedVaa, {gasLimit: 500000});
      await tx.wait();

      const originalDefaultRelayProvider = (await sourceCoreRelayer.getDefaultRelayProvider());

      expect(originalDefaultRelayProvider).toBe(currentAddress);

  });*/

});
