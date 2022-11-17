// import { textToHexString } from '@certusone/wormhole-sdk';
// import { Ed25519Keypair, JsonRpcProvider, RawSigner } from '@mysten/sui.js';
// import { NETWORKS } from "./networks";

import { Ed25519Keypair } from '@mysten/sui.js';
let private_key = new Uint8Array(Buffer.from("AGEL2sY5slGBrnl7Fyob6YN3kiTjzzrakHDZIGcg66VNjkJIyXxMqwdReEZrCrJpgbyQUSMkGID/0RRCcOB+JtE=", 'base64'))
private_key = private_key.slice(1) //first byte is 00, indicating that it is ed25519
console.log(Ed25519Keypair.fromSecretKey(private_key))

// // Generate a new Keypair
// let privkey: string | undefined = NETWORKS["DEVNET"]["sui"].key;
// if (privkey === undefined) {
//   throw new Error("No privkey for sui");
// }

// let rpc: string | undefined = NETWORKS["DEVNET"]["sui"].rpc;
// if (rpc === undefined) {
//   throw new Error("No rpc for sui");
// }

// //console.log("decoded privkey key is: ", bs58.decode(privkey))

// //let public_key = new Uint8Array(Buffer.from(pubkey, "hex"))
// console.log(Buffer.from(privkey))

//let private_key = new Uint8Array(Buffer.from("AGEL2sY5slGBrnl7Fyob6YN3kiTjzzrakHDZIGcg66VNjkJIyXxMqwdReEZrCrJpgbyQUSMkGID/0RRCcOB+JtE=", 'base64'))
//private_key = private_key.slice(1)
//console.log("private key len: ", private_key.length)
//console.log("public_key is: ", public_key)
//console.log("private_key is: ", private_key)

//Ed25519Keypair.fromSecretKey(new Uint8Array(Buffer.from("00c7b92e86c16fd62f38db61d6ec504251115eb2a6ee3bb7af6c84e2be83034b926adeca51fed8b9872d397dcfa55f7f9fecf1ea818ca935fcc4a0d5f951029969")))

//console.log(Ed25519Keypair.generate())
//console.log(Ed25519Keypair.fromSecretKey(private_key))
// const keypair = new Ed25519Keypair({"publicKey" : public_key, "secretKey": private_key});
// console.log("keypair: ", keypair)

// const provider = new JsonRpcProvider(rpc);
// const signer = new RawSigner(keypair, provider);


// async function callEntryFunc() {
//     const moveCallTxn = await signer.executeMoveCall({
//         packageObjectId: '0x2',
//         module: 'devnet_nft',
//         function: 'mint',
//         typeArguments: [],
//         arguments: [
//           'Example NFT',
//           'An NFT created by the wallet Command Line Tool',
//           'ipfs://bafkreibngqhl3gaa7daob4i2vccziay2jjlp435cf66vhono7nrvww53ty',
//         ],
//         gasBudget: 20000,
//       });
//       return moveCallTxn
// }

// callEntryFunc().then(x => {console.log(x)})
