import { CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import migrateTokensTx from "@certusone/wormhole-sdk/lib/esm/migration/migrateTokens";
import getPoolAddress from "@certusone/wormhole-sdk/lib/esm/migration/poolAddress";
import getToCustodyAddress from "@certusone/wormhole-sdk/lib/esm/migration/toCustodyAddress";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, PublicKey } from "@solana/web3.js";
import { parseUnits } from "ethers/lib/utils";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useAssociatedAccountExistsState } from "../components/SolanaCreateAssociatedAddress";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import useIsWalletReady from "../hooks/useIsWalletReady";
import useMetaplexData from "../hooks/useMetaplexData";
import useSolanaTokenMap from "../hooks/useSolanaTokenMap";
import { DataWrapper } from "../store/helpers";
import { MIGRATION_PROGRAM_ADDRESS, SOLANA_HOST } from "../utils/consts";
import { getMultipleAccounts, signSendAndConfirm } from "../utils/solana";

const getDecimals = async (
  connection: Connection,
  mint: string,
  setter: (decimals: number | undefined) => void
) => {
  setter(undefined);
  if (mint) {
    try {
      const pk = new PublicKey(mint);
      const info = await connection.getParsedAccountInfo(pk);
      // @ts-ignore
      const decimals = info.value?.data.parsed.info.decimals;
      setter(decimals);
    } catch (e) {
      console.log(`Unable to determine decimals of ${mint}`);
    }
  }
};

const getBalance = async (
  connection: Connection,
  address: string | undefined,
  setter: (balance: string | undefined) => void
) => {
  setter(undefined);
  if (address) {
    try {
      const pk = new PublicKey(address);
      const info = await connection.getParsedAccountInfo(pk);
      // @ts-ignore
      const balance = info.value?.data.parsed.info.tokenAmount.uiAmountString;
      setter(balance);
    } catch (e) {
      console.log(`Unable to determine balance of ${address}`);
    }
  }
};

//If the pool doesn't exist in this app, it's an error.
export type SolanaMigratorInformation = {
  poolAddress: string;
  fromMint: string;
  toMint: string;
  fromMintDecimals: number;
  fromAssociatedTokenAccountExists: boolean;
  toAssociatedTokenAccountExists: boolean;
  setToTokenAccountExists: any;
  fromAssociatedTokenAccount: string;
  toAssociatedTokenAccount: string;
  fromAssociatedTokenAccountBalance: string;
  toAssociatedTokenAccountBalance: string | null;
  toCustodyAddress: string;
  toCustodyBalance: string;

  fromName: string | null;
  fromSymbol: string | null;
  fromLogo: string | null;
  toName: string | null;
  toSymbol: string | null;
  toLogo: string | null;

  getNotReadyCause: (amount: string) => string | null;

  migrateTokens: (amount: string) => Promise<string>;
};

//TODO refactor the workflow page to use this hook
export default function useSolanaMigratorInformation(
  fromMint: string,
  toMint: string,
  fromTokenAccount: string
): DataWrapper<SolanaMigratorInformation> {
  const connection = useMemo(
    () => new Connection(SOLANA_HOST, "confirmed"),
    []
  );
  const wallet = useSolanaWallet();
  const { isReady } = useIsWalletReady(CHAIN_ID_SOLANA, false);
  const solanaTokenMap = useSolanaTokenMap();
  const metaplexArray = useMemo(() => [fromMint, toMint], [fromMint, toMint]);
  const metaplexData = useMetaplexData(metaplexArray);

  const [poolAddress, setPoolAddress] = useState("");
  const [poolExists, setPoolExists] = useState<boolean | undefined>(undefined);
  const [fromTokenAccountBalance, setFromTokenAccountBalance] = useState<
    string | undefined
  >(undefined);
  const [toTokenAccount, setToTokenAccount] = useState<string | undefined>(
    undefined
  );
  const [toTokenAccountBalance, setToTokenAccountBalance] = useState<
    string | undefined
  >(undefined);
  const [fromMintDecimals, setFromMintDecimals] = useState<number | undefined>(
    undefined
  );

  const {
    associatedAccountExists: fromTokenAccountExists,
    //setAssociatedAccountExists: setFromTokenAccountExists,
  } = useAssociatedAccountExistsState(
    CHAIN_ID_SOLANA,
    fromMint,
    fromTokenAccount
  );
  const {
    associatedAccountExists: toTokenAccountExists,
    setAssociatedAccountExists: setToTokenAccountExists,
  } = useAssociatedAccountExistsState(CHAIN_ID_SOLANA, toMint, toTokenAccount);

  const [toCustodyAddress, setToCustodyAddress] = useState<string | undefined>(
    undefined
  );
  const [toCustodyBalance, setToCustodyBalance] = useState<string | undefined>(
    undefined
  );

  const [error, setError] = useState("");

  /* Effects
   */
  useEffect(() => {
    getDecimals(connection, fromMint, setFromMintDecimals);
  }, [connection, fromMint]);

  //Retrieve user balance when fromTokenAccount changes
  useEffect(() => {
    // TODO: cancellable
    if (fromTokenAccount && fromTokenAccountExists) {
      getBalance(connection, fromTokenAccount, setFromTokenAccountBalance);
    } else {
      setFromTokenAccountBalance(undefined);
    }
  }, [
    connection,
    fromTokenAccountExists,
    fromTokenAccount,
    setFromTokenAccountBalance,
  ]);

  useEffect(() => {
    // TODO: cancellable
    if (toTokenAccount && toTokenAccountExists) {
      getBalance(connection, toTokenAccount, setToTokenAccountBalance);
    } else {
      setToTokenAccountBalance(undefined);
    }
  }, [
    connection,
    toTokenAccountExists,
    toTokenAccount,
    setFromTokenAccountBalance,
  ]);

  useEffect(() => {
    // TODO: cancellable
    if (toCustodyAddress) {
      getBalance(connection, toCustodyAddress, setToCustodyBalance);
    } else {
      setToCustodyBalance(undefined);
    }
  }, [connection, toCustodyAddress, setToCustodyBalance]);

  //Retrieve pool address on selectedTokens change
  useEffect(() => {
    if (toMint && fromMint) {
      setPoolAddress("");
      setPoolExists(undefined);
      getPoolAddress(MIGRATION_PROGRAM_ADDRESS, fromMint, toMint).then(
        (result) => {
          const key = new PublicKey(result).toString();
          setPoolAddress(key);
        },
        (error) => console.log("Could not calculate pool address.")
      );
    }
  }, [toMint, fromMint, setPoolAddress]);

  //Retrieve the poolAccount every time the pool address changes.
  useEffect(() => {
    if (poolAddress) {
      setPoolExists(undefined);
      try {
        getMultipleAccounts(
          connection,
          [new PublicKey(poolAddress)],
          "confirmed"
        ).then((result) => {
          if (result.length && result[0] !== null) {
            setPoolExists(true);
          } else if (result.length && result[0] === null) {
            setPoolExists(false);
            setError("There is no swap pool for this token.");
          } else {
            setError(
              "unexpected error in fetching pool address. Please reload and try again"
            );
          }
        });
      } catch (e) {
        setError("Could not fetch pool address");
      }
    }
  }, [connection, poolAddress]);

  //Set relevant information derived from poolAddress
  useEffect(() => {
    if (poolAddress) {
      getToCustodyAddress(MIGRATION_PROGRAM_ADDRESS, poolAddress)
        .then((result: any) =>
          setToCustodyAddress(new PublicKey(result).toString())
        )
        .catch((e) => {
          setToCustodyAddress(undefined);
        });
    } else {
      setToCustodyAddress(undefined);
    }
  }, [poolAddress]);

  useEffect(() => {
    if (wallet && wallet.publicKey && toMint) {
      Token.getAssociatedTokenAddress(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        new PublicKey(toMint),
        wallet.publicKey || new PublicKey([])
      ).then(
        (result) => {
          setToTokenAccount(result.toString());
        },
        (error) => {}
      );
    }
  }, [toMint, wallet]);
  /*
      End effects
      */

  const migrateTokens = useCallback(
    async (amount) => {
      const instruction = await migrateTokensTx(
        connection,
        wallet.publicKey?.toString() || "",
        MIGRATION_PROGRAM_ADDRESS,
        fromMint,
        toMint,
        fromTokenAccount || "",
        toTokenAccount || "",
        parseUnits(amount, fromMintDecimals).toBigInt()
      );
      return await signSendAndConfirm(wallet, connection, instruction);
    },
    [
      connection,
      fromMint,
      fromTokenAccount,
      toMint,
      toTokenAccount,
      wallet,
      fromMintDecimals,
    ]
  );

  const fromParse = useCallback(
    (amount: string) => {
      try {
        return parseUnits(amount, fromMintDecimals).toBigInt();
      } catch (e) {
        return BigInt(0);
      }
    },
    [fromMintDecimals]
  );

  const getNotReadyCause = useCallback(
    (amount: string) => {
      const hasRequisiteData = fromMint && toMint && poolAddress && poolExists;
      const accountsReady = fromTokenAccountExists && toTokenAccountExists;
      const amountGreaterThanZero = fromParse(amount) > BigInt(0);
      const sufficientFromTokens =
        fromTokenAccountBalance &&
        amount &&
        fromParse(amount) <= fromParse(fromTokenAccountBalance);
      const sufficientPoolBalance =
        toCustodyBalance &&
        amount &&
        parseFloat(amount) <= parseFloat(toCustodyBalance);

      if (!hasRequisiteData) {
        return "This asset is not supported.";
      } else if (!isReady) {
        return "Wallet is not connected.";
      } else if (!accountsReady) {
        return "You have not created the necessary token accounts.";
      } else if (!amount) {
        return "Enter an amount to transfer.";
      } else if (!amountGreaterThanZero) {
        return "Enter an amount greater than zero.";
      } else if (!sufficientFromTokens) {
        return "There are not sufficient funds in your wallet for this transfer.";
      } else if (!sufficientPoolBalance) {
        return "There are not sufficient funds in the pool for this transfer.";
      } else {
        return "";
      }
    },
    [
      fromMint,
      fromParse,
      fromTokenAccountBalance,
      fromTokenAccountExists,
      isReady,
      poolAddress,
      poolExists,
      toCustodyBalance,
      toMint,
      toTokenAccountExists,
    ]
  );

  const getMetadata = useCallback(
    (address: string) => {
      const tokenMapItem = solanaTokenMap.data?.find(
        (x) => x.address === address
      );
      const metaplexItem = metaplexData.data?.get(address);

      return {
        symbol: tokenMapItem?.symbol || metaplexItem?.data?.symbol || undefined,
        name: tokenMapItem?.name || metaplexItem?.data?.name || undefined,
        logo: tokenMapItem?.logoURI || metaplexItem?.data?.uri || undefined,
      };
    },
    [metaplexData.data, solanaTokenMap.data]
  );

  const isFetching = solanaTokenMap.isFetching || metaplexData.isFetching; //TODO add loading state on the actual Solana information
  const hasRequisiteData = !!(
    fromMintDecimals !== null &&
    fromMintDecimals !== undefined &&
    toTokenAccount &&
    fromTokenAccountBalance &&
    toCustodyAddress &&
    toCustodyBalance
  );

  const output: DataWrapper<SolanaMigratorInformation> = useMemo(() => {
    let data: SolanaMigratorInformation | null = null;
    if (hasRequisiteData) {
      data = {
        poolAddress,
        fromMint,
        toMint,
        fromMintDecimals,
        fromAssociatedTokenAccountExists: fromTokenAccountExists,
        toAssociatedTokenAccountExists: toTokenAccountExists,
        fromAssociatedTokenAccount: fromTokenAccount,
        toAssociatedTokenAccount: toTokenAccount,
        fromAssociatedTokenAccountBalance: fromTokenAccountBalance,
        toAssociatedTokenAccountBalance: toTokenAccountBalance || null,
        toCustodyAddress,
        toCustodyBalance,

        fromName: getMetadata(fromMint)?.name || null,
        fromSymbol: getMetadata(fromMint)?.symbol || null,
        fromLogo: getMetadata(fromMint)?.logo || null,
        toName: getMetadata(toMint)?.name || null,
        toSymbol: getMetadata(toMint)?.symbol || null,
        toLogo: getMetadata(toMint)?.logo || null,

        setToTokenAccountExists,

        getNotReadyCause: getNotReadyCause,

        migrateTokens,
      };
    }

    return {
      isFetching: isFetching,
      error: error || !hasRequisiteData,
      receivedAt: null,
      data,
    };
  }, [
    error,
    isFetching,
    hasRequisiteData,
    poolAddress,
    fromMint,
    toMint,
    fromMintDecimals,
    fromTokenAccountExists,
    toTokenAccountExists,
    fromTokenAccount,
    toTokenAccount,
    fromTokenAccountBalance,
    toTokenAccountBalance,
    toCustodyAddress,
    toCustodyBalance,
    getMetadata,
    getNotReadyCause,
    migrateTokens,
    setToTokenAccountExists,
  ]);

  return output;
}
