import { useCallback, useMemo, useState } from "react";
import { useSolanaWallet } from "../contexts/SolanaWalletContext";
import SolanaConnectWalletDialog from "./SolanaConnectWalletDialog";
import ToggleConnectedButton from "./ToggleConnectedButton";

const SolanaWalletKey = () => {
  const { publicKey, wallet, disconnect } = useSolanaWallet();
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const openDialog = useCallback(() => {
    setIsDialogOpen(true);
  }, [setIsDialogOpen]);

  const closeDialog = useCallback(() => {
    setIsDialogOpen(false);
  }, [setIsDialogOpen]);

  const publicKeyBase58 = useMemo(() => {
    return publicKey?.toBase58() || "";
  }, [publicKey]);

  return (
    <>
      <ToggleConnectedButton
        connect={openDialog}
        disconnect={disconnect}
        connected={!!wallet?.adapter.connected}
        pk={publicKeyBase58}
        walletIcon={wallet?.adapter.icon}
      />
      <SolanaConnectWalletDialog isOpen={isDialogOpen} onClose={closeDialog} />
    </>
  );
};

export default SolanaWalletKey;
