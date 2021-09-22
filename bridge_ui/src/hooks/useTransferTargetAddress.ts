import { useMemo } from "react";
import { useSelector } from "react-redux";
import { selectTransferTargetAddressHex } from "../store/selectors";
import { hexToUint8Array } from "@certusone/wormhole-sdk";

export default function useTransferTargetAddressHex() {
  const targetAddressHex = useSelector(selectTransferTargetAddressHex);
  const targetAddress = useMemo(
    () => (targetAddressHex ? hexToUint8Array(targetAddressHex) : undefined),
    [targetAddressHex]
  );
  return targetAddress;
}
