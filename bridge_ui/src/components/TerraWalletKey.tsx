import { useConnectedWallet, useWallet } from "@terra-money/wallet-provider";
import { useCallback, useState } from "react";
import TerraConnectWalletDialog from "./TerraConnectWalletDialog";
import ToggleConnectedButton from "./ToggleConnectedButton";

const TerraWalletKey = () => {
  const wallet = useWallet();
  const connectedWallet = useConnectedWallet();

  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const connect = useCallback(() => {
    setIsDialogOpen(true);
  }, [setIsDialogOpen]);

  const closeDialog = useCallback(() => {
    setIsDialogOpen(false);
  }, [setIsDialogOpen]);

  return (
    <>
      <ToggleConnectedButton
        connect={connect}
        disconnect={wallet.disconnect}
        connected={!!connectedWallet}
        pk={connectedWallet?.terraAddress || ""}
      />
      <TerraConnectWalletDialog isOpen={isDialogOpen} onClose={closeDialog} />
    </>
  );
};

export default TerraWalletKey;
