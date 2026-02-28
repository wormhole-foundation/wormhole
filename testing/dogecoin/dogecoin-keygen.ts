import * as bitcoin from "bitcoinjs-lib";
import * as ecc from "tiny-secp256k1";
import { ECPairFactory } from "ecpair";

bitcoin.initEccLib(ecc);
const ECPair = ECPairFactory(ecc);

// Dogecoin Testnet network parameters
const dogecoinTestnet: bitcoin.Network = {
  messagePrefix: "\x19Dogecoin Signed Message:\n",
  bech32: "tdge",
  bip32: {
    public: 0x043587cf,
    private: 0x04358394,
  },
  pubKeyHash: 0x71, // addresses start with 'n'
  scriptHash: 0xc4,
  wif: 0xf1, // WIF prefix for testnet
};

// Generate a new keypair
const keyPair = ECPair.makeRandom({ network: dogecoinTestnet });

// Get private key in WIF format
const privateKeyWIF = keyPair.toWIF();

// Get public key
const publicKey = Buffer.from(keyPair.publicKey).toString("hex");

// Generate address (P2PKH)
const { address } = bitcoin.payments.p2pkh({
  pubkey: new Uint8Array(Buffer.from(keyPair.publicKey)),
  network: dogecoinTestnet,
});

// Save to JSON file
const keypairData = {
  network: "dogecoin-testnet",
  privateKeyWIF,
  publicKey,
  address,
  generatedAt: new Date().toISOString(),
};

await Bun.write(
  "dogecoin-testnet-keypair.json",
  JSON.stringify(keypairData, null, 2),
);

console.log("Keypair saved to dogecoin-testnet-keypair.json");
