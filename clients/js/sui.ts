import { textToHexString } from '@certusone/wormhole-sdk';
import { Ed25519Keypair, JsonRpcProvider, RawSigner } from '@mysten/sui.js';
import { NETWORKS } from "./networks";
const bs58 = require('bs58')

// Generate a new Keypair
let privkey: string | undefined = NETWORKS["DEVNET"]["sui"].key;
if (privkey === undefined) {
  throw new Error("No privkey for sui");
}

let pubkey: string | undefined = NETWORKS["DEVNET"]["sui"].pubkey;
if (pubkey === undefined) {
  throw new Error("No pubkey for sui");
}

console.log("pubkey is: ", pubkey)

let rpc: string | undefined = NETWORKS["DEVNET"]["sui"].rpc;
if (rpc === undefined) {
  throw new Error("No rpc for sui");
}

//console.log("decoded privkey key is: ", bs58.decode(privkey))

let public_key = new Uint8Array(Buffer.from(pubkey, "hex"))
let private_key = new Uint8Array(Buffer.from(privkey))

console.log("public_key is: ", public_key)


const keypair = new Ed25519Keypair({"publicKey" : public_key, "secretKey": private_key});
console.log("keypair: ", keypair)

const provider = new JsonRpcProvider(rpc);
const signer = new RawSigner(keypair, provider);


async function callEntryFunc() {
    const moveCallTxn = await signer.executeMoveCall({
        packageObjectId: '0x2',
        module: 'devnet_nft',
        function: 'mint',
        typeArguments: [],
        arguments: [
          'Example NFT',
          'An NFT created by the wallet Command Line Tool',
          'ipfs://bafkreibngqhl3gaa7daob4i2vccziay2jjlp435cf66vhono7nrvww53ty',
        ],
        gasBudget: 20000,
      });
      return moveCallTxn
}

callEntryFunc().then(x => {console.log(x)})
