declare module "near-seed-phrase" {
  export function parseSeedPhrase(
    seedPhrase: string,
    derivationPath?: string
  ): {
    seedPhrase: string;
    secretKey: string;
    publicKey: string;
  };
}
