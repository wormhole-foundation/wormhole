import { Types } from "aptos";
import { ChainId } from "../../utils";

// Guardian set upgrade

export const upgradeGuardianSet = (
  coreBridgeAddress: string,
  vaa: Uint8Array
): Types.EntryFunctionPayload => {
  if (!coreBridgeAddress) throw new Error("Need core bridge address.");
  return {
    function: `${coreBridgeAddress}::guardian_set_upgrade::submit_vaa_entry`,
    type_arguments: [],
    arguments: [vaa],
  };
};

// Init WH

export const initWormhole = (
  coreBridgeAddress: string,
  chainId: ChainId,
  governanceChainId: number,
  governanceContract: Uint8Array,
  initialGuardian: Uint8Array
): Types.EntryFunctionPayload => {
  if (!coreBridgeAddress) throw new Error("Need core bridge address.");
  return {
    function: `${coreBridgeAddress}::wormhole::init`,
    type_arguments: [],
    arguments: [
      chainId,
      governanceChainId,
      governanceContract,
      initialGuardian,
    ],
  };
};
