import { useConnectedWallet, useWallet } from "@terra-money/wallet-provider";
import { useState } from "react";
import TerraConnectWalletDialog from "./TerraConnectWalletDialog";
import ToggleConnectedButton from "./ToggleConnectedButton";

const TerraWalletKey = () => {
  const wallet = useWallet();
  const connectedWallet = useConnectedWallet();

  const [isDialogOpen, setIsDialogOpen] = useState(false);

  return (
    <>
      <ToggleConnectedButton
        connect={() => setIsDialogOpen(true)}
        disconnect={() => wallet.disconnect()}
        connected={!!connectedWallet}
        pk={connectedWallet?.terraAddress || ""}
      />
      <TerraConnectWalletDialog
        isOpen={isDialogOpen}
        onClose={() => setIsDialogOpen(false)}
      />
    </>
  );
};

export default TerraWalletKey;
