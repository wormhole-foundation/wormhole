import { Keypair } from "@solana/web3.js";
import bs58 from "bs58";

// Generate a new keypair
const keypair = Keypair.generate();

// Get private key in base58 format (common format for Solana wallets)
const privateKeyBase58 = bs58.encode(keypair.secretKey);

// Get private key as byte array (used by Solana CLI)
const privateKeyArray = Array.from(keypair.secretKey);

// Get public key (this is also the address in Solana)
const publicKey = keypair.publicKey.toBase58();

// Save to JSON file
const keypairData = {
  network: "solana-devnet",
  privateKeyBase58,
  privateKeyArray,
  publicKey,
  address: publicKey, // In Solana, address and public key are the same
  generatedAt: new Date().toISOString(),
};

await Bun.write(
  "solana-devnet-keypair.json",
  JSON.stringify(keypairData, null, 2),
);

console.log("Keypair saved to solana-devnet-keypair.json");
