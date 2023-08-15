import * as anchor from "@coral-xyz/anchor";
import { Account, getOrCreateAssociatedTokenAccount } from "@solana/spl-token";
import {
  ETHEREUM_TOKEN_BRIDGE_ADDRESS,
  MINT_INFO_WRAPPED_7,
  MINT_INFO_WRAPPED_8,
  WrappedMintInfo,
  expectIxOkDetails,
  getTokenBalances,
  parallelPostVaa,
  expectIxErr,
} from "../helpers";
import {
  CHAIN_ID_SOLANA,
  tryNativeToHexString,
  parseVaa,
  CHAIN_ID_ETH,
  tryUint8ArrayToNative,
} from "@certusone/wormhole-sdk";
import { GUARDIAN_KEYS } from "../helpers";
import * as tokenBridge from "../helpers/tokenBridge";
import * as coreBridge from "../helpers/coreBridge";
import { MockTokenBridge, MockGuardians } from "@certusone/wormhole-sdk/lib/cjs/mock";
import { expect } from "chai";

const GUARDIAN_SET_INDEX = 2;
const dummyTokenBridge = new MockTokenBridge(
  tryNativeToHexString(ETHEREUM_TOKEN_BRIDGE_ADDRESS, 2),
  2,
  0, // Consistency level
  690 // Starting sequence
);
const guardians = new MockGuardians(GUARDIAN_SET_INDEX, GUARDIAN_KEYS);

describe("Token Bridge -- Legacy Instruction: Complete Transfer (Native)", () => {
  anchor.setProvider(anchor.AnchorProvider.env());

  const provider = anchor.getProvider() as anchor.AnchorProvider;
  const connection = provider.connection;
  const program = tokenBridge.getAnchorProgram(connection, tokenBridge.localnet());
  const wormholeProgram = coreBridge.getAnchorProgram(connection, coreBridge.localnet());
  const payer = (provider.wallet as anchor.Wallet).payer;

  const forkedProgram = tokenBridge.getAnchorProgram(connection, tokenBridge.mainnet());

  const wrappedMints: WrappedMintInfo[] = [MINT_INFO_WRAPPED_8, MINT_INFO_WRAPPED_7];

  describe("Ok", () => {
    for (const { chain, decimals, address } of wrappedMints.slice(0, 1)) {
      it(`Invoke \`complete_transfer_wrapped\` (${decimals} Decimals, No Fee)`, async () => {
        const [mint, forkMint] = [program, forkedProgram].map((program) =>
          tokenBridge.wrappedMintPda(program.programId, chain, Array.from(address))
        );
        // Create recipient token account.
        const recipient = anchor.web3.Keypair.generate();
        const [recipientToken, forkRecipientToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, recipient.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, recipient.publicKey),
        ]);

        const [payerToken, forkPayerToken] = await Promise.all([
          getOrCreateAssociatedTokenAccount(connection, payer, mint, payer.publicKey),
          getOrCreateAssociatedTokenAccount(connection, payer, forkMint, payer.publicKey),
        ]);

        // Amounts.
        const amount = BigInt(699999);
        const fee = BigInt(0);

        // Create the signed transfer VAA.
        const [signedVaa, forkSignedVaa] = [recipientToken, forkRecipientToken].map((acct) =>
          getSignedTransferVaa(address, amount, fee, acct.address)
        );

        // Fetch balances before.
        const [recipientBalancesBefore, relayerBalancesBefore] = await Promise.all([
          getTokenBalances(
            program,
            forkedProgram,
            recipientToken.address,
            forkRecipientToken.address
          ),
          getTokenBalances(program, forkedProgram, payerToken.address, forkPayerToken.address),
        ]);

        // Complete the transfer.
        await parallelTxDetails(
          program,
          forkedProgram,
          {
            recipientToken,
            forkRecipientToken,
          },
          {
            signedVaa,
            forkSignedVaa,
          },
          payer
        );

        // Check recipient and relayer token balance changes.
        // await Promise.all([
        //   tokenBridge.expectCorrectTokenBalanceChanges(
        //     connection,
        //     recipientToken.address,
        //     recipientBalancesBefore,
        //     tokenBridge.TransferDirection.In
        //   ),
        //   tokenBridge.expectCorrectRelayerBalanceChanges(
        //     connection,
        //     payerToken.address,
        //     relayerBalancesBefore,
        //     fee * BigInt(10 ** (decimals - 8))
        //   ),
        // ]);
      });
    }
  });
});

function getSignedTransferVaa(
  tokenAddress: Uint8Array,
  amount: bigint,
  fee: bigint,
  recipient: anchor.web3.PublicKey,
  targetChain?: number
): Buffer {
  const vaaBytes = dummyTokenBridge.publishTransferTokens(
    Buffer.from(tokenAddress).toString("hex"),
    CHAIN_ID_ETH,
    amount,
    targetChain ?? CHAIN_ID_SOLANA,
    recipient.toBuffer().toString("hex"),
    fee
  );
  return guardians.addSignatures(vaaBytes, [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12]);
}

async function parallelTxDetails(
  program: tokenBridge.TokenBridgeProgram,
  forkedProgram: tokenBridge.TokenBridgeProgram,
  tokenAccounts: {
    recipientToken: Account;
    forkRecipientToken: Account;
  },
  signedVaas: { signedVaa: Buffer; forkSignedVaa: Buffer },
  payer: anchor.web3.Keypair
) {
  const connection = program.provider.connection;
  const { recipientToken, forkRecipientToken } = tokenAccounts;
  const { signedVaa, forkSignedVaa } = signedVaas;

  // Post the VAA.
  const parsed = await parallelPostVaa(connection, payer, signedVaa);
  const forkParsed = await parallelPostVaa(connection, payer, forkSignedVaa);

  // Create instruction.
  const ix = tokenBridge.legacyCompleteTransferWrappedIx(
    program,
    { payer: payer.publicKey, recipientToken: recipientToken.address },
    parsed
  );
  const forkedIx = tokenBridge.legacyCompleteTransferWrappedIx(
    forkedProgram,
    { payer: payer.publicKey, recipientToken: forkRecipientToken.address },
    forkParsed
  );
  return expectIxOkDetails(connection, [ix, forkedIx], [payer]);
}
