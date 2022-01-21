import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  isEVMChain,
} from "@certusone/wormhole-sdk";
import { Provider } from "@ethersproject/abstract-provider";
import { formatUnits } from "@ethersproject/units";
import { Typography } from "@material-ui/core";
import { LocalGasStation } from "@material-ui/icons";
import { Connection, PublicKey } from "@solana/web3.js";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import {
  getDefaultNativeCurrencySymbol,
  SOLANA_HOST,
  TERRA_HOST,
} from "../utils/consts";
import { getMultipleAccountsRPC } from "../utils/solana";
import { NATIVE_TERRA_DECIMALS } from "../utils/terra";
import useIsWalletReady from "./useIsWalletReady";
import { LCDClient } from "@terra-money/terra.js";

export type GasEstimate = {
  currentGasPrice: string;
  lowEstimate: string;
  highEstimate: string;
};

export type MethodType = "nft" | "createWrapped" | "transfer";

//It's difficult to project how many fees the user will accrue during the
//workflow, as a variable number of transactions can be sent, and different
//execution paths can be hit in the smart contracts, altering gas used.
//As such, for the moment it is best to just check for a reasonable 'low balance' threshold.
//Still it would be good to calculate a reasonable value at runtime based off current gas prices,
//rather than a hardcoded value.
const SOLANA_THRESHOLD_LAMPORTS: bigint = BigInt(300000);
const ETHEREUM_THRESHOLD_WEI: bigint = BigInt(35000000000000000);
const TERRA_THRESHOLD_ULUNA: bigint = BigInt(100000);
const TERRA_THRESHOLD_UUSD: bigint = BigInt(10000000);

const isSufficientBalance = (
  chainId: ChainId,
  balance: bigint | undefined,
  terraFeeDenom?: string
) => {
  if (balance === undefined || !chainId) {
    return true;
  }
  if (CHAIN_ID_SOLANA === chainId) {
    return balance > SOLANA_THRESHOLD_LAMPORTS;
  }
  if (isEVMChain(chainId)) {
    return balance > ETHEREUM_THRESHOLD_WEI;
  }
  if (terraFeeDenom === "uluna") {
    return balance > TERRA_THRESHOLD_ULUNA;
  }
  if (terraFeeDenom === "uusd") {
    return balance > TERRA_THRESHOLD_UUSD;
  }

  return true;
};

type TerraBalance = {
  denom: string;
  balance: bigint;
};

const isSufficientBalanceTerra = (balances: TerraBalance[]) => {
  return balances.some(({ denom, balance }) => {
    if (denom === "uluna") {
      return balance > TERRA_THRESHOLD_ULUNA;
    }
    if (denom === "uusd") {
      return balance > TERRA_THRESHOLD_UUSD;
    }
    return false;
  });
};

//TODO move to more generic location
const getBalanceSolana = async (walletAddress: string) => {
  const connection = new Connection(SOLANA_HOST);
  return getMultipleAccountsRPC(connection, [
    new PublicKey(walletAddress),
  ]).then(
    (results) => {
      if (results.length && results[0]) {
        return BigInt(results[0].lamports);
      }
    },
    (error) => {
      return BigInt(0);
    }
  );
};

const getBalanceEvm = async (walletAddress: string, provider: Provider) => {
  return provider.getBalance(walletAddress).then((result) => result.toBigInt());
};

const getBalancesTerra = async (walletAddress: string) => {
  const TARGET_DENOMS = ["uluna", "uusd"];

  const lcd = new LCDClient(TERRA_HOST);
  return lcd.bank
    .balance(walletAddress)
    .then((coins) => {
      const balances = coins
        .filter(({ denom }) => {
          return TARGET_DENOMS.includes(denom);
        })
        .map(({ amount, denom }) => {
          return {
            denom,
            balance: BigInt(amount.toString()),
          };
        });
      if (balances) {
        return balances;
      } else {
        return Promise.reject();
      }
    })
    .catch((e) => {
      return Promise.reject();
    });
};

const toBalanceString = (balance: bigint | undefined, chainId: ChainId) => {
  if (!chainId || balance === undefined) {
    return "";
  }
  if (isEVMChain(chainId)) {
    return formatUnits(balance, 18); //wei decimals
  } else if (chainId === CHAIN_ID_SOLANA) {
    return formatUnits(balance, 9); //lamports to sol decmals
  } else if (chainId === CHAIN_ID_TERRA) {
    return formatUnits(balance, NATIVE_TERRA_DECIMALS);
  }
};

export default function useTransactionFees(chainId: ChainId) {
  const { walletAddress, isReady } = useIsWalletReady(chainId);
  const { provider } = useEthereumProvider();
  const [balance, setBalance] = useState<bigint | undefined>(undefined);
  const [terraBalances, setTerraBalances] = useState<TerraBalance[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");

  const loadStart = useCallback(() => {
    setBalance(undefined);
    setIsLoading(true);
    setError("");
  }, []);

  useEffect(() => {
    if (chainId === CHAIN_ID_SOLANA && isReady && walletAddress) {
      loadStart();
      getBalanceSolana(walletAddress).then(
        (result) => {
          const adjustedresult =
            result === undefined || result === null ? BigInt(0) : result;
          setIsLoading(false);
          setBalance(adjustedresult);
        },
        (error) => {
          setIsLoading(false);
          setError("Cannot load wallet balance");
        }
      );
    } else if (isEVMChain(chainId) && isReady && walletAddress) {
      if (provider) {
        loadStart();
        getBalanceEvm(walletAddress, provider).then(
          (result) => {
            const adjustedresult =
              result === undefined || result === null ? BigInt(0) : result;
            setIsLoading(false);
            setBalance(adjustedresult);
          },
          (error) => {
            setIsLoading(false);
            setError("Cannot load wallet balance");
          }
        );
      }
    } else if (chainId === CHAIN_ID_TERRA && isReady && walletAddress) {
      loadStart();
      getBalancesTerra(walletAddress).then(
        (results) => {
          const adjustedResults = results.map(({ denom, balance }) => {
            return {
              denom,
              balance:
                balance === undefined || balance === null ? BigInt(0) : balance,
            };
          });
          setIsLoading(false);
          setTerraBalances(adjustedResults);
        },
        (error) => {
          setIsLoading(false);
          setError("Cannot load wallet balance");
        }
      );
    }
  }, [provider, walletAddress, isReady, chainId, loadStart]);

  const results = useMemo(() => {
    return {
      isSufficientBalance:
        chainId === CHAIN_ID_TERRA
          ? isSufficientBalanceTerra(terraBalances)
          : isSufficientBalance(chainId, balance),
      balance,
      balanceString: toBalanceString(balance, chainId),
      isLoading,
      error,
    };
  }, [balance, terraBalances, chainId, isLoading, error]);

  return results;
}

export function useEthereumGasPrice(contract: MethodType, chainId: ChainId) {
  const { provider } = useEthereumProvider();
  const { isReady } = useIsWalletReady(chainId);
  const [estimateResults, setEstimateResults] = useState<GasEstimate | null>(
    null
  );

  useEffect(() => {
    if (provider && isReady && !estimateResults) {
      getGasEstimates(provider, contract).then(
        (results) => {
          setEstimateResults(results);
        },
        (error) => {
          console.log(error);
        }
      );
    }
  }, [provider, isReady, estimateResults, contract]);

  const results = useMemo(() => estimateResults, [estimateResults]);
  return results;
}

function EthGasEstimateSummary({
  methodType,
  chainId,
}: {
  methodType: MethodType;
  chainId: ChainId;
}) {
  const estimate = useEthereumGasPrice(methodType, chainId);
  if (!estimate) {
    return null;
  }

  return (
    <Typography
      component="div"
      style={{
        display: "flex",
        alignItems: "center",
        marginTop: 8,
        flexWrap: "wrap",
      }}
    >
      <div style={{ display: "flex", alignItems: "center" }}>
        <LocalGasStation fontSize="inherit" />
        &nbsp;{estimate.currentGasPrice}
      </div>
      <div>&nbsp;&nbsp;&nbsp;</div>
      <div>
        Est. Fees: {estimate.lowEstimate} - {estimate.highEstimate}{" "}
        {getDefaultNativeCurrencySymbol(chainId)}
      </div>
    </Typography>
  );
}

const terraEstimatesByContract = {
  transfer: {
    lowGasEstimate: BigInt(50000),
    highGasEstimate: BigInt(90000),
  },
};

const evmEstimatesByContract = {
  transfer: {
    lowGasEstimate: BigInt(80000),
    highGasEstimate: BigInt(130000),
  },
  nft: {
    lowGasEstimate: BigInt(350000),
    highGasEstimate: BigInt(500000),
  },
  createWrapped: {
    lowGasEstimate: BigInt(450000),
    highGasEstimate: BigInt(700000),
  },
};

export async function getGasEstimates(
  provider: Provider,
  contract: MethodType
): Promise<GasEstimate | null> {
  const lowEstimateGasAmount = evmEstimatesByContract[contract].lowGasEstimate;
  const highEstimateGasAmount =
    evmEstimatesByContract[contract].highGasEstimate;

  let lowEstimate;
  let highEstimate;
  let currentGasPrice;
  if (provider) {
    const priceInWei = await provider.getGasPrice();
    if (priceInWei) {
      lowEstimate = parseFloat(
        formatUnits(lowEstimateGasAmount * priceInWei.toBigInt(), "ether")
      ).toFixed(4);
      highEstimate = parseFloat(
        formatUnits(highEstimateGasAmount * priceInWei.toBigInt(), "ether")
      ).toFixed(4);
      currentGasPrice = parseFloat(formatUnits(priceInWei, "gwei")).toFixed(0);
    }
  }

  const output =
    currentGasPrice && highEstimate && lowEstimate
      ? {
          currentGasPrice,
          lowEstimate,
          highEstimate,
        }
      : null;

  return output;
}

function TerraGasEstimateSummary({
  methodType,
  chainId,
}: {
  methodType: MethodType;
  chainId: ChainId;
}) {
  if (methodType === "transfer") {
    const lowEstimate = formatUnits(
      terraEstimatesByContract.transfer.lowGasEstimate,
      NATIVE_TERRA_DECIMALS
    );
    const highEstimate = formatUnits(
      terraEstimatesByContract.transfer.highGasEstimate,
      NATIVE_TERRA_DECIMALS
    );
    return (
      <Typography
        component="div"
        style={{
          display: "flex",
          alignItems: "center",
          marginTop: 8,
          flexWrap: "wrap",
        }}
      >
        <div>
          Est. Fees: {lowEstimate} - {highEstimate}{" "}
          {getDefaultNativeCurrencySymbol(chainId)}
        </div>
      </Typography>
    );
  } else {
    return null;
  }
}

export function GasEstimateSummary({
  methodType,
  chainId,
}: {
  methodType: MethodType;
  chainId: ChainId;
}) {
  if (isEVMChain(chainId)) {
    return <EthGasEstimateSummary chainId={chainId} methodType={methodType} />;
  } else if (chainId === CHAIN_ID_TERRA) {
    return (
      <TerraGasEstimateSummary chainId={chainId} methodType={methodType} />
    );
  } else {
    return null;
  }
}
