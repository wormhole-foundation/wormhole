import { createInitializeInstruction } from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";
import { AnchorProvider, Wallet, web3 } from "@coral-xyz/anchor";
import { bs58 } from "@coral-xyz/anchor/dist/cjs/utils/bytes";

function displayHelp() {
  console.log(`

    The following environment variables are required:
    
    RPC_URL: The RPC URL of the SVM network (Fogo or Solana).
    CORE_BRIDGE_PROGRAM_ID: The program ID of the core bridge program.
    PRIVATE_KEY: The private key of the account that will be used to initialize the core bridge program.
                 Can be a keypair-file path or a base58 encoded string.
    GUARDIAN_SET: A JSON-array containing one or more guardian addresses, in non-0x prefixed hex format. 

    Optional environment variables:
    FEE: The fee that will be used to initialize the core bridge program. Defaults to 100000 lamports.
    EXPIRATION_TIME: The expiration time of the guardian set. Defaults to 86400 seconds (1 day).

    `)
}

const DEFAULT_FEE = 100000;
const DEFAULT_EXPIRATION_TIME = 86400;

(async () => {
  const RPC_URL = process.env.RPC_URL;
  if (!RPC_URL) {
    console.error("RPC_URL is required");
    displayHelp();
    process.exit(1);
  }

  const CORE_BRIDGE_PROGRAM_ID = process.env.CORE_BRIDGE_PROGRAM_ID;
  if (!CORE_BRIDGE_PROGRAM_ID) {
    console.error("CORE_BRIDGE_PROGRAM_ID is required");
    displayHelp();
    process.exit(1);
  }

  const coreBridgeAddress = new web3.PublicKey(CORE_BRIDGE_PROGRAM_ID);

  const connection = new web3.Connection(RPC_URL, "confirmed");

  const key = process.env.PRIVATE_KEY;

  if (!key) {
    console.error("PRIVATE_KEY is required");
    displayHelp();
    process.exit(1);
  }

  const payer = web3.Keypair.fromSecretKey(
    key.endsWith(".json") ? new Uint8Array(require(key)) : bs58.decode(key)
  );
  const provider = new AnchorProvider(connection, new Wallet(payer));

  const guardianAddress = process.env.GUARDIAN_SET;
  if (!guardianAddress) {
    console.error("GUARDIAN_SET is required");
    displayHelp();
    process.exit(1);
  }
  const fee = process.env.FEE || DEFAULT_FEE;
  const expirationTime = process.env.EXPIRATION_TIME || DEFAULT_EXPIRATION_TIME;

  if (BigInt(fee) === 0n) {
    console.error("FEE must be greater than 0");
    displayHelp();
    process.exit(1);
  }

  if (Number(expirationTime) === 0) {
    console.error("EXPIRATION_TIME must be greater than 0");
    displayHelp();
    process.exit(1);
  }

  // Parse the guardian set
  const guardianSet = JSON.parse(guardianAddress);
  if (!Array.isArray(guardianSet)) {
    console.error("GUARDIAN_SET must be a JSON-array");
    displayHelp();
    process.exit(1);
  }
  if (guardianSet.length === 0) {
    console.error("GUARDIAN_SET must contain at least one guardian address");
    displayHelp();
    process.exit(1);
  }

  const guardianSetBuffer = guardianSet.map((guardian) => Buffer.from(guardian, "hex"));

  if (guardianSetBuffer.some((address) => address.length !== 20)) {
    console.error("GUARDIAN_SET must only contain non-0x prefixed, 20-byte long hex addresses");
    displayHelp();
    process.exit(1);
  }

  const ix = createInitializeInstruction(
    coreBridgeAddress,
    payer.publicKey.toString(),
    Number(expirationTime),
    BigInt(fee),
    guardianSetBuffer
  );
  const transaction = new web3.Transaction();
  transaction.add(ix);
  const tx = await provider.sendAndConfirm(transaction);
  console.log(tx);
})();
