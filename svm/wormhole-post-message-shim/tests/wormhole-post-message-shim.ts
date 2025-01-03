import * as anchor from "@coral-xyz/anchor";
import { Program } from "@coral-xyz/anchor";
import WormholePostMessageShimIdl from "../target/idl/wormhole_post_message_shim.json";
import { WormholePostMessageShim } from "../target/types/wormhole_post_message_shim";
import { expect } from "chai";
import { bs58 } from "@coral-xyz/anchor/dist/cjs/utils/bytes";

async function getSequenceFromTx(tx: string): Promise<bigint> {
  // Fetch the transaction details
  let txDetails: anchor.web3.VersionedTransactionResponse | null = null;
  while (!txDetails) {
    txDetails = await anchor.getProvider().connection.getTransaction(tx, {
      maxSupportedTransactionVersion: 0,
      commitment: "confirmed",
    });
  }

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
  console.log(borshEvents);
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
  it("Posts a third message!", async () => {
    const tx = await postMessage("hello world");
    console.log("Your transaction signature", tx);
    expect(await getSequenceFromTx(tx)).to.equal(BigInt(2));
  });
});
