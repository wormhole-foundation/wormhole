import { Secp256k1Keypair, JsonRpcProvider, RawSigner } from '@mysten/sui.js';
// Generate a new Secp256k1 Keypair
const keypair = new Secp256k1Keypair();

const signer = new RawSigner(
  keypair,
  new JsonRpcProvider('https://gateway.devnet.sui.io:443')
);
