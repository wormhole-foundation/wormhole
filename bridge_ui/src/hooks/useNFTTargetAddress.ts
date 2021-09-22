import { useMemo } from "react";
import { useSelector } from "react-redux";
import { selectNFTTargetAddressHex } from "../store/selectors";
import { hexToUint8Array } from "@certusone/wormhole-sdk";

export default function useNFTTargetAddressHex() {
  const targetAddressHex = useSelector(selectNFTTargetAddressHex);
  const targetAddress = useMemo(
    () => (targetAddressHex ? hexToUint8Array(targetAddressHex) : undefined),
    [targetAddressHex]
  );
  return targetAddress;
}
