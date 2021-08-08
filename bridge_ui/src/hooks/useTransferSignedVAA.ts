import { useMemo } from "react";
import { useSelector } from "react-redux";
import { selectSignedVAAHex } from "../store/selectors";
import { hexToUint8Array } from "../utils/array";

export default function useTransferSignedVAA() {
  const signedVAAHex = useSelector(selectSignedVAAHex);
  const signedVAA = useMemo(
    () => (signedVAAHex ? hexToUint8Array(signedVAAHex) : undefined),
    [signedVAAHex]
  );
  return signedVAA;
}
