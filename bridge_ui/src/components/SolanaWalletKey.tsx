import { Toolbar, Typography } from "@material-ui/core";
import DisconnectIcon from "@material-ui/icons/LinkOff";
import {
  WalletDisconnectButton,
  WalletMultiButton,
} from "@solana/wallet-adapter-material-ui";
import { useWallet } from "@solana/wallet-adapter-react";
import React, { FC } from "react";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";

const SolanaWalletKey = () => {
  const wallet = useSolanaWallet();
  return (
    <Toolbar style={{ display: "flex" }}>
      <WalletMultiButton />
      {wallet && (
        <WalletDisconnectButton
          startIcon={<DisconnectIcon />}
          style={{ marginLeft: 8 }}
        />
      )}
    </Toolbar>
  );
};

export default SolanaWalletKey;
