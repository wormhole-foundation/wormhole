import { Connection, Keypair, PublicKey } from '@solana/web3.js';
import {
  encoding,
} from '@wormhole-foundation/sdk-connect';

import { privateKeyToEvmAddress, TestingWormholeCore } from "../tests/testing-wormhole-core.js";

import yargs from "yargs";
import { hideBin } from 'yargs/helpers';

import { inspect } from "util";

async function main() {
  const parser = yargs(hideBin(process.argv))
    .epilogue("Note that this is only meant to be used in a testing environment. DO NOT USE IN PRODUCTION.")
    .config()
    .option("guardianKeys", {
      description: "Array with guardian private keys ",
      array: true,
      demandOption: true,
      type: "string",
    })
    .option('programId', {
      description: 'Program id of the v1 wormhole core',
      demandOption: true,
      type: 'string',
    })
    .option('url', {
      description: 'URL of the Solana RPC',
      default: "https://api.devnet.solana.com",
      type: 'string',
    })
    .option('expirationTime', {
      description: 'Expiration time for guardian set',
      default: 86400,
      type: 'number',
    })
    .option('coreFee', {
      description: 'Wormhole core publish message fee',
      default: 100,
      type: 'number',
    })
    .option('signer', {
      description: 'Signer private key in number array',
      demandOption: true,
      array: true,
      type: 'number',
    });
  const args = await parser.parse();

  const signer = Keypair.fromSecretKey(Uint8Array.from(args.signer));
  const connection = new Connection(args.url, "confirmed");
  const coreV1ProgramId = new PublicKey(args.programId);


  const coreV1 = new TestingWormholeCore(
    signer,
    connection,
    "Testnet",
    coreV1ProgramId,
    { coreBridge: args.programId },
  );

  const guardianAddresses = args.guardianKeys
    .map(encoding.hex.decode)
    .map(privateKeyToEvmAddress)
    .map((key) => encoding.hex.encode(key, true));
  console.log(`Guardian addresses: [${guardianAddresses.join(", ")}]`)


  const accounts = await connection.getProgramAccounts(coreV1ProgramId)
  console.log(`Core accounts: ${inspect(accounts)}`)
  // assert(accounts.length === 2, "Expected 2 accounts")

  const guardianSetIndex = await coreV1.client.getGuardianSetIndex()
  console.log(`Core current guardian set: ${inspect(guardianSetIndex)}`)
  // assert(guardianSetIndex === 0, "Expected guardian set index to be 0")
  const guardianSet = await coreV1.client.getGuardianSet(guardianSetIndex);

  console.log(`Core guardian set ${inspect(guardianSet.index)}:`)
  // assert(guardianSet.index === 0, "Expected guardian set index to be 0")
  console.log(`${inspect(guardianSet.keys.join(", "))}`)
  // assert(guardianSet.keys.length === 1, "Expected guardian set keys to have length 1")

  // const queriedGuardian = new UniversalAddress(guardianSet.keys[0], "hex")
  // const expectedGuardian = toUniversal("Ethereum", guardianAddress)
  // assert(queriedGuardian.equals(expectedGuardian), "Expected guardian set keys to be the devnet guardian")

  // const txid = await coreV1.initialize(args.guardianKeys, args.expirationTime, args.coreFee);
  // console.log(`Sent initialize tx ${txid}`);
}


main().catch((error) => {
  console.error(error.stack || error);
  process.exit(1);
})
