import { Typography } from "@material-ui/core";
import {
  ASSOCIATED_TOKEN_PROGRAM_ID,
  Token,
  TOKEN_PROGRAM_ID,
} from "@solana/spl-token";
import { Connection, PublicKey, Transaction } from "@solana/web3.js";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import { SOLANA_URL } from "../utils/consts";
import { signSendAndConfirm } from "../utils/solana";
import ButtonWithLoader from "./ButtonWithLoader";

export function useAssociatedAccountExistsState(
  mintAddress: string | null | undefined,
  readableTargetAddress: string | undefined
) {
  const [associatedAccountExists, setAssociatedAccountExists] = useState(true); // for now, assume it exists until we confirm it doesn't
  const solanaWallet = useSolanaWallet();
  const solPK = solanaWallet?.publicKey;
  useEffect(() => {
    setAssociatedAccountExists(true);
    if (!mintAddress || !readableTargetAddress || !solPK) return;
    let cancelled = false;
    (async () => {
      const connection = new Connection(SOLANA_URL, "confirmed");
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
  }, [mintAddress, readableTargetAddress, solPK]);
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
  mintAddress: string | undefined;
  readableTargetAddress: string | undefined;
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
      try {
        const connection = new Connection(SOLANA_URL, "confirmed");
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
          }
        }
      } catch (e) {
        console.log("cannot create specified spl token account");
        console.error(e);
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
