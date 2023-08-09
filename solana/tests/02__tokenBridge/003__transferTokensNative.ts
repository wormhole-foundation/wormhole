import * as anchor from "@coral-xyz/anchor";
import { ethers } from "ethers";
import {
  GUARDIAN_KEYS,
  InvalidAccountConfig,
  InvalidArgConfig,
  expectDeepEqual,
  expectIxErr,
  expectIxOk,
  expectIxOkDetails,
} from "../helpers";
import * as coreBridge from "../helpers/coreBridge";
import * as tokenBridge from "../helpers/tokenBridge";
import { expect } from "chai";
import {
  NATIVE_MINT,
  createAssociatedTokenAccount,
  createMint,
  getAssociatedTokenAddressSync,
  mintTo,
} from "@solana/spl-token";
import { PublicKey } from "@solana/web3.js";

describe("Token Bridge -- Instruction: Transfer Tokens (Native)", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(
    connection,
    tokenBridge.getProgramId("B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE")
  );
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(
    connection,
    tokenBridge.getProgramId("wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb")
  );

  const mint = anchor.web3.Keypair.generate();
  const srcToken = getAssociatedTokenAddressSync(
    mint.publicKey,
    payer.publicKey
  );

  before("Set Up Mint and Token Accounts", async () => {
    await createMint(
      connection,
      payer,
      payer.publicKey,
      payer.publicKey,
      9,
      mint
    );

    await createAssociatedTokenAccount(
      connection,
      payer,
      mint.publicKey,
      payer.publicKey
    );

    await mintTo(
      connection,
      payer,
      mint.publicKey,
      srcToken,
      payer,
      BigInt("1000000000000000")
    );
  });

  describe("Ok", () => {
    it("Invoke `transfer_tokens_native`", async () => {
      const amount = new anchor.BN("88888888");
      const relayerFee = new anchor.BN("11111111");
      const [coreMessage, txDetails, forkCoreMessage, forkTxDetails] =
        await parallelTxDetails(
          program,
          forkedProgram,
          { payer: payer.publicKey, mint: mint.publicKey, srcToken },
          defaultArgs(amount, relayerFee),
          payer
        );

      // TODO: Check message accounts.
    });
  });
});

function defaultArgs(amount: anchor.BN, relayerFee: anchor.BN) {
  return {
    nonce: 420,
    amount,
    relayerFee,
    recipient: Array.from(Buffer.alloc(32, "deadbeef", "hex")),
    recipientChain: 2,
  };
}

async function parallelTxDetails(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  accounts: { payer: PublicKey; mint: PublicKey; srcToken: PublicKey },
  args: tokenBridge.LegacyTransferTokensArgs,
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  const { payer: owner, srcToken: token } = accounts;
  const { amount } = args;
  const coreMessage = anchor.web3.Keypair.generate();
  const approveIx = tokenBridge.approveTransferAuthorityIx(
    program,
    token,
    owner,
    amount
  );
  const ix = tokenBridge.legacyTransferTokensNativeIx(
    program,
    {
      coreMessage: coreMessage.publicKey,
      coreBridgeProgram: coreBridge.getProgramId(
        "Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o"
      ),
      ...accounts,
    },
    args
  );

  const forkCoreMessage = anchor.web3.Keypair.generate();
  const forkedApproveIx = tokenBridge.approveTransferAuthorityIx(
    forkedProgram,
    token,
    owner,
    amount
  );
  const forkedIx = tokenBridge.legacyTransferTokensNativeIx(
    forkedProgram,
    {
      coreMessage: forkCoreMessage.publicKey,
      coreBridgeProgram: coreBridge.getProgramId(
        "worm2ZoG2kUd4vFXhvjh93UUH596ayRfgQ2MgjNMTth"
      ),
      ...accounts,
    },
    args
  );

  const [txDetails, forkTxDetails] = await Promise.all([
    expectIxOkDetails(connection, [approveIx, ix], [payer, coreMessage]),
    expectIxOkDetails(
      connection,
      [forkedApproveIx, forkedIx],
      [payer, forkCoreMessage]
    ),
  ]);

  return [coreMessage, txDetails, forkCoreMessage, forkTxDetails];
}
