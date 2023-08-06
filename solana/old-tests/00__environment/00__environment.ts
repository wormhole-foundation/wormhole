import { web3 } from "@coral-xyz/anchor";
import { expect } from "chai";
import * as fs from "fs";
import { BpfProgramData, ProgramData } from "wormhole-solana-sdk";
import {
  CORE_BRIDGE_PROGRAM_ID,
  LOCALHOST,
  TOKEN_BRIDGE_PROGRAM_ID,
  airdrop,
  artifactsPath,
  coreBridgeKeyPath,
  deployProgram,
  removeTmpPath,
  tmpPath,
  tokenBridgeKeyPath,
} from "../helpers";
import {
  Metadata,
  PROGRAM_ID as TOKEN_METADATA_PROGRAM_ID,
} from "@metaplex-foundation/mpl-token-metadata";
import { NATIVE_MINT } from "@solana/spl-token";

describe("Environment", () => {
  // This is the first test that runs. We will purge the tmp directory if it exists.
  removeTmpPath();

  const connection = new web3.Connection(LOCALHOST, "processed");

  // Deployer for all programs.
  //
  // NOTE: After initialization, this keypair will no longer be able to upgrade accounts because
  // the upgrade authority will be each program's upgrade PDA.
  const deployerSigner = web3.Keypair.generate();

  // Write keypair to temporary file.
  const deployerKeypath = `${tmpPath()}/deployer.json`;
  fs.writeFileSync(
    deployerKeypath,
    JSON.stringify(Array.from(deployerSigner.secretKey))
  );

  // For convenience.
  const deployer = deployerSigner.publicKey;

  // Make sure the deployer has some SOL.
  it("Airdrop SOL", async () => {
    const lamports = await airdrop(connection, deployer);

    const balance = await connection.getBalance(deployer);
    expect(balance).equals(lamports);
  });

  it(`Verify ${NATIVE_MINT.toString()} Token Metadata`, async () => {
    const [metadataPubkey, _] = web3.PublicKey.findProgramAddressSync(
      [
        Buffer.from("metadata"),
        TOKEN_METADATA_PROGRAM_ID.toBuffer(),
        NATIVE_MINT.toBuffer(),
      ],
      TOKEN_METADATA_PROGRAM_ID
    );

    const metadata = await Metadata.fromAccountAddress(
      connection,
      metadataPubkey
    );
    expect(metadata.data.symbol).equals("SOL".padEnd(10, "\u0000"));
    expect(metadata.data.name).equals("Wrapped SOL".padEnd(32, "\u0000"));
  });

  it.skip("Create NFTs", async () => {
    // TODO
  });

  it.skip("Create NFT Metadata", async () => {
    // TODO
  });

  it("Deploy Core Bridge", async () => {
    const output = deployProgram(
      deployerKeypath,
      `${artifactsPath()}/solana_wormhole_core_bridge.so`,
      coreBridgeKeyPath()
    );
    expect(output).equals(`Program Id: ${CORE_BRIDGE_PROGRAM_ID.toString()}\n`);

    const addr = ProgramData.address(CORE_BRIDGE_PROGRAM_ID);
    const programMetadata = await ProgramData.fromAccountAddress(
      connection,
      addr
    );

    const programData = programMetadata as BpfProgramData;
    expect(programData.upgradeAuthorityAddress!.equals(deployer)).is.true;
  });

  it.skip("Deploy Token Bridge", async () => {
    const output = deployProgram(
      deployerKeypath,
      `${artifactsPath()}/solana_wormhole_token_bridge.so`,
      tokenBridgeKeyPath()
    );
    expect(output).equals(
      `Program Id: ${TOKEN_BRIDGE_PROGRAM_ID.toString()}\n`
    );

    const addr = ProgramData.address(TOKEN_BRIDGE_PROGRAM_ID);
    const programMetadata = await ProgramData.fromAccountAddress(
      connection,
      addr
    );

    const programData = programMetadata as BpfProgramData;
    expect(programData.upgradeAuthorityAddress!.equals(deployer)).is.true;
  });

  it.skip("Deploy NFT Bridge", async () => {
    // TODO
  });
});
