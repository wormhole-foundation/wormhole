import {
  AnchorProvider,
  Program,
  setProvider,
  Wallet,
  web3,
} from "@coral-xyz/anchor";
import WormholePostMessageShimIdl from "../idls/wormhole_post_message_shim.json";
import { WormholePostMessageShim } from "../idls/wormhole_post_message_shim";
import WormholeIntegratorExampleIdl from "./idls/devnet/wormhole_integrator_example.json";
import { WormholeIntegratorExample } from "./idls/devnet/wormhole_integrator_example";
import { getSequenceFromTx } from "./helpers";
import { getSignedVAAWithRetry } from "@certusone/wormhole-sdk/lib/cjs/rpc";
import { parseVaa } from "@certusone/wormhole-sdk/lib/cjs/vaa/wormhole";
import { NodeHttpTransport } from "@improbable-eng/grpc-web-node-http-transport";
import { getSequenceTracker } from "@certusone/wormhole-sdk/lib/cjs/solana/wormhole";

(async () => {
  const SOLANA_RPC_URL = "http://127.0.0.1:8899";
  const GUARDIAN_URL = "http://127.0.0.1:7071";

  const CORE_BRIDGE_PROGRAM_ID = new web3.PublicKey(
    "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
  );

  const connection = new web3.Connection(SOLANA_RPC_URL, "confirmed");

  const getShimProgramWithNewSigner = async () => {
    const payer = web3.Keypair.generate();
    {
      const tx = await connection.requestAirdrop(payer.publicKey, 10000000000);
      await connection.confirmTransaction({
        ...(await connection.getLatestBlockhash()),
        signature: tx,
      });
    }
    const provider = new AnchorProvider(connection, new Wallet(payer));
    setProvider(provider);

    const program = new Program<WormholePostMessageShim>(
      WormholePostMessageShimIdl as WormholePostMessageShim
    );
    return program;
  };

  const getIntegrationProgramWithNewSigner = async () => {
    const payer = web3.Keypair.generate();
    {
      const tx = await connection.requestAirdrop(payer.publicKey, 10000000000);
      await connection.confirmTransaction({
        ...(await connection.getLatestBlockhash()),
        signature: tx,
      });
    }
    const provider = new AnchorProvider(connection, new Wallet(payer));
    setProvider(provider);

    const program = new Program<WormholeIntegratorExample>(
      WormholeIntegratorExampleIdl as WormholeIntegratorExample
    );
    return program;
  };

  const postMessage = async (
    program: Program<WormholePostMessageShim>,
    msg: string,
    consistencyLevel: number
  ): Promise<string> =>
    await program.methods
      .postMessage(
        0,
        consistencyLevel === 0 ? { confirmed: {} } : { finalized: {} },
        Buffer.from(msg, "ascii")
      )
      .accounts({
        emitter: program.provider.publicKey,
        wormholeProgram: CORE_BRIDGE_PROGRAM_ID,
      })
      .preInstructions([
        // gotta pay the fee
        web3.SystemProgram.transfer({
          fromPubkey: program.provider.publicKey,
          toPubkey: new web3.PublicKey(
            "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
          ), // fee collector
          lamports: 100, // hardcoded for tilt in devnet_setup.sh
        }),
      ])
      .rpc();

  const testTopLevel = async (
    program: Program<WormholePostMessageShim>,
    msg: string,
    expectedSeq: bigint,
    consistencyLevel: number,
    testDescription: string
  ) => {
    const tx = await postMessage(program, msg, consistencyLevel);
    const emitter = program.provider.publicKey.toBuffer().toString("hex");
    const seq = await getSequenceFromTx(tx).then((evt) =>
      evt.sequence.toString()
    );

    const { vaaBytes } = await getSignedVAAWithRetry(
      [GUARDIAN_URL],
      1,
      emitter,
      seq,
      {
        transport: NodeHttpTransport(),
      },
      500, // every .5 secs
      100 // 100 times, or 50 seconds
    );

    const vaa = parseVaa(vaaBytes);
    if (
      vaa.sequence === expectedSeq &&
      vaa.consistencyLevel === consistencyLevel &&
      vaa.payload.equals(Buffer.from(msg, "ascii"))
    ) {
      console.log(`✅ ${testDescription} success!`);
    } else {
      throw new Error(`❌ ${testDescription} failed!`);
    }
  };

  const testInner = async (
    program: Program<WormholeIntegratorExample>,
    expectedSeq: bigint,
    testDescription: string
  ) => {
    const postShimProgram = new Program<WormholePostMessageShim>(
      WormholePostMessageShimIdl as WormholePostMessageShim
    );
    const emitterBuf = web3.PublicKey.findProgramAddressSync(
      [Buffer.from("emitter")],
      program.programId
    )[0].toBuffer();
    const tx = await program.methods
      .postMessage()
      .accounts({
        // sequence: web3.PublicKey.findProgramAddressSync(
        //   [Buffer.from("Sequence"), emitterBuf],
        //   new web3.PublicKey("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o")
        // )[0],
        wormholePostMessageShimEa: web3.PublicKey.findProgramAddressSync(
          [Buffer.from("__event_authority")],
          postShimProgram.programId
        )[0],
      })
      .preInstructions([
        // gotta pay the fee
        web3.SystemProgram.transfer({
          fromPubkey: program.provider.publicKey,
          toPubkey: new web3.PublicKey(
            "GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs"
          ), // fee collector
          lamports: 100, // hardcoded for tilt in devnet_setup.sh
        }),
      ])
      .rpc();
    const emitter = emitterBuf.toString("hex");
    const seq = await getSequenceFromTx(tx).then((evt) =>
      evt.sequence.toString()
    );

    const { vaaBytes } = await getSignedVAAWithRetry(
      [GUARDIAN_URL],
      1,
      emitter,
      seq,
      {
        transport: NodeHttpTransport(),
      },
      500, // every .5 secs
      100 // 100 times, or 50 seconds
    );

    const vaa = parseVaa(vaaBytes);
    if (
      vaa.sequence === expectedSeq &&
      // ../programs/wormhole-integrator-example/src/instructions/post_message.rs
      vaa.consistencyLevel === 1 &&
      vaa.payload.equals(Buffer.from("your message goes here!", "ascii"))
    ) {
      console.log(`✅ ${testDescription} success!`);
    } else {
      throw new Error(`❌ ${testDescription} failed!`);
    }
  };

  {
    const program = await getShimProgramWithNewSigner();
    await testTopLevel(
      program,
      "hello world",
      BigInt(0),
      0,
      "Top level initial message, confirmed"
    );
    await testTopLevel(
      program,
      "hello everyone",
      BigInt(1),
      0,
      "Top level subsequent message, confirmed"
    );
  }
  {
    const program = await getShimProgramWithNewSigner();
    await testTopLevel(
      program,
      "hello here",
      BigInt(0),
      1,
      "Top level initial message, finalized"
    );
    await testTopLevel(
      program,
      "hello there",
      BigInt(1),
      1,
      "Top level subsequent message, finalized"
    );
  }
  {
    const program = await getIntegrationProgramWithNewSigner();
    let currentSequence = BigInt(0);
    try {
      currentSequence = (
        await getSequenceTracker(
          connection,
          web3.PublicKey.findProgramAddressSync(
            [Buffer.from("emitter")],
            program.programId
          )[0],
          new web3.PublicKey("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o")
        )
      ).sequence;
    } catch (e) {}
    await testInner(
      program,
      currentSequence,
      "Integration initial message, finalized"
    );
    await testInner(
      program,
      currentSequence + BigInt(1),
      "Integration subsequent message, finalized"
    );
  }
})();
