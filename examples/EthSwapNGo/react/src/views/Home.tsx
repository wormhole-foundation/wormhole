import {
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  createNonce,
  hexToUint8Array,
  nativeToHexString,
  parseSequenceFromLogEth,
  TokenImplementation__factory,
} from "@certusone/wormhole-sdk";
import { formatUnits, parseUnits } from "@ethersproject/units";
import {
  Button,
  Container,
  makeStyles,
  Paper,
  Typography,
} from "@material-ui/core";
import Collapse from "@material-ui/core/Collapse";
import CheckCircleOutlineRoundedIcon from "@material-ui/icons/CheckCircleOutlineRounded";
import { ethers } from "ethers";
import { useCallback, useEffect, useMemo, useState } from "react";
import { SimpleDex__factory } from "../abi/factories/SimpleDex__factory";
import ButtonWithLoader from "../components/ButtonWithLoader";
import ChainSelectDialog from "../components/ChainSelectDialog";
import CircleLoader from "../components/CircleLoader";
import EthereumSignerKey from "../components/EthereumSignerKey";
import HoverIcon from "../components/HoverIcon";
import NumberTextField from "../components/NumberTextField";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import useAllowance from "../hooks/useAllowance";
import useIsWalletReady from "../hooks/useIsWalletReady";
import useRestRelayer from "../hooks/useRestRelayer";
import Wormhole from "../icons/wormhole-network.svg";
import { COLORS } from "../muiTheme";
import {
  CHAINS,
  getBridgeAddressForChain,
  getDefaultNativeCurrencySymbol,
  getTokenBridgeAddressForChain,
} from "../utils/consts";
import getSwapPool from "../utils/getSwapPoolAddress";

const useStyles = makeStyles((theme) => ({
  numberField: {
    flexGrow: 1,
    "& > * > .MuiInputBase-input": {
      textAlign: "right",
      height: "100%",
      flexGrow: "1",
      fontSize: "3rem",
      fontFamily: "Roboto Mono, monospace",
      caretShape: "block",
      width: "0",
      "&::-webkit-outer-spin-button, &::-webkit-inner-spin-button": {
        "-webkit-appearance": "none",
        "-moz-appearance": "none",
        margin: 0,
      },
      "&[type=number]": {
        "-webkit-appearance": "textfield",
        "-moz-appearance": "textfield",
      },
    },
    "& > * > input::-webkit-inner-spin-button": {
      webkitAppearance: "none",
      margin: "0",
    },
  },
  sourceContainer: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    border: "3px solid #333333",
    padding: ".6rem",
    borderRadius: "30px",
    "& > *": {
      margin: ".5rem",
    },
    margin: "1rem 0rem 1rem 0rem",
  },
  centeredContainer: {
    textAlign: "center",
    width: "100%",
  },
  stepNum: {},
  explainerText: {
    marginBottom: "1rem",
  },
  spacer: {
    height: "1rem",
  },
  mainPaper: {
    padding: "2rem",
    backgroundColor: COLORS.nearBlackWithMinorTransparency,
  },
  chainSelectorContainer: {},
  downArrow: {
    height: "5rem",
  },
  titleBar: {
    marginTop: "10rem",
    "& > *": {
      margin: ".5rem",
      alignSelf: "flex-end",
    },
  },
  appBar: {
    background: COLORS.nearBlackWithMinorTransparency,
    "& > .MuiToolbar-root": {
      margin: "auto",
      width: "100%",
      maxWidth: 1100,
    },
  },
  link: {
    ...theme.typography.body1,
    color: theme.palette.text.primary,
    marginLeft: theme.spacing(6),
    [theme.breakpoints.down("sm")]: {
      marginLeft: theme.spacing(2.5),
    },
    [theme.breakpoints.down("xs")]: {
      marginLeft: theme.spacing(1),
    },
    "&.active": {
      color: theme.palette.primary.light,
    },
  },
  bg: {
    background:
      "linear-gradient(160deg, rgba(69,74,117,.1) 0%, rgba(138,146,178,.1) 33%, rgba(69,74,117,.1) 66%, rgba(98,104,143,.1) 100%), linear-gradient(45deg, rgba(153,69,255,.1) 0%, rgba(121,98,231,.1) 20%, rgba(0,209,140,.1) 100%)",
    display: "flex",
    flexDirection: "column",
    minHeight: "100vh",
  },
  actionArea: {
    display: "grid",
    placeItems: "center",
    height: "10rem",
    width: "100%",
  },
  content: {
    margin: theme.spacing(2, 0),
    [theme.breakpoints.up("md")]: {
      margin: theme.spacing(4, 0),
    },
  },
  brandLink: {
    display: "inline-flex",
    alignItems: "center",
    "&:hover": {
      textDecoration: "none",
    },
  },
  iconButton: {
    [theme.breakpoints.up("md")]: {
      marginRight: theme.spacing(2.5),
    },
    [theme.breakpoints.down("sm")]: {
      marginRight: theme.spacing(2.5),
    },
    [theme.breakpoints.down("xs")]: {
      marginRight: theme.spacing(1),
    },
  },
  gradientButton: {
    backgroundImage: `linear-gradient(45deg, ${COLORS.blue} 0%, ${COLORS.nearBlack}20 50%,  ${COLORS.blue}30 62%, ${COLORS.nearBlack}50  120%)`,
    transition: "0.75s",
    backgroundSize: "200% auto",
    boxShadow: "0 0 20px #222",
    "&:hover": {
      backgroundPosition:
        "right center" /* change the direction of the change here */,
    },
    width: "100%",
    height: "3rem",
    marginTop: "1rem",
  },
  disabled: {
    background: COLORS.gray,
  },
  betaBanner: {
    background: `linear-gradient(to left, ${COLORS.blue}40, ${COLORS.green}40);`,
    padding: theme.spacing(1, 0),
  },
  loaderHolder: {
    display: "flex",
    justifyContent: "center",
    flexDirection: "column",
    alignItems: "center",
  },
  wormholeIcon: {
    height: 60,
    filter: "contrast(0)",
    transition: "filter 0.5s",
    "&:hover": {
      filter: "contrast(1)",
    },
    verticalAlign: "middle",
    margin: "1rem",
    display: "inline-block",
  },
  successIcon: {
    color: COLORS.green,
    fontSize: "200px",
  },
}));

function Home() {
  const classes = useStyles();

  const [transferHolderString, setTransferHolderString] = useState<string>("");
  const [price, setPrice] = useState<BigInt | null>(null);
  const [sequence, setSequence] = useState<string>("");
  const [sourceChain, setSourceChain] = useState(CHAIN_ID_ETH);
  const [targetChain, setTargetChain] = useState(CHAIN_ID_BSC);
  const { swapPoolAddress, targetAsset } = useMemo(() => {
    const holder = getSwapPool(sourceChain, targetChain) as any;
    return {
      swapPoolAddress: holder.poolAddress,
      targetAsset: holder.tokenAddress,
    };
  }, [sourceChain, targetChain]);
  const relayInfo = useRestRelayer(sourceChain, sequence, targetChain);
  const allowanceInfo = useAllowance(
    sourceChain,
    swapPoolAddress,
    targetAsset,
    BigInt(100000000000000000000000),
    false
  );
  console.log("allowance info", allowanceInfo);
  console.log("relay info", relayInfo);
  console.log("sequence", sequence);
  console.log("price", price);
  const { isReady: isEthWalletReady, walletAddress } = useIsWalletReady(
    sourceChain,
    true
  );
  const ethWallet = useEthereumProvider();

  const handleSourceChange = useCallback(
    (event) => {
      const newSourceChain = event.target.value;
      if (newSourceChain === targetChain) {
        const newTargetChain = CHAINS.find(
          (chain) => chain.id !== newSourceChain
        );
        setTargetChain(newTargetChain?.id || CHAIN_ID_ETH);
      }
      setSourceChain(newSourceChain);
      setTransferHolderString("");
    },
    [targetChain]
  );

  const handleTargetChange = useCallback(
    (event) => {
      console.log(event.target.value, "value");
      const newTargetChain = event.target.value;
      if (newTargetChain === sourceChain) {
        const newSourceChain = CHAINS.find(
          (chain) => chain.id !== newTargetChain
        );
        setSourceChain(newSourceChain?.id || CHAIN_ID_ETH);
      }
      setTargetChain(newTargetChain);
      setTransferHolderString("");
    },
    [sourceChain]
  );

  const swapChains = useCallback(() => {
    setSourceChain(targetChain);
    setTargetChain(sourceChain);
    setTransferHolderString("");
  }, [targetChain, sourceChain]);

  const calcPrice = useCallback(async () => {
    if (
      !isEthWalletReady ||
      !transferHolderString ||
      !parseUnits(transferHolderString, 18)
    ) {
      setPrice(null);
      return;
    }
    const dex = SimpleDex__factory.connect(
      swapPoolAddress,
      ethWallet.provider as any
    );
    const token = TokenImplementation__factory.connect(
      targetAsset,
      ethWallet.provider as any
    );
    const tokensInPool = await token.balanceOf(swapPoolAddress);
    const nativeInPool = await dex.totalLiquidity();

    const price = await dex.price(
      parseUnits(transferHolderString, 18),
      nativeInPool,
      tokensInPool
    );
    setPrice(price.toBigInt());
  }, [
    isEthWalletReady,
    swapPoolAddress,
    ethWallet.provider,
    targetAsset,
    transferHolderString,
  ]);

  //calcPrice
  //TODO debounce
  useEffect(() => {
    calcPrice();
  }, [calcPrice]);

  //TODO check that the user has enough balance
  const sufficientPoolBalance =
    price && transferHolderString && parseUnits(transferHolderString, "18");

  const readyToGo =
    isEthWalletReady &&
    walletAddress &&
    !isNaN(parseFloat(transferHolderString)) &&
    parseFloat(transferHolderString) > 0 &&
    allowanceInfo.sufficientAllowance &&
    sufficientPoolBalance; //TODO check pool balances

  const handleTransfer = useCallback(async () => {
    if (!readyToGo || !walletAddress) {
      return;
    }
    console.log(swapPoolAddress);
    const dex = SimpleDex__factory.connect(
      swapPoolAddress,
      ethWallet.signer as any
    );

    const nonce = createNonce();
    const receipt = await (
      await dex.swapNGo(
        getTokenBridgeAddressForChain(sourceChain),
        targetChain,
        hexToUint8Array(nativeToHexString(walletAddress, targetChain) as any),
        "0",
        nonce,
        { value: parseUnits(transferHolderString, 18) }
      )
    ).wait();

    console.log("transaction receipt", receipt);

    const sequence = parseSequenceFromLogEth(
      receipt,
      getBridgeAddressForChain(sourceChain)
    );

    setSequence(sequence);
  }, [
    ethWallet.signer,
    readyToGo,
    sourceChain,
    swapPoolAddress,
    targetChain,
    transferHolderString,
    walletAddress,
  ]);

  const handleAllowanceIncrease = useCallback(() => {
    allowanceInfo.approveAmount(ethers.constants.MaxUint256.toBigInt());
  }, [allowanceInfo]);
  const handleAmountChange = useCallback((event) => {
    setTransferHolderString(event.target.value);
  }, []);

  const handleReset = useCallback(() => {
    setTransferHolderString("");
    setSequence("");
  }, []);

  const sourceContent = (
    <div className={classes.sourceContainer}>
      <ChainSelectDialog
        value={sourceChain}
        onChange={handleSourceChange}
        chains={CHAINS}
        style2={true}
      />
      <NumberTextField
        className={classes.numberField}
        value={transferHolderString}
        onChange={handleAmountChange}
        autoFocus={true}
        InputProps={{ disableUnderline: true }}
      />
    </div>
  );
  const middleButton = <HoverIcon onClick={swapChains} />; //TODO onclick
  const targetContent = (
    <div className={classes.sourceContainer}>
      <ChainSelectDialog
        value={targetChain}
        onChange={handleTargetChange}
        chains={CHAINS}
      />
      <NumberTextField
        className={classes.numberField}
        value={
          !!price
            ? parseFloat(formatUnits(price.toString(), 18)).toFixed(2)
            : "0"
        }
        InputProps={{ disableUnderline: true }}
        disabled={true}
      />
    </div>
  );

  const walletButton = <EthereumSignerKey />;

  const allowanceButton = (
    <>
      <ButtonWithLoader
        onClick={handleAllowanceIncrease}
        showLoader={allowanceInfo.isApproveProcessing}
        disabled={allowanceInfo.isApproveProcessing || !isEthWalletReady}
      >
        Allow Wormhole Transfers
      </ButtonWithLoader>
    </>
  );

  const convertButton = (
    <ButtonWithLoader
      disabled={!readyToGo}
      onClick={handleTransfer}
      className={
        classes.gradientButton + (!readyToGo ? " " + classes.disabled : "")
      }
    >
      Convert
    </ButtonWithLoader>
  );
  const buttonContent = !allowanceInfo.sufficientAllowance
    ? allowanceButton
    : convertButton;

  return (
    <div className={classes.bg}>
      <Container className={classes.centeredContainer} maxWidth="sm">
        <div className={classes.titleBar}></div>
        <Typography variant="h4" color="textSecondary">
          {"Crypto Converter"}
        </Typography>
        <div className={classes.spacer} />
        <Paper className={classes.mainPaper}>
          <Collapse in={!!relayInfo.isComplete}>
            <>
              <CheckCircleOutlineRoundedIcon
                fontSize={"inherit"}
                className={classes.successIcon}
              />
              <Typography>All Set!</Typography>
              <Typography variant="h5">
                {"You now have " +
                  parseFloat(formatUnits((price || 0).toString(), 18)).toFixed(
                    2
                  ) +
                  " " +
                  getDefaultNativeCurrencySymbol(targetChain)}
              </Typography>
              <div className={classes.spacer} />
              <div className={classes.spacer} />
              <Button onClick={handleReset} variant="contained" color="primary">
                Convert More Coins
              </Button>
            </>
          </Collapse>
          <div className={classes.loaderHolder}>
            <Collapse in={!!relayInfo.isLoading && !relayInfo.isComplete}>
              <div className={classes.loaderHolder}>
                <CircleLoader />
                <div className={classes.spacer} />
                <div className={classes.spacer} />
                <Typography variant="h5">
                  {"Your " +
                    getDefaultNativeCurrencySymbol(sourceChain) +
                    " is being converted to " +
                    getDefaultNativeCurrencySymbol(targetChain)}
                </Typography>
                <div className={classes.spacer} />
                <Typography>{"Please wait a moment"}</Typography>
              </div>
            </Collapse>
          </div>
          <div className={classes.chainSelectorContainer}>
            <Collapse
              in={
                !!ethWallet.provider &&
                !relayInfo.isLoading &&
                !relayInfo.isComplete &&
                !sequence
              }
            >
              {ethWallet.provider ? (
                <>
                  {sourceContent}
                  {middleButton}
                  {targetContent}
                  <div className={classes.spacer} />
                  <div className={classes.spacer} />
                </>
              ) : null}
              {buttonContent}
            </Collapse>
            {!ethWallet.provider && walletButton}
          </div>
        </Paper>
        <div className={classes.spacer} />
        <Typography variant="subtitle1" color="textSecondary">
          {"powered by wormhole"}
        </Typography>
        <img src={Wormhole} alt="Wormhole" className={classes.wormholeIcon} />
      </Container>
    </div>
  );
}

export default Home;
