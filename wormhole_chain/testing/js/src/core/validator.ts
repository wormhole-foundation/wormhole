import {
  Secp256k1HdWallet,
  SigningCosmosClient,
  Msg,
  coins,
} from "@cosmjs/launchpad";
let elliptic = require("elliptic"); //No TS defs?

export function registerValidatorAddressMsg(
  address: string,
  publicKey: string,
  privateKey: string
): Msg {
  const ec = new elliptic.ec("secp256k1");
  const key = ec.keyFromPrivate(privateKey);
  const signature = key.sign(address, { canonical: true });

  return {
    type: "x/wormhole/RegisterValidator",
    value: {
      guardianKey: publicKey,
      signature: signature,
    },
  };
}
