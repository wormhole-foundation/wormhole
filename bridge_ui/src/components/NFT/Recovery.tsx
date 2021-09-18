import {
  ChainId,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_SOLANA,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
} from "@certusone/wormhole-sdk";
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Fab,
  makeStyles,
  MenuItem,
  TextField,
  Tooltip,
  Typography,
} from "@material-ui/core";
import { Restore } from "@material-ui/icons";
import { Alert } from "@material-ui/lab";
import { Connection } from "@solana/web3.js";
import { BigNumber, ethers } from "ethers";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import { setSignedVAAHex, setStep, setTargetChain } from "../../store/nftSlice";
import {
  selectNFTSignedVAAHex,
  selectNFTSourceChain,
} from "../../store/selectors";
import {
  hexToNativeString,
  hexToUint8Array,
  uint8ArrayToHex,
} from "../../utils/array";
import {
  CHAINS,
  ETH_BRIDGE_ADDRESS,
  ETH_NFT_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_NFT_BRIDGE_ADDRESS,
  WORMHOLE_RPC_HOSTS,
} from "../../utils/consts";
import { getSignedVAAWithRetry } from "../../utils/getSignedVAAWithRetry";
import { METADATA_REPLACE } from "../../utils/metaplex";
import KeyAndBalance from "../KeyAndBalance";

const useStyles = makeStyles((theme) => ({
  fab: {
    position: "fixed",
    bottom: theme.spacing(2),
    right: theme.spacing(2),
  },
}));

async function eth(provider: ethers.providers.Web3Provider, tx: string) {
  try {
    const receipt = await provider.getTransactionReceipt(tx);
    const sequence = parseSequenceFromLogEth(receipt, ETH_BRIDGE_ADDRESS);
    const emitterAddress = getEmitterAddressEth(ETH_NFT_BRIDGE_ADDRESS);
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_ETH,
      emitterAddress,
      sequence.toString(),
      WORMHOLE_RPC_HOSTS.length
    );
    return uint8ArrayToHex(vaaBytes);
  } catch (e) {
    console.error(e);
  }
  return "";
}

async function solana(tx: string) {
  try {
    const connection = new Connection(SOLANA_HOST, "confirmed");
    const info = await connection.getTransaction(tx);
    if (!info) {
      throw new Error("An error occurred while fetching the transaction info");
    }
    const sequence = parseSequenceFromLogSolana(info);
    const emitterAddress = await getEmitterAddressSolana(
      SOL_NFT_BRIDGE_ADDRESS
    );
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_SOLANA,
      emitterAddress,
      sequence.toString(),
      WORMHOLE_RPC_HOSTS.length
    );
    return uint8ArrayToHex(vaaBytes);
  } catch (e) {
    console.error(e);
  }
  return "";
}

// note: actual first byte is message type
//     0   [u8; 32] token_address
//     32  u16      token_chain
//     34  [u8; 32] symbol
//     66  [u8; 32] name
//     98  u256     tokenId
//     130 u8       uri_len
//     131 [u8;len] uri
//     ?   [u8; 32] recipient
//     ?   u16      recipient_chain

// TODO: move to wasm / sdk, share with solana
const parsePayload = (arr: Buffer) => {
  const originAddress = arr.slice(1, 1 + 32).toString("hex");
  const originChain = arr.readUInt16BE(33) as ChainId;
  const symbol = Buffer.from(arr.slice(35, 35 + 32))
    .toString("utf8")
    .replace(METADATA_REPLACE, "");
  const name = Buffer.from(arr.slice(67, 67 + 32))
    .toString("utf8")
    .replace(METADATA_REPLACE, "");
  const tokenId = BigNumber.from(arr.slice(99, 99 + 32));
  const uri_len = arr.readUInt8(131);
  const uri = Buffer.from(arr.slice(132, 132 + uri_len))
    .toString("utf8")
    .replace(METADATA_REPLACE, "");
  const target_offset = 132 + uri_len;
  const targetAddress = arr
    .slice(target_offset, target_offset + 32)
    .toString("hex");
  const targetChain = arr.readUInt16BE(target_offset + 32) as ChainId;
  return {
    originAddress,
    originChain,
    symbol,
    name,
    tokenId,
    uri,
    targetAddress,
    targetChain,
  };
};

function RecoveryDialogContent({
  onClose,
  disabled,
}: {
  onClose: () => void;
  disabled: boolean;
}) {
  const dispatch = useDispatch();
  const { provider } = useEthereumProvider();
  const currentSourceChain = useSelector(selectNFTSourceChain);
  const [recoverySourceChain, setRecoverySourceChain] =
    useState(currentSourceChain);
  const [recoverySourceTx, setRecoverySourceTx] = useState("");
  const currentSignedVAA = useSelector(selectNFTSignedVAAHex);
  const [recoverySignedVAA, setRecoverySignedVAA] = useState(currentSignedVAA);
  const [recoveryParsedVAA, setRecoveryParsedVAA] = useState<any>(null);
  useEffect(() => {
    if (!recoverySignedVAA) {
      setRecoverySourceTx("");
      setRecoverySourceChain(currentSourceChain);
    }
  }, [recoverySignedVAA, currentSourceChain]);
  useEffect(() => {
    if (recoverySourceTx) {
      let cancelled = false;
      if (recoverySourceChain === CHAIN_ID_ETH && provider) {
        (async () => {
          const vaa = await eth(provider, recoverySourceTx);
          if (!cancelled) {
            setRecoverySignedVAA(vaa);
          }
        })();
      } else if (recoverySourceChain === CHAIN_ID_SOLANA) {
        (async () => {
          const vaa = await solana(recoverySourceTx);
          if (!cancelled) {
            setRecoverySignedVAA(vaa);
          }
        })();
      }
      return () => {
        cancelled = true;
      };
    }
  }, [recoverySourceChain, recoverySourceTx, provider]);
  useEffect(() => {
    setRecoverySignedVAA(currentSignedVAA);
  }, [currentSignedVAA]);
  const handleSourceChainChange = useCallback((event) => {
    setRecoverySourceTx("");
    setRecoverySourceChain(event.target.value);
  }, []);
  const handleSourceTxChange = useCallback((event) => {
    setRecoverySourceTx(event.target.value.trim());
  }, []);
  const handleSignedVAAChange = useCallback((event) => {
    setRecoverySignedVAA(event.target.value.trim());
  }, []);
  useEffect(() => {
    let cancelled = false;
    if (recoverySignedVAA) {
      (async () => {
        try {
          const { parse_vaa } = await import(
            "@certusone/wormhole-sdk/lib/solana/core/bridge"
          );
          const parsedVAA = parse_vaa(hexToUint8Array(recoverySignedVAA));
          if (!cancelled) {
            setRecoveryParsedVAA(parsedVAA);
          }
        } catch (e) {
          console.log(e);
          if (!cancelled) {
            setRecoveryParsedVAA(null);
          }
        }
      })();
    }
    return () => {
      cancelled = true;
    };
  }, [recoverySignedVAA]);
  const parsedPayload = useMemo(
    () =>
      recoveryParsedVAA?.payload
        ? parsePayload(Buffer.from(new Uint8Array(recoveryParsedVAA.payload)))
        : null,
    [recoveryParsedVAA]
  );
  const parsedPayloadTargetChain = parsedPayload?.targetChain;
  const enableRecovery = recoverySignedVAA && parsedPayloadTargetChain;
  const handleRecoverClick = useCallback(() => {
    if (enableRecovery && recoverySignedVAA && parsedPayloadTargetChain) {
      // TODO: make recovery reducer
      dispatch(setSignedVAAHex(recoverySignedVAA));
      dispatch(setTargetChain(parsedPayloadTargetChain));
      dispatch(setStep(3));
      onClose();
    }
  }, [
    dispatch,
    enableRecovery,
    recoverySignedVAA,
    parsedPayloadTargetChain,
    onClose,
  ]);
  return (
    <>
      <DialogContent>
        <Alert severity="info">
          If you have sent your tokens but have not redeemed them, you may paste
          in the Source Transaction ID (from Step 3) to resume your transfer.
        </Alert>
        <TextField
          select
          label="Source Chain"
          disabled={!!recoverySignedVAA}
          value={recoverySourceChain}
          onChange={handleSourceChainChange}
          fullWidth
          margin="normal"
        >
          {CHAINS.filter(
            ({ id }) => id === CHAIN_ID_ETH || id === CHAIN_ID_SOLANA
          ).map(({ id, name }) => (
            <MenuItem key={id} value={id}>
              {name}
            </MenuItem>
          ))}
        </TextField>
        {recoverySourceChain === CHAIN_ID_ETH ||
        recoverySourceChain === CHAIN_ID_BSC ? (
          <KeyAndBalance chainId={recoverySourceChain} />
        ) : null}
        <TextField
          label="Source Tx"
          disabled={!!recoverySignedVAA}
          value={recoverySourceTx}
          onChange={handleSourceTxChange}
          fullWidth
          margin="normal"
        />
        <Box mt={4}>
          <Typography>or</Typography>
        </Box>
        <TextField
          label="Signed VAA (Hex)"
          value={recoverySignedVAA || ""}
          onChange={handleSignedVAAChange}
          fullWidth
          margin="normal"
        />
        <Box my={4}>
          <Divider />
        </Box>
        <TextField
          label="Emitter Chain"
          disabled
          value={recoveryParsedVAA?.emitter_chain || ""}
          fullWidth
          margin="normal"
        />
        <TextField
          label="Emitter Address"
          disabled
          value={
            (recoveryParsedVAA &&
              hexToNativeString(
                recoveryParsedVAA.emitter_address,
                recoveryParsedVAA.emitter_chain
              )) ||
            ""
          }
          fullWidth
          margin="normal"
        />
        <TextField
          label="Sequence"
          disabled
          value={recoveryParsedVAA?.sequence || ""}
          fullWidth
          margin="normal"
        />
        <TextField
          label="Timestamp"
          disabled
          value={
            (recoveryParsedVAA &&
              new Date(recoveryParsedVAA.timestamp * 1000).toLocaleString()) ||
            ""
          }
          fullWidth
          margin="normal"
        />
        <Box my={4}>
          <Divider />
        </Box>
        <TextField
          label="Origin Chain"
          disabled
          value={parsedPayload?.originChain.toString() || ""}
          fullWidth
          margin="normal"
        />
        <TextField
          label="Origin Token Address"
          disabled
          value={
            (parsedPayload &&
              hexToNativeString(
                parsedPayload.originAddress,
                parsedPayload.originChain
              )) ||
            ""
          }
          fullWidth
          margin="normal"
        />
        <TextField
          label="Origin Token ID"
          disabled
          value={parsedPayload?.tokenId || ""}
          fullWidth
          margin="normal"
        />
        <TextField
          label="Target Chain"
          disabled
          value={parsedPayload?.targetChain.toString() || ""}
          fullWidth
          margin="normal"
        />
        <TextField
          label="Target Address"
          disabled
          value={
            (parsedPayload &&
              hexToNativeString(
                parsedPayload.targetAddress,
                parsedPayload.targetChain
              )) ||
            ""
          }
          fullWidth
          margin="normal"
        />
        <Box my={4}>
          <Divider />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} variant="outlined" color="default">
          Cancel
        </Button>
        <Button
          onClick={handleRecoverClick}
          variant="contained"
          color="primary"
          disabled={!enableRecovery || disabled}
        >
          Recover
        </Button>
      </DialogActions>
    </>
  );
}

export default function Recovery({
  open,
  setOpen,
  disabled,
}: {
  open: boolean;
  setOpen: (open: boolean) => void;
  disabled: boolean;
}) {
  const classes = useStyles();
  const handleOpenClick = useCallback(() => {
    setOpen(true);
  }, [setOpen]);
  const handleCloseClick = useCallback(() => {
    setOpen(false);
  }, [setOpen]);
  return (
    <>
      <Tooltip title="Open Recovery Dialog">
        <Fab className={classes.fab} onClick={handleOpenClick}>
          <Restore />
        </Fab>
      </Tooltip>
      <Dialog open={open} onClose={handleCloseClick} maxWidth="md" fullWidth>
        <DialogTitle>Recovery</DialogTitle>
        <RecoveryDialogContent onClose={handleCloseClick} disabled={disabled} />
      </Dialog>
    </>
  );
}
