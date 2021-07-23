import { Typography } from "@material-ui/core";
import { useEffect, useState } from "react";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";

const EthereumSignerKey = () => {
  const provider = useEthereumProvider();
  const [pk, setPk] = useState("");
  // TODO: should this be moved to the context?
  useEffect(() => {
    let mounted = true;
    provider
      ?.getSigner()
      .getAddress()
      .then((pk) => {
        if (mounted) {
          setPk(pk);
        }
      })
      .catch(() => {
        console.error("Failed to get signer address");
      });
    return () => {
      mounted = false;
    };
  }, [provider]);
  if (!pk) return null;
  return (
    <Typography>
      {pk.substring(0, 6)}...{pk.substr(pk.length - 4)}
    </Typography>
  );
};

export default EthereumSignerKey;
