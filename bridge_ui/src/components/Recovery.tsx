import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  getEmitterAddressEth,
  getEmitterAddressSolana,
  getEmitterAddressTerra,
  hexToNativeString,
  hexToUint8Array,
  isEVMChain,
  parseNFTPayload,
  parseSequenceFromLogEth,
  parseSequenceFromLogSolana,
  parseSequenceFromLogTerra,
  parseTransferPayload,
  uint8ArrayToHex,
} from "@certusone/wormhole-sdk";
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Card,
  CircularProgress,
  Container,
  Divider,
  makeStyles,
  MenuItem,
  TextField,
} from "@material-ui/core";
import { ExpandMore } from "@material-ui/icons";
import { Alert } from "@material-ui/lab";
import { Connection } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { ethers } from "ethers";
import { useSnackbar } from "notistack";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useDispatch } from "react-redux";
import { useHistory, useLocation } from "react-router";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import useIsWalletReady from "../hooks/useIsWalletReady";
import { COLORS } from "../muiTheme";
import { setRecoveryVaa as setRecoveryNFTVaa } from "../store/nftSlice";
import { setRecoveryVaa } from "../store/transferSlice";
import {
  CHAINS,
  CHAINS_BY_ID,
  CHAINS_WITH_NFT_SUPPORT,
  getBridgeAddressForChain,
  getNFTBridgeAddressForChain,
  getTokenBridgeAddressForChain,
  SOLANA_HOST,
  SOL_NFT_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_HOST,
  TERRA_TOKEN_BRIDGE_ADDRESS,
  WORMHOLE_RPC_HOSTS,
} from "../utils/consts";
import { getSignedVAAWithRetry } from "../utils/getSignedVAAWithRetry";
import parseError from "../utils/parseError";
import ButtonWithLoader from "./ButtonWithLoader";
import ChainSelect from "./ChainSelect";
import KeyAndBalance from "./KeyAndBalance";

const useStyles = makeStyles((theme) => ({
  mainCard: {
    padding: theme.spacing(2),
    backgroundColor: COLORS.nearBlackWithMinorTransparency,
  },
  advancedContainer: {
    padding: theme.spacing(2, 0),
  },
}));

async function evm(
  provider: ethers.providers.Web3Provider,
  tx: string,
  enqueueSnackbar: any,
  chainId: ChainId,
  nft: boolean
) {
  try {
    const receipt = await provider.getTransactionReceipt(tx);
    const sequence = parseSequenceFromLogEth(
      receipt,
      getBridgeAddressForChain(chainId)
    );
    const emitterAddress = getEmitterAddressEth(
      nft
        ? getNFTBridgeAddressForChain(chainId)
        : getTokenBridgeAddressForChain(chainId)
    );
    const { vaaBytes } = await getSignedVAAWithRetry(
      chainId,
      emitterAddress,
      sequence.toString(),
      WORMHOLE_RPC_HOSTS.length
    );
    return { vaa: uint8ArrayToHex(vaaBytes), error: null };
  } catch (e) {
    console.error(e);
    enqueueSnackbar(null, {
      content: <Alert severity="error">{parseError(e)}</Alert>,
    });
    return { vaa: null, error: parseError(e) };
  }
}

async function solana(tx: string, enqueueSnackbar: any, nft: boolean) {
  try {
    const connection = new Connection(SOLANA_HOST, "confirmed");
    const info = await connection.getTransaction(tx);
    if (!info) {
      throw new Error("An error occurred while fetching the transaction info");
    }
    const sequence = parseSequenceFromLogSolana(info);
    const emitterAddress = await getEmitterAddressSolana(
      nft ? SOL_NFT_BRIDGE_ADDRESS : SOL_TOKEN_BRIDGE_ADDRESS
    );
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_SOLANA,
      emitterAddress,
      sequence.toString(),
      WORMHOLE_RPC_HOSTS.length
    );
    return { vaa: uint8ArrayToHex(vaaBytes), error: null };
  } catch (e) {
    console.error(e);
    enqueueSnackbar(null, {
      content: <Alert severity="error">{parseError(e)}</Alert>,
    });
    return { vaa: null, error: parseError(e) };
  }
}

async function terra(tx: string, enqueueSnackbar: any) {
  try {
    const lcd = new LCDClient(TERRA_HOST);
    const info = await lcd.tx.txInfo(tx);
    const sequence = parseSequenceFromLogTerra(info);
    if (!sequence) {
      throw new Error("Sequence not found");
    }
    const emitterAddress = await getEmitterAddressTerra(
      TERRA_TOKEN_BRIDGE_ADDRESS
    );
    const { vaaBytes } = await getSignedVAAWithRetry(
      CHAIN_ID_TERRA,
      emitterAddress,
      sequence,
      WORMHOLE_RPC_HOSTS.length
    );
    return { vaa: uint8ArrayToHex(vaaBytes), error: null };
  } catch (e) {
    console.error(e);
    enqueueSnackbar(null, {
      content: <Alert severity="error">{parseError(e)}</Alert>,
    });
    return { vaa: null, error: parseError(e) };
  }
}

export default function Recovery() {
  const classes = useStyles();
  const { push } = useHistory();
  const { enqueueSnackbar } = useSnackbar();
  const dispatch = useDispatch();
  const { provider } = useEthereumProvider();
  const [type, setType] = useState("Token");
  const isNFT = type === "NFT";
  const [recoverySourceChain, setRecoverySourceChain] =
    useState(CHAIN_ID_SOLANA);
  const [recoverySourceTx, setRecoverySourceTx] = useState("");
  const [recoverySourceTxIsLoading, setRecoverySourceTxIsLoading] =
    useState(false);
  const [recoverySourceTxError, setRecoverySourceTxError] = useState("");
  const [recoverySignedVAA, setRecoverySignedVAA] = useState("");
  const [recoveryParsedVAA, setRecoveryParsedVAA] = useState<any>(null);
  const { isReady, statusMessage } = useIsWalletReady(recoverySourceChain);
  const walletConnectError =
    isEVMChain(recoverySourceChain) && !isReady ? statusMessage : "";
  const parsedPayload = useMemo(() => {
    try {
      return recoveryParsedVAA?.payload
        ? isNFT
          ? parseNFTPayload(
              Buffer.from(new Uint8Array(recoveryParsedVAA.payload))
            )
          : parseTransferPayload(
              Buffer.from(new Uint8Array(recoveryParsedVAA.payload))
            )
        : null;
    } catch (e) {
      console.error(e);
      return null;
    }
  }, [recoveryParsedVAA, isNFT]);

  const { search } = useLocation();
  const query = useMemo(() => new URLSearchParams(search), [search]);
  const pathSourceChain = query.get("sourceChain");
  const pathSourceTransaction = query.get("transactionId");

  //This effect initializes the state based on the path params.
  useEffect(() => {
    if (!pathSourceChain && !pathSourceTransaction) {
      return;
    }
    try {
      const sourceChain: ChainId =
        CHAINS_BY_ID[parseFloat(pathSourceChain || "") as ChainId]?.id;

      if (sourceChain) {
        setRecoverySourceChain(sourceChain);
      }
      if (pathSourceTransaction) {
        setRecoverySourceTx(pathSourceTransaction);
      }
    } catch (e) {
      console.error(e);
      console.error("Invalid path params specified.");
    }
  }, [pathSourceChain, pathSourceTransaction]);

  useEffect(() => {
    if (recoverySourceTx && (!isEVMChain(recoverySourceChain) || isReady)) {
      let cancelled = false;
      if (isEVMChain(recoverySourceChain) && provider) {
        setRecoverySourceTxError("");
        setRecoverySourceTxIsLoading(true);
        (async () => {
          const { vaa, error } = await evm(
            provider,
            recoverySourceTx,
            enqueueSnackbar,
            recoverySourceChain,
            isNFT
          );
          if (!cancelled) {
            setRecoverySourceTxIsLoading(false);
            if (vaa) {
              setRecoverySignedVAA(vaa);
            }
            if (error) {
              setRecoverySourceTxError(error);
            }
          }
        })();
      } else if (recoverySourceChain === CHAIN_ID_SOLANA) {
        setRecoverySourceTxError("");
        setRecoverySourceTxIsLoading(true);
        (async () => {
          const { vaa, error } = await solana(
            recoverySourceTx,
            enqueueSnackbar,
            isNFT
          );
          if (!cancelled) {
            setRecoverySourceTxIsLoading(false);
            if (vaa) {
              setRecoverySignedVAA(vaa);
            }
            if (error) {
              setRecoverySourceTxError(error);
            }
          }
        })();
      } else if (recoverySourceChain === CHAIN_ID_TERRA) {
        setRecoverySourceTxError("");
        setRecoverySourceTxIsLoading(true);
        (async () => {
          const { vaa, error } = await terra(recoverySourceTx, enqueueSnackbar);
          if (!cancelled) {
            setRecoverySourceTxIsLoading(false);
            if (vaa) {
              setRecoverySignedVAA(vaa);
            }
            if (error) {
              setRecoverySourceTxError(error);
            }
          }
        })();
      }
      return () => {
        cancelled = true;
      };
    }
  }, [
    recoverySourceChain,
    recoverySourceTx,
    provider,
    enqueueSnackbar,
    isNFT,
    isReady,
  ]);
  const handleTypeChange = useCallback((event) => {
    setRecoverySourceChain((prevChain) =>
      event.target.value === "NFT" &&
      !CHAINS_WITH_NFT_SUPPORT.find((chain) => chain.id === prevChain)
        ? CHAIN_ID_SOLANA
        : prevChain
    );
    setType(event.target.value);
  }, []);
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
            "@certusone/wormhole-sdk/lib/esm/solana/core/bridge"
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
  const parsedPayloadTargetChain = parsedPayload?.targetChain;
  const enableRecovery = recoverySignedVAA && parsedPayloadTargetChain;
  const handleRecoverClick = useCallback(() => {
    if (enableRecovery && recoverySignedVAA && parsedPayloadTargetChain) {
      // TODO: make recovery reducer
      if (isNFT) {
        dispatch(
          setRecoveryNFTVaa({
            vaa: recoverySignedVAA,
            parsedPayload: {
              targetChain: parsedPayload.targetChain,
              targetAddress: parsedPayload.targetAddress,
              originChain: parsedPayload.originChain,
              originAddress: parsedPayload.originAddress,
            },
          })
        );
        push("/nft");
      } else {
        dispatch(
          setRecoveryVaa({
            vaa: recoverySignedVAA,
            parsedPayload: {
              targetChain: parsedPayload.targetChain,
              targetAddress: parsedPayload.targetAddress,
              originChain: parsedPayload.originChain,
              originAddress: parsedPayload.originAddress,
              amount:
                "amount" in parsedPayload
                  ? parsedPayload.amount.toString()
                  : "",
            },
          })
        );
        push("/transfer");
      }
    }
  }, [
    dispatch,
    enableRecovery,
    recoverySignedVAA,
    parsedPayloadTargetChain,
    parsedPayload,
    isNFT,
    push,
  ]);
  return (
    <Container maxWidth="md">
      <Card className={classes.mainCard}>
        <Alert severity="info" variant="outlined">
          If you have sent your tokens but have not redeemed them, you may paste
          in the Source Transaction ID (from Step 3) to resume your transfer.
        </Alert>
        <TextField
          select
          variant="outlined"
          label="Type"
          disabled={!!recoverySignedVAA}
          value={type}
          onChange={handleTypeChange}
          fullWidth
          margin="normal"
        >
          <MenuItem value="Token">Token</MenuItem>
          <MenuItem value="NFT">NFT</MenuItem>
        </TextField>
        <ChainSelect
          select
          variant="outlined"
          label="Source Chain"
          disabled={!!recoverySignedVAA}
          value={recoverySourceChain}
          onChange={handleSourceChainChange}
          fullWidth
          margin="normal"
          chains={isNFT ? CHAINS_WITH_NFT_SUPPORT : CHAINS}
        />
        {isEVMChain(recoverySourceChain) ? (
          <KeyAndBalance chainId={recoverySourceChain} />
        ) : null}
        <TextField
          variant="outlined"
          label="Source Tx (paste here)"
          disabled={
            !!recoverySignedVAA ||
            recoverySourceTxIsLoading ||
            !!walletConnectError
          }
          value={recoverySourceTx}
          onChange={handleSourceTxChange}
          error={!!recoverySourceTxError || !!walletConnectError}
          helperText={recoverySourceTxError || walletConnectError}
          fullWidth
          margin="normal"
        />
        <ButtonWithLoader
          onClick={handleRecoverClick}
          disabled={!enableRecovery}
          showLoader={recoverySourceTxIsLoading}
        >
          Recover
        </ButtonWithLoader>
        <div className={classes.advancedContainer}>
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMore />}>
              Advanced
            </AccordionSummary>
            <AccordionDetails>
              <div>
                <Box position="relative">
                  <TextField
                    variant="outlined"
                    label="Signed VAA (Hex)"
                    disabled={recoverySourceTxIsLoading}
                    value={recoverySignedVAA || ""}
                    onChange={handleSignedVAAChange}
                    fullWidth
                    margin="normal"
                  />
                  {recoverySourceTxIsLoading ? (
                    <Box
                      position="absolute"
                      style={{
                        top: 0,
                        right: 0,
                        left: 0,
                        bottom: 0,
                        backgroundColor: "rgba(0,0,0,0.5)",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                      }}
                    >
                      <CircularProgress />
                    </Box>
                  ) : null}
                </Box>
                <Box my={4}>
                  <Divider />
                </Box>
                <TextField
                  variant="outlined"
                  label="Emitter Chain"
                  disabled
                  value={recoveryParsedVAA?.emitter_chain || ""}
                  fullWidth
                  margin="normal"
                />
                <TextField
                  variant="outlined"
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
                  variant="outlined"
                  label="Sequence"
                  disabled
                  value={recoveryParsedVAA?.sequence || ""}
                  fullWidth
                  margin="normal"
                />
                <TextField
                  variant="outlined"
                  label="Timestamp"
                  disabled
                  value={
                    (recoveryParsedVAA &&
                      new Date(
                        recoveryParsedVAA.timestamp * 1000
                      ).toLocaleString()) ||
                    ""
                  }
                  fullWidth
                  margin="normal"
                />
                <Box my={4}>
                  <Divider />
                </Box>
                <TextField
                  variant="outlined"
                  label="Origin Chain"
                  disabled
                  value={parsedPayload?.originChain.toString() || ""}
                  fullWidth
                  margin="normal"
                />
                <TextField
                  variant="outlined"
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
                {isNFT ? (
                  <TextField
                    variant="outlined"
                    label="Origin Token ID"
                    disabled
                    // @ts-ignore
                    value={parsedPayload?.tokenId || ""}
                    fullWidth
                    margin="normal"
                  />
                ) : null}
                <TextField
                  variant="outlined"
                  label="Target Chain"
                  disabled
                  value={parsedPayload?.targetChain.toString() || ""}
                  fullWidth
                  margin="normal"
                />
                <TextField
                  variant="outlined"
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
                {isNFT ? null : (
                  <TextField
                    variant="outlined"
                    label="Amount"
                    disabled
                    // @ts-ignore
                    value={parsedPayload?.amount.toString() || ""}
                    fullWidth
                    margin="normal"
                  />
                )}
              </div>
            </AccordionDetails>
          </Accordion>
        </div>
      </Card>
    </Container>
  );
}
