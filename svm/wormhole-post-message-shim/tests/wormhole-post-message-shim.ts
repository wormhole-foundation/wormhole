import * as anchor from "@coral-xyz/anchor";
import { Program } from "@coral-xyz/anchor";
import WormholePostMessageShimIdl from "../target/idl/wormhole_post_message_shim.json";
import { WormholePostMessageShim } from "../target/types/wormhole_post_message_shim";
import { expect } from "chai";
import { bs58 } from "@coral-xyz/anchor/dist/cjs/utils/bytes";

async function getTransactionDetails(
  tx: string
): Promise<anchor.web3.VersionedTransactionResponse> {
  let txDetails: anchor.web3.VersionedTransactionResponse | null = null;
  while (!txDetails) {
    txDetails = await anchor.getProvider().connection.getTransaction(tx, {
      maxSupportedTransactionVersion: 0,
      commitment: "confirmed",
    });
  }
  return txDetails;
}

async function getSequenceFromTx(tx: string): Promise<bigint> {
  const txDetails = await getTransactionDetails(tx);

  const borshEventCoder = new anchor.BorshEventCoder(
    WormholePostMessageShimIdl as any
  );

  const innerInstructions = txDetails.meta.innerInstructions[0].instructions;

  // Get the last instruction from the inner instructions
  const lastInstruction = innerInstructions[innerInstructions.length - 1];

  // Decode the Base58 encoded data
  const decodedData = bs58.decode(lastInstruction.data);

  // Remove the instruction discriminator and re-encode the rest as Base58
  const eventData = Buffer.from(decodedData.subarray(8)).toString("base64");

  const borshEvents = borshEventCoder.decode(eventData);
  return BigInt(borshEvents.data.sequence.toString());
}

describe("wormhole-post-message-shim", () => {
  // Configure the client to use the local cluster.
  anchor.setProvider(anchor.AnchorProvider.env());

  const program = anchor.workspace
    .WormholePostMessageShim as Program<WormholePostMessageShim>;

  const postMessage = async (msg: string): Promise<string> =>
    await program.methods
      .postMessage({
        nonce: 0,
        payload: Buffer.from(msg, "ascii"),
        consistencyLevel: { confirmed: {} },
      })
      .accounts({
        emitter: program.provider.publicKey,
        sequence: anchor.web3.PublicKey.findProgramAddressSync(
          [Buffer.from("Sequence"), program.provider.publicKey.toBuffer()],
          new anchor.web3.PublicKey(
            "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
          )
        )[0],
        // these are needed if removing the address checks
        // bridge: new anchor.web3.PublicKey(
        //   "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
        // ),
        // feeCollector: new anchor.web3.PublicKey(
        //   "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
        // ),
        // clock: new anchor.web3.PublicKey(
        //   "SysvarC1ock11111111111111111111111111111111"
        // ),
        // rent: new anchor.web3.PublicKey(
        //   "SysvarRent111111111111111111111111111111111"
        // ),
      })
      .preInstructions([
        // gotta pay the fee
        anchor.web3.SystemProgram.transfer({
          fromPubkey: program.provider.publicKey,
          toPubkey: new anchor.web3.PublicKey(
            "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
          ), // fee collector
          lamports: 100, // hardcoded for tilt in devnet_setup.sh
        }),
      ])
      .rpc();

  it("Posts a message!", async () => {
    const tx = await postMessage("hello world");
    console.log("Your transaction signature", tx);
    expect(await getSequenceFromTx(tx)).to.equal(BigInt(0));
  });
  it("Posts a second message!", async () => {
    const tx = await postMessage("hello world");
    console.log("Your transaction signature", tx);
    expect(await getSequenceFromTx(tx)).to.equal(BigInt(1));
  });
  it("Compares core post_message to shim post_message!", async () => {
    async function logCostAndCompute(method: string, tx: string) {
      const SOL_PRICE = 217.54; // 2025-01-03
      const txDetails = await getTransactionDetails(tx);
      const lamports =
        txDetails.meta.preBalances[0] - txDetails.meta.postBalances[0];
      const sol = lamports / 1_000_000_000;
      console.log(
        `${method}: lamports ${lamports} SOL ${sol}, $${sol * SOL_PRICE}, CU ${
          txDetails.meta.computeUnitsConsumed
        }`
      );
    }
    {
      const acct = new anchor.web3.Keypair();
      const data = Buffer.from(
        "01000000000b00000068656c6c6f20776f726c6400",
        "hex"
      );
      const transaction = new anchor.web3.Transaction();
      transaction.add(
        anchor.web3.SystemProgram.transfer({
          fromPubkey: program.provider.publicKey,
          toPubkey: new anchor.web3.PublicKey(
            "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
          ), // fee collector
          lamports: 100, // hardcoded for tilt in devnet_setup.sh
        })
      );
      transaction.add(
        new anchor.web3.TransactionInstruction({
          keys: [
            {
              // config
              isSigner: false,
              isWritable: true,
              pubkey: new anchor.web3.PublicKey(
                "FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP"
              ),
            },
            {
              // message
              isSigner: true,
              isWritable: true,
              pubkey: acct.publicKey,
            },
            {
              // emitter
              isSigner: true,
              isWritable: false,
              pubkey: program.provider.publicKey,
            },
            {
              // sequence
              isSigner: false,
              isWritable: true,
              pubkey: anchor.web3.PublicKey.findProgramAddressSync(
                [
                  Buffer.from("Sequence"),
                  program.provider.publicKey.toBuffer(),
                ],
                new anchor.web3.PublicKey(
                  "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
                )
              )[0],
            },
            {
              // payer
              isSigner: true,
              isWritable: true,
              pubkey: program.provider.publicKey,
            },
            {
              // fee collector
              isSigner: false,
              isWritable: true,
              pubkey: new anchor.web3.PublicKey(
                "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
              ),
            },
            {
              // clock
              isSigner: false,
              isWritable: false,
              pubkey: new anchor.web3.PublicKey(
                "SysvarC1ock11111111111111111111111111111111"
              ),
            },
            {
              // system program
              isSigner: false,
              isWritable: false,
              pubkey: new anchor.web3.PublicKey(
                "11111111111111111111111111111111"
              ),
            },
            {
              // rent
              isSigner: false,
              isWritable: false,
              pubkey: new anchor.web3.PublicKey(
                "SysvarRent111111111111111111111111111111111"
              ),
            },
          ],
          programId: new anchor.web3.PublicKey(
            "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
          ),
          data,
        })
      );
      const tx = await program.provider.sendAndConfirm(transaction, [acct]);
      await logCostAndCompute("core", tx);
    }
    {
      const tx = await postMessage("hello world");
      await logCostAndCompute("shim", tx);
    }
  });
});
