import { Ed25519Keypair, JsonRpcProvider, RawSigner } from '@mysten/sui.js';
// Generate a new Secp256k1 Keypair
const keypair = new Ed25519Keypair();

const signer = new RawSigner(
  keypair,
  new JsonRpcProvider('https://gateway.devnet.sui.io:443')
);

console.log(keypair)