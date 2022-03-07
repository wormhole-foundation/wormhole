import { EncodeObject } from "@cosmjs/proto-signing";
import { getQueryClient, getStargateClient } from "./walletHelpers";
let elliptic = require("elliptic"); //No TS defs?
import { StakingExtension } from "@cosmjs/stargate";

export function registerValidatorAddressMsg(
  address: string,
  publicKey: string,
  privateKey: string
): EncodeObject {
  const ec = new elliptic.ec("secp256k1");
  const key = ec.keyFromPrivate(privateKey);
  const signature = key.sign(address, { canonical: true });

  return {
    typeUrl: "x/wormhole/RegisterValidator",
    value: {
      guardianKey: publicKey,
      signature: signature,
    },
  };
}

/*
Will attempt to return an array of all the registered validators. These are not necessarily all active.
*/
export async function getValidators() {
  const client = await getQueryClient();
  //TODO handle pagination here
  const validators = await client.staking.validators("BOND_STATUS_BONDED");

  return validators;
}
