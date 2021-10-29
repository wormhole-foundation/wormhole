import {
  hexToNativeString,
  parseTransferPayload,
} from "@certusone/wormhole-sdk";
import { formatUnits } from "@ethersproject/units";
import axios from "axios";
import { useEffect, useMemo, useState } from "react";
import { DataWrapper } from "../store/helpers";
import useTVL from "./useTVL";

function convertbase64ToBinary(base64: string) {
  var raw = window.atob(base64);
  var rawLength = raw.length;
  var array = new Uint8Array(new ArrayBuffer(rawLength));

  console.log(rawLength, "rawlength");

  for (let i = 0; i < rawLength; i++) {
    array[i] = raw.charCodeAt(i);
  }
  return array;
}

//Don't actually mount this hook, it's way to expensive for the prod site.
const useTotalTransactedAmount = (): DataWrapper<number> => {
  const tvl = useTVL();
  const [everyVaaPayloadInHistory, setEveryVaaPayloadInHistory] = useState<
    { EmitterChain: string; EmitterAddress: string; Payload: string }[] | null
  >(null);

  useEffect(() => {
    const URL = "http://localhost:8080/recent?numRows=15000";
    let response: {
      EmitterChain: string;
      EmitterAddress: string;
      Payload: string;
    }[] = [];

    axios.get(URL).then((result) => {
      const payload = result?.data["*"];
      response = payload;
      setEveryVaaPayloadInHistory(response as any);
    });
  }, []);

  const output = useMemo(() => {
    const emittersThatMatter = [
      `ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5`, //SOLANA TOKEN
      `0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585`, //ETH token
      `0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2`, //terra
      `000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7`, //bsc
    ];

    if (!everyVaaPayloadInHistory || tvl.isFetching || !tvl.data) {
      return 0;
    }

    let total = 0;

    everyVaaPayloadInHistory.forEach((result) => {
      const isImportant = emittersThatMatter.find(
        (x) => x.toLowerCase() === result.EmitterAddress.toLowerCase()
      );

      if (!isImportant) {
        return;
      }

      console.log("about to parse", result.Payload);
      let payload;
      try {
        payload = parseTransferPayload(
          Buffer.from(convertbase64ToBinary(result.Payload))
        );
      } catch (e) {
        console.log("parse fail");
        console.log(e);
      }

      if (!payload) {
        return;
      }

      const assetAddress =
        hexToNativeString(payload.originAddress, payload.originChain) || "";

      const tvlItem = tvl.data?.find((item) => {
        return (
          assetAddress &&
          item.assetAddress.toLowerCase() === assetAddress.toLowerCase()
        );
      });

      if (!assetAddress || !tvlItem) {
        return;
      }

      const quote = tvlItem?.quotePrice;
      const decimals =
        tvlItem?.decimals === undefined || tvlItem?.decimals === null
          ? null
          : tvlItem.decimals > 8
          ? 8
          : tvlItem.decimals;
      const amount =
        decimals != null && formatUnits(payload.amount.toString(), decimals);

      const valueAdd =
        quote && amount && parseFloat(amount) && quote * parseFloat(amount);
      console.log("value add", valueAdd);

      total = total + (valueAdd || 0);
    });

    return total;
  }, [everyVaaPayloadInHistory, tvl.isFetching, tvl.data]);

  return {
    data: output,
    isFetching: tvl.isFetching || output === 0,
    error: "",
    receivedAt: null,
  };
};

export default useTotalTransactedAmount;
