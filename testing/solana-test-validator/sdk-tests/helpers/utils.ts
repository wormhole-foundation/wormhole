import { PublicKey } from "@solana/web3.js";

// used to use solana cli
const { execSync } = require("child_process");

export function now() {
  return Math.floor(Date.now() / 1000);
}

export function ethAddressToBuffer(address: string) {
  return Buffer.concat([
    Buffer.alloc(12),
    Buffer.from(address.substring(2), "hex"),
  ]);
}

interface Erc721Token {
  address: string;
  tokenId: bigint;
  name: string;
  symbol: string;
  uri: string;
}

export function makeErc721Token(
  address: string,
  tokenId: bigint,
  name: string,
  symbol: string,
  uri: string
): Erc721Token {
  return {
    address,
    tokenId,
    name,
    symbol,
    uri,
  };
}

export function deployProgram(
  keyPath: string,
  artifactPath: string,
  programIdPath: string,
  programId: PublicKey, // could derive it from programIdPath, but whatevs
  upgradeAuthority: PublicKey
) {
  // deploy
  execSync(
    `solana -k ${keyPath} program deploy ${artifactPath} --program-id ${programIdPath}`
  );

  // set upgrade authority
  execSync(
    `solana -k ${keyPath} program set-upgrade-authority ${programId.toString()} --new-upgrade-authority ${upgradeAuthority.toString()}`
  );
}

export function execSolanaWriteBufferAndSetBufferAuthority(
  keyPath: string,
  artifactPath: string,
  upgradeAuthority: PublicKey
): PublicKey {
  // solana program write-buffer
  const buffer = (() => {
    const output = execSync(
      `solana -k ${keyPath} program write-buffer ${artifactPath} -u localhost`
    );
    return new PublicKey(output.toString().match(/^.{8}([A-Za-z0-9]+)/)[1]);
  })();

  // solana program set-buffer-authority
  execSync(
    `solana -k ${keyPath} program set-buffer-authority ${buffer.toString()} --new-buffer-authority ${upgradeAuthority.toString()} -u localhost`
  );
  return buffer;
}
