import axios from "axios";
import { useEffect, useState } from "react";
import {
  DataWrapper,
  errorDataWrapper,
  fetchDataWrapper,
  receiveDataWrapper,
} from "../store/helpers";
import { NOTIONAL_TRANSFERRED_URL } from "../utils/consts";

export interface TransferFromData {
  [leavingChainId: string]: number;
}

export interface NotionalTransferredFrom {
  Total: number;
  Daily: {
    [date: string]: TransferFromData;
  };
}

const useNotionalTransferred = () => {
  const [notionalTransferred, setNotionalTransferred] = useState<
    DataWrapper<NotionalTransferredFrom>
  >(fetchDataWrapper());

  useEffect(() => {
    let cancelled = false;
    axios
      .get<NotionalTransferredFrom>(NOTIONAL_TRANSFERRED_URL)
      .then((response) => {
        if (!cancelled) {
          setNotionalTransferred(receiveDataWrapper(response.data));
        }
      })
      .catch((error) => {
        if (!cancelled) {
          setNotionalTransferred(errorDataWrapper(error));
          console.error(error);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return notionalTransferred;
};

export default useNotionalTransferred;
