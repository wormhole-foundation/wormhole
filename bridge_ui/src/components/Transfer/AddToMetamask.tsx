import { CHAIN_ID_ETH } from "@certusone/wormhole-sdk";
import { Button, makeStyles } from "@material-ui/core";
import detectEthereumProvider from "@metamask/detect-provider";
import { useCallback } from "react";
import { useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import {
  selectTransferTargetAsset,
  selectTransferTargetChain,
} from "../../store/selectors";
import {
  ethTokenToParsedTokenAccount,
  getEthereumToken,
} from "../../utils/ethereum";

const useStyles = makeStyles((theme) => ({
  addButton: {
    display: "block",
    margin: `${theme.spacing(1)}px auto 0px`,
  },
}));

export default function AddToMetamask() {
  const classes = useStyles();
  const targetChain = useSelector(selectTransferTargetChain);
  const targetAsset = useSelector(selectTransferTargetAsset);
  const { provider, signerAddress } = useEthereumProvider();
  const handleClick = useCallback(() => {
    if (provider && targetAsset && signerAddress) {
      (async () => {
        try {
          const token = await getEthereumToken(targetAsset, provider);
          const { symbol, decimals } = await ethTokenToParsedTokenAccount(
            token,
            signerAddress
          );
          const ethereum = (await detectEthereumProvider()) as any;
          ethereum.request({
            method: "wallet_watchAsset",
            params: {
              type: "ERC20", // In the future, other standards will be supported
              options: {
                address: targetAsset, // The address of the token contract
                symbol, // A ticker symbol or shorthand, up to 5 characters
                decimals, // The number of token decimals
                // image: string; // A string url of the token logo
              },
            },
          });
        } catch (e) {
          console.error(e);
        }
      })();
    }
  }, [provider, targetAsset, signerAddress]);
  return provider &&
    signerAddress &&
    targetAsset &&
    targetChain === CHAIN_ID_ETH ? (
    <Button
      onClick={handleClick}
      size="small"
      variant="outlined"
      className={classes.addButton}
    >
      Add to Metamask
    </Button>
  ) : null;
}
