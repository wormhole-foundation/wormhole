import { createInitializeInstruction } from "@certusone/wormhole-sdk/lib/cjs/solana/tokenBridge";
import { AnchorProvider, Wallet, web3 } from "@coral-xyz/anchor";
import { bs58 } from "@coral-xyz/anchor/dist/cjs/utils/bytes";

function displayHelp() {
    console.log(`
  
      The following environment variables are required:
      
      RPC_URL: The RPC URL of the SVM network (Fogo or Solana).
      TOKEN_BRIDGE_PROGRAM_ID: The program ID of the token bridge program.
      PRIVATE_KEY: The private key of the account that will be used to initialize the token bridge program.
                   Can be a keypair-file path or a base58 encoded string.
      CORE_BRIDGE_PROGRAM_ID: The program ID of the Core bridge program.
      `)
}

(async () => {
  const RPC_URL = process.env.RPC_URL;
  if (!RPC_URL) {
    console.error("RPC_URL is required");
    displayHelp();
    process.exit(1);
  }

  const TOKEN_BRIDGE_PROGRAM_ID = process.env.TOKEN_BRIDGE_PROGRAM_ID;
  if (!TOKEN_BRIDGE_PROGRAM_ID) {
    console.error("TOKEN_BRIDGE_PROGRAM_ID is required");
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
  const tokenBridgeAddress = new web3.PublicKey(TOKEN_BRIDGE_PROGRAM_ID);

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

  const ix = createInitializeInstruction(
    tokenBridgeAddress,
    payer.publicKey.toString(),
    coreBridgeAddress
  );

  const transaction = new web3.Transaction();
  transaction.add(ix);
  const tx = await provider.sendAndConfirm(transaction);
  console.log(tx);
})();