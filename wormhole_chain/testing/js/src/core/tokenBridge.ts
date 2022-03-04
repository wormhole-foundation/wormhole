import {
  Secp256k1HdWallet,
  SigningCosmosClient,
  Msg,
  coins,
} from "@cosmjs/launchpad";

export function redeemVaaMsg(hexVaa: string) {
  return {
    type: "x/tokenBridge/Redeem", //TODO correct type
    value: {
      vaa: hexVaa,
    },
  };
}
