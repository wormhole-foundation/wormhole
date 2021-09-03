import { CHAIN_ID_ETH, CHAIN_ID_SOLANA } from "@certusone/wormhole-sdk";
import { LinearProgress, makeStyles, Typography } from "@material-ui/core";
import { Connection } from "@solana/web3.js";
import { useEffect, useState } from "react";
import { useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import {
  selectTransferIsSendComplete,
  selectTransferSourceChain,
  selectTransferTransferTx,
} from "../../store/selectors";
import { CHAINS_BY_ID, SOLANA_HOST } from "../../utils/consts";

const useStyles = makeStyles((theme) => ({
  root: {
    marginTop: theme.spacing(2),
    textAlign: "center",
  },
  message: {
    marginTop: theme.spacing(1),
  },
}));

export default function TransferProgress() {
  const classes = useStyles();
  const sourceChain = useSelector(selectTransferSourceChain);
  const transferTx = useSelector(selectTransferTransferTx);
  const isSendComplete = useSelector(selectTransferIsSendComplete);
  const { provider } = useEthereumProvider();
  const [currentBlock, setCurrentBlock] = useState(0);
  useEffect(() => {
    if (isSendComplete || !transferTx) return;
    let cancelled = false;
    if (sourceChain === CHAIN_ID_ETH && provider) {
      (async () => {
        while (!cancelled) {
          await new Promise((resolve) => setTimeout(resolve, 500));
          try {
            const newBlock = await provider.getBlockNumber();
            if (!cancelled) {
              setCurrentBlock(newBlock);
            }
          } catch (e) {
            console.error(e);
          }
        }
      })();
    }
    if (sourceChain === CHAIN_ID_SOLANA) {
      (async () => {
        const connection = new Connection(SOLANA_HOST, "confirmed");
        while (!cancelled) {
          await new Promise((resolve) => setTimeout(resolve, 200));
          try {
            const newBlock = await connection.getSlot();
            if (!cancelled) {
              setCurrentBlock(newBlock);
            }
          } catch (e) {
            console.error(e);
          }
        }
      })();
    }
    return () => {
      cancelled = true;
    };
  }, [isSendComplete, sourceChain, provider, transferTx]);
  const blockDiff =
    transferTx && transferTx.block && currentBlock
      ? currentBlock - transferTx.block
      : undefined;
  const expectedBlocks =
    sourceChain === CHAIN_ID_SOLANA
      ? 32
      : sourceChain === CHAIN_ID_ETH
      ? 15
      : 1;
  if (
    !isSendComplete &&
    (sourceChain === CHAIN_ID_SOLANA || sourceChain === CHAIN_ID_ETH) &&
    blockDiff !== undefined
  ) {
    return (
      <div className={classes.root}>
        <LinearProgress
          value={
            blockDiff < expectedBlocks ? (blockDiff / expectedBlocks) * 75 : 75
          }
          variant="determinate"
        />
        <Typography variant="body2" className={classes.message}>
          {blockDiff < expectedBlocks
            ? `Waiting for ${blockDiff} / ${expectedBlocks} confirmations on ${CHAINS_BY_ID[sourceChain].name}...`
            : `Waiting for Wormhole Network consensus...`}
        </Typography>
      </div>
    );
  }
  return null;
}
