import {
  ChainId,
  getEmitterAddressEth,
  uint8ArrayToHex,
} from "@certusone/wormhole-sdk";
import getSignedVAAWithRetry from "@certusone/wormhole-sdk/lib/esm/rpc/getSignedVAAWithRetry";
import axios from "axios";
import { useEffect, useMemo, useState } from "react";
import {
  getTokenBridgeAddressForChain,
  WORMHOLE_RPC_HOSTS,
} from "../utils/consts";

const RELAYER_ENDPOINT_URL = "http://localhost:3111/relay";

export type RelayRequest = {
  signedVaa: string;
  chainId: ChainId; //This is the target chain
  unwrapNative: boolean;
};

export default function useRestRelayer(
  sourceChain: ChainId,
  sourceSequence: string,
  targetChain: ChainId
) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [complete, setComplete] = useState(false);

  useEffect(() => {
    if (!sourceChain || !sourceSequence || !targetChain) {
      setLoading(false);
      setError("");
      setComplete(false);
      return;
    }
    let cancelled = false;
    console.log(
      "Relay action triggered: ",
      sourceChain,
      targetChain,
      sourceSequence
    );
    setLoading(true);
    setError("");
    getSignedVAAWithRetry(
      WORMHOLE_RPC_HOSTS,
      sourceChain,
      getEmitterAddressEth(getTokenBridgeAddressForChain(sourceChain)),
      sourceSequence
    )
      .then((VAA) => {
        const signedVaaHex = uint8ArrayToHex(VAA.vaaBytes);
        console.log("got Vaa with retry.");
        if (VAA) {
          axios
            .post(RELAYER_ENDPOINT_URL, {
              signedVAA: signedVaaHex,
              chainId: targetChain,
              unwrapNative: true,
            })
            .then((result) => {
              if (!cancelled) {
                setComplete(true);
                setLoading(false);
              }
            });
        }
      })
      .catch((error) => {
        if (!cancelled) {
          console.error(error);
          setError("Unable to relay the VAA");
          setLoading(false);
        }
      });
  }, [sourceChain, sourceSequence, targetChain]);

  const output = useMemo(() => {
    return { isLoading: loading, error, isComplete: complete };
  }, [loading, error, complete]);

  return output;
}
