import {
  ChainId,
  CHAIN_ID_SOLANA,
  getForeignAssetSolana,
  hexToNativeString,
  hexToUint8Array,
} from "@certusone/wormhole-sdk";
import { Button, Typography } from "@material-ui/core";
import { Alert } from "@material-ui/lab";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSelector } from "react-redux";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import {
  selectTransferOriginAsset,
  selectTransferOriginChain,
  selectTransferTargetAddressHex,
} from "../store/selectors";
import { SOLANA_HOST, SOL_TOKEN_BRIDGE_ADDRESS } from "../utils/consts";
import parseError from "../utils/parseError";
import { signSendAndConfirm } from "../utils/solana";
import ButtonWithLoader from "./ButtonWithLoader";
import SmartAddress from "./SmartAddress";

export function useAssociatedAccountExistsState(
  targetChain: ChainId,
  mintAddress: string | null | undefined,
  readableTargetAddress: string | undefined
) {
  const [associatedAccountExists, setAssociatedAccountExists] = useState(true); // for now, assume it exists until we confirm it doesn't
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  useEffect(() => {
    setAssociatedAccountExists(true);
    if (
      targetChain !== CHAIN_ID_SOLANA ||
      !mintAddress ||
      !readableTargetAddress ||
      !solPK
    )
      return;
    let cancelled = false;
    (async () => {
      const connection = new Connection(SOLANA_HOST, "confirmed");
      const mintPublicKey = new PublicKey(mintAddress);
      const payerPublicKey = new PublicKey(solPK); // currently assumes the wallet is the owner
      const associatedAddress = await Token.getAssociatedTokenAddress(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        mintPublicKey,
        payerPublicKey
      );
      const match = associatedAddress.toString() === readableTargetAddress;
      if (match) {
        const associatedAddressInfo = await connection.getAccountInfo(
          associatedAddress
        );
        if (!associatedAddressInfo) {
          if (!cancelled) {
            setAssociatedAccountExists(false);
          }
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [targetChain, mintAddress, readableTargetAddress, solPK]);
  return useMemo(
    () => ({ associatedAccountExists, setAssociatedAccountExists }),
    [associatedAccountExists]
  );
}

export default function SolanaCreateAssociatedAddress({
  mintAddress,
  readableTargetAddress,
  associatedAccountExists,
  setAssociatedAccountExists,
}: {
  mintAddress: string;
  readableTargetAddress: string;
  associatedAccountExists: boolean;
  setAssociatedAccountExists: (associatedAccountExists: boolean) => void;
}) {
  const [isCreating, setIsCreating] = useState(false);
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const handleClick = useCallback(() => {
    if (
      associatedAccountExists ||
      !mintAddress ||
      !readableTargetAddress ||
      !solPK
    )
      return;
    (async () => {
      const connection = new Connection(SOLANA_HOST, "confirmed");
      const mintPublicKey = new PublicKey(mintAddress);
      const payerPublicKey = new PublicKey(solPK); // currently assumes the wallet is the owner
      const associatedAddress = await Token.getAssociatedTokenAddress(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        mintPublicKey,
        payerPublicKey
      );
      const match = associatedAddress.toString() === readableTargetAddress;
      if (match) {
        const associatedAddressInfo = await connection.getAccountInfo(
          associatedAddress
        );
        if (!associatedAddressInfo) {
          setIsCreating(true);
          const transaction = new Transaction().add(
            await Token.createAssociatedTokenAccountInstruction(
              ASSOCIATED_TOKEN_PROGRAM_ID,
              TOKEN_PROGRAM_ID,
              mintPublicKey,
              associatedAddress,
              payerPublicKey, // owner
              payerPublicKey // payer
            )
          );
          const { blockhash } = await connection.getRecentBlockhash();
          transaction.recentBlockhash = blockhash;
          transaction.feePayer = new PublicKey(payerPublicKey);
          await signSendAndConfirm(solanaWallet, connection, transaction);
          setIsCreating(false);
          setAssociatedAccountExists(true);
        } else {
          console.log("Account already exists.");
        }
      }
    })();
  }, [
    associatedAccountExists,
    setAssociatedAccountExists,
    mintAddress,
    solPK,
    readableTargetAddress,
    solanaWallet,
  ]);
  if (associatedAccountExists) return null;
  return (
    <>
      <Typography color="error" variant="body2">
        This associated token account doesn't exist.
      </Typography>
      <ButtonWithLoader
        disabled={
          !mintAddress || !readableTargetAddress || !solPK || isCreating
        }
        onClick={handleClick}
        showLoader={isCreating}
      >
        Create Associated Token Account
      </ButtonWithLoader>
    </>
  );
}

export function SolanaCreateAssociatedAddressAlternate() {
  const { enqueueSnackbar } = useSnackbar();
  const originChain = useSelector(selectTransferOriginChain);
  const originAsset = useSelector(selectTransferOriginAsset);
  const addressHex = useSelector(selectTransferTargetAddressHex);
  const base58TargetAddress = useMemo(
    () => hexToNativeString(addressHex, CHAIN_ID_SOLANA) || "",
    [addressHex]
  );
  const base58OriginAddress = useMemo(
    () => hexToNativeString(originAsset, CHAIN_ID_SOLANA) || "",
    [originAsset]
  );
  const connection = useMemo(() => new Connection(SOLANA_HOST), []);
  const [targetAsset, setTargetAsset] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    if (!(originChain && originAsset && addressHex && base58TargetAddress)) {
      setTargetAsset(null);
    } else if (originChain === CHAIN_ID_SOLANA && base58OriginAddress) {
      setTargetAsset(base58OriginAddress);
    } else {
      getForeignAssetSolana(
        connection,
        SOL_TOKEN_BRIDGE_ADDRESS,
        originChain,
        hexToUint8Array(originAsset)
      ).then((result) => {
        if (!cancelled) {
          setTargetAsset(result);
        }
      });
    }

    return () => {
      cancelled = true;
    };
  }, [
    originChain,
    originAsset,
    addressHex,
    base58TargetAddress,
    connection,
    base58OriginAddress,
  ]);

  const { associatedAccountExists, setAssociatedAccountExists } =
    useAssociatedAccountExistsState(
      CHAIN_ID_SOLANA,
      targetAsset,
      base58TargetAddress
    );

  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  const handleForceCreateClick = useCallback(() => {
    if (!targetAsset || !base58TargetAddress || !solPK) return;
    (async () => {
      const connection = new Connection(SOLANA_HOST, "confirmed");
      const mintPublicKey = new PublicKey(targetAsset);
      const payerPublicKey = new PublicKey(solPK); // currently assumes the wallet is the owner
      const associatedAddress = await Token.getAssociatedTokenAddress(
        ASSOCIATED_TOKEN_PROGRAM_ID,
        TOKEN_PROGRAM_ID,
        mintPublicKey,
        payerPublicKey
      );
      const match = associatedAddress.toString() === base58TargetAddress;
      if (match) {
        try {
          const transaction = new Transaction().add(
            await Token.createAssociatedTokenAccountInstruction(
              ASSOCIATED_TOKEN_PROGRAM_ID,
              TOKEN_PROGRAM_ID,
              mintPublicKey,
              associatedAddress,
              payerPublicKey, // owner
              payerPublicKey // payer
            )
          );
          const { blockhash } = await connection.getRecentBlockhash();
          transaction.recentBlockhash = blockhash;
          transaction.feePayer = new PublicKey(payerPublicKey);
          await signSendAndConfirm(solanaWallet, connection, transaction);
          setAssociatedAccountExists(true);
          enqueueSnackbar(null, {
            content: (
              <Alert severity="success">
                Successfully created associated token account
              </Alert>
            ),
          });
        } catch (e) {
          enqueueSnackbar(null, {
            content: <Alert severity="error">{parseError(e)}</Alert>,
          });
        }
      } else {
        enqueueSnackbar(null, {
          content: (
            <Alert severity="error">
              Derived address does not match the target address. Do you have the
              same wallet connected?
            </Alert>
          ),
        });
      }
    })();
  }, [
    setAssociatedAccountExists,
    targetAsset,
    solPK,
    base58TargetAddress,
    solanaWallet,
    enqueueSnackbar,
  ]);

  return targetAsset ? (
    <div style={{ textAlign: "center" }}>
      <Typography variant="subtitle2">Recipient Address:</Typography>
      <Typography component="div">
        <SmartAddress
          chainId={CHAIN_ID_SOLANA}
          address={base58TargetAddress}
          variant="h6"
          extraContent={
            <Button
              size="small"
              variant="outlined"
              onClick={handleForceCreateClick}
              disabled={!targetAsset || !base58TargetAddress || !solPK}
            >
              Force Create Account
            </Button>
          }
        />
      </Typography>
      {associatedAccountExists ? null : (
        <SolanaCreateAssociatedAddress
          mintAddress={targetAsset}
          readableTargetAddress={base58TargetAddress}
          associatedAccountExists={associatedAccountExists}
          setAssociatedAccountExists={setAssociatedAccountExists}
        />
      )}
    </div>
  ) : null;
}
