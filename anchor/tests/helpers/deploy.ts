import { Keypair, PublicKey } from "@solana/web3.js";
import { execSync } from "child_process";
import * as fs from "fs";
import { LOCALHOST } from "./consts";
import { tmpPath } from "./misc";

export function deployProgram(
  deployerKeypath: string,
  artifactPath: string,
  programKeyPath: string
) {
  // Deploy program using solana CLI.
  const output = execSync(
    `solana program deploy -u ${LOCALHOST} -k ${deployerKeypath} ${artifactPath} --program-id ${programKeyPath}`,
    { stdio: "pipe", encoding: "utf-8" }
  ).toString();

  return output.substring(0, output.length - 1);
}

export function loadProgramBpf(
  publisher: Keypair,
  artifactPath: string,
  bufferAuthority: PublicKey
): PublicKey {
  // Write keypair to temporary file.
  const keypath = `${tmpPath()}/payer_${new Date().toISOString()}.json`;
  fs.writeFileSync(keypath, JSON.stringify(Array.from(publisher.secretKey)));

  // Invoke BPF Loader Upgradeable `write-buffer` instruction.
  const buffer = (() => {
    const output = execSync(
      `solana -k ${keypath} program write-buffer ${artifactPath} -u localhost`
    );
    return new PublicKey(output.toString().match(/^.{8}([A-Za-z0-9]+)/)[1]);
  })();

  // Invoke BPF Loader Upgradeable `set-buffer-authority` instruction.
  execSync(
    `solana -k ${keypath} program set-buffer-authority ${buffer.toString()} --new-buffer-authority ${bufferAuthority.toString()} -u localhost`
  );

  // Return the pubkey for the buffer (our new program implementation).
  return buffer;
}
