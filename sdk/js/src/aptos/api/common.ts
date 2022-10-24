import { Types } from "aptos";

// Contract upgrade

export const authorizeUpgrade = (
  address: string,
  vaa: Uint8Array
): Types.EntryFunctionPayload => {
  if (!address) throw new Error("Need bridge address.");
  return {
    function: `${address}::contract_upgrade::submit_vaa_entry`,
    type_arguments: [],
    arguments: [vaa],
  };
};

export const upgradeContract = (
  address: string,
  metadataSerialized: Uint8Array,
  code: Array<Uint8Array>
): Types.EntryFunctionPayload => {
  if (!address) throw new Error("Need bridge address.");
  return {
    function: `${address}::contract_upgrade::upgrade`,
    type_arguments: [],
    arguments: [metadataSerialized, code],
  };
};

export const migrateContract = (
  address: string
): Types.EntryFunctionPayload => {
  if (!address) throw new Error("Need bridge address.");
  return {
    function: `${address}::contract_upgrade::migrate`,
    type_arguments: [],
    arguments: [],
  };
};
