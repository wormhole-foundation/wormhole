import { ChainId } from "@certusone/wormhole-sdk";
import { BigNumber } from "@ethersproject/bignumber";
import {
  Button,
  CircularProgress,
  createStyles,
  Dialog,
  DialogContent,
  DialogTitle,
  IconButton,
  List,
  ListItem,
  makeStyles,
  TextField,
  Tooltip,
  Typography,
} from "@material-ui/core";
import KeyboardArrowDownIcon from "@material-ui/icons/KeyboardArrowDown";
import RefreshIcon from "@material-ui/icons/Refresh";
import { Alert } from "@material-ui/lab";
import React, { useCallback, useEffect, useMemo, useState } from "react";
import { NFTParsedTokenAccount } from "../../store/nftSlice";
import { shortenAddress } from "../../utils/solana";
import NFTViewer from "./NFTViewer";

const useStyles = makeStyles((theme) =>
  createStyles({
    alignCenter: {
      textAlign: "center",
    },
    optionContainer: {
      padding: 0,
    },
    optionContent: {
      padding: theme.spacing(1),
    },
    tokenList: {
      maxHeight: theme.spacing(80), //TODO smarter
      height: theme.spacing(80),
      overflow: "auto",
    },
    dialogContent: {
      overflowX: "hidden",
    },
    selectionButtonContainer: {
      //display: "flex",
      textAlign: "center",
      marginTop: theme.spacing(2),
      marginBottom: theme.spacing(2),
    },
    selectionButton: {
      maxWidth: "100%",
      width: theme.breakpoints.values.sm,
    },
    tokenOverviewContainer: {
      display: "flex",
      width: "100%",
      alignItems: "center",
      "& div": {
        margin: theme.spacing(1),
        flexBasis: "25%",
        "&$tokenImageContainer": {
          maxWidth: 40,
        },
        "&:last-child": {
          textAlign: "right",
        },
        flexShrink: 1,
      },
      flexWrap: "wrap",
    },
    tokenImageContainer: {
      display: "flex",
      alignItems: "center",
      justifyContent: "center",
      width: 40,
    },
    tokenImage: {
      maxHeight: "2.5rem", //Eyeballing this based off the text size
    },
    migrationAlert: {
      width: "100%",
      "& .MuiAlert-message": {
        width: "100%",
      },
    },
    flexTitle: {
      display: "flex",
      flexDirection: "row",
      alignItems: "center",
    },
    grower: {
      flexGrow: 1,
    },
  })
);

const balancePretty = (uiString: string) => {
  const numberString = uiString.split(".")[0];
  const bignum = BigNumber.from(numberString);
  if (bignum.gte(1000000)) {
    return numberString.substring(0, numberString.length - 6) + " M";
  } else if (uiString.length > 8) {
    return uiString.substr(0, 8);
  } else {
    return uiString;
  }
};

export const BasicAccountRender = (
  account: NFTParsedTokenAccount,
  isMigrationEligible: (address: string) => boolean,
  nft: boolean
) => {
  const classes = useStyles();
  const mintPrettyString = shortenAddress(account.mintKey);
  const uri = nft ? account.image_256 : account.logo || account.uri;
  const symbol = account.symbol || "Unknown";
  const name = account.name || "Unknown";
  const tokenId = account.tokenId;
  const balancePrettyString = balancePretty(account.uiAmountString);

  const nftContent = (
    <div className={classes.tokenOverviewContainer}>
      <div className={classes.tokenImageContainer}>
        {uri && <img alt="" className={classes.tokenImage} src={uri} />}
      </div>
      <div>
        <Typography>{symbol}</Typography>
        <Typography>{name}</Typography>
      </div>
      <div>
        <Typography>{mintPrettyString}</Typography>
        <Typography>{tokenId}</Typography>
      </div>
    </div>
  );

  const tokenContent = (
    <div className={classes.tokenOverviewContainer}>
      <div className={classes.tokenImageContainer}>
        {uri && <img alt="" className={classes.tokenImage} src={uri} />}
      </div>
      <div>
        <Typography variant="subtitle1">{symbol}</Typography>
      </div>
      <div>
        {
          <Typography variant="body1">
            {account.isNativeAsset ? "Native" : mintPrettyString}
          </Typography>
        }
      </div>
      <div>
        <Typography variant="body2">{"Balance"}</Typography>
        <Typography variant="h6">{balancePrettyString}</Typography>
      </div>
    </div>
  );

  const migrationRender = (
    <div className={classes.migrationAlert}>
      <Alert severity="warning">
        <Typography variant="body2">
          This is a legacy asset eligible for migration.
        </Typography>
        <div>{tokenContent}</div>
      </Alert>
    </div>
  );

  return nft
    ? nftContent
    : isMigrationEligible(account.mintKey)
    ? migrationRender
    : tokenContent;
};

export default function TokenPicker({
  value,
  options,
  RenderOption,
  onChange,
  isValidAddress,
  getAddress,
  disabled,
  resetAccounts,
  nft,
  chainId,
  error,
  showLoader,
  useTokenId,
}: {
  value: NFTParsedTokenAccount | null;
  options: NFTParsedTokenAccount[];
  RenderOption: ({
    account,
  }: {
    account: NFTParsedTokenAccount;
  }) => JSX.Element;
  onChange: (newValue: NFTParsedTokenAccount | null) => Promise<void>;
  isValidAddress?: (address: string) => boolean;
  getAddress?: (
    address: string,
    tokenId?: string
  ) => Promise<NFTParsedTokenAccount>;
  disabled: boolean;
  resetAccounts: (() => void) | undefined;
  nft: boolean;
  chainId: ChainId;
  error?: string;
  showLoader?: boolean;
  useTokenId?: boolean;
}) {
  const classes = useStyles();
  const [holderString, setHolderString] = useState("");
  const [tokenIdHolderString, setTokenIdHolderString] = useState("");
  const [loadingError, setLoadingError] = useState("");
  const [isLocalLoading, setLocalLoading] = useState(false);
  const [dialogIsOpen, setDialogIsOpen] = useState(false);
  const [selectionError, setSelectionError] = useState("");

  const openDialog = useCallback(() => {
    setHolderString("");
    setDialogIsOpen(true);
  }, []);

  const closeDialog = useCallback(() => {
    setDialogIsOpen(false);
  }, []);

  const handleSelectOption = useCallback(
    async (option: NFTParsedTokenAccount) => {
      setSelectionError("");
      onChange(option).then(
        () => {
          closeDialog();
        },
        (error) => {
          setSelectionError(error?.message || "Error verifying the token.");
        }
      );
    },
    [onChange, closeDialog]
  );

  const filteredOptions = useMemo(() => {
    return options.filter((option: NFTParsedTokenAccount) => {
      if (!holderString) {
        return true;
      }
      const optionString = (
        (option.publicKey || "") +
        " " +
        (option.mintKey || "") +
        " " +
        (option.symbol || "") +
        " " +
        (option.name || " ")
      ).toLowerCase();
      const searchString = holderString.toLowerCase();
      return optionString.includes(searchString);
    });
  }, [holderString, options]);

  const localFind = useCallback(
    (address: string, tokenIdHolderString: string) => {
      return options.find(
        (x) =>
          x.mintKey === address &&
          (!tokenIdHolderString || x.tokenId === tokenIdHolderString)
      );
    },
    [options]
  );

  //This is the effect which allows pasting an address in directly
  useEffect(() => {
    if (!isValidAddress || !getAddress) {
      return;
    }
    if (useTokenId && !tokenIdHolderString) {
      return;
    }
    let cancelled = false;
    if (isValidAddress(holderString)) {
      const option = localFind(holderString, tokenIdHolderString);
      if (option) {
        handleSelectOption(option);
        return;
      }
      setLocalLoading(true);
      setLoadingError("");
      getAddress(
        holderString,
        useTokenId ? tokenIdHolderString : undefined
      ).then(
        (result) => {
          if (!cancelled) {
            setLocalLoading(false);
            if (result) {
              handleSelectOption(result);
            }
          }
        },
        (error) => {
          if (!cancelled) {
            setLocalLoading(false);
            setLoadingError("Could not find the specified address.");
          }
        }
      );
    }
  }, [
    holderString,
    isValidAddress,
    getAddress,
    handleSelectOption,
    localFind,
    tokenIdHolderString,
    useTokenId,
  ]);

  //TODO reset button
  //TODO debounce & save hotloaded options as an option before automatically selecting
  //TODO sigfigs function on the balance strings

  const localLoader = (
    <div className={classes.alignCenter}>
      <CircularProgress />
      <Typography variant="body2">
        {showLoader ? "Loading available tokens" : "Searching for results"}
      </Typography>
    </div>
  );

  const displayLocalError = (
    <div className={classes.alignCenter}>
      <CircularProgress />
      <Typography variant="body2" color="error">
        {loadingError || selectionError}
      </Typography>
    </div>
  );

  const dialog = (
    <Dialog
      onClose={closeDialog}
      aria-labelledby="simple-dialog-title"
      open={dialogIsOpen}
      maxWidth="sm"
      fullWidth
    >
      <DialogTitle>
        <div id="simple-dialog-title" className={classes.flexTitle}>
          <Typography variant="h5">Select a token</Typography>
          <div className={classes.grower} />
          <Tooltip title="Reload tokens">
            <IconButton onClick={resetAccounts}>
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </div>
      </DialogTitle>
      <DialogContent className={classes.dialogContent}>
        <TextField
          variant="outlined"
          label="Search"
          value={holderString}
          onChange={(event) => setHolderString(event.target.value)}
          fullWidth
          margin="normal"
        />
        {useTokenId ? (
          <TextField
            variant="outlined"
            label="Token Id"
            value={tokenIdHolderString}
            onChange={(event) => setTokenIdHolderString(event.target.value)}
            fullWidth
            margin="normal"
          />
        ) : null}
        {isLocalLoading || showLoader ? (
          localLoader
        ) : loadingError || selectionError ? (
          displayLocalError
        ) : filteredOptions.length ? (
          <List className={classes.tokenList}>
            {filteredOptions.map((option) => {
              return (
                <ListItem
                  button
                  onClick={() => handleSelectOption(option)}
                  key={
                    option.publicKey + option.mintKey + (option.tokenId || "")
                  }
                >
                  <RenderOption account={option} />
                </ListItem>
              );
            })}
          </List>
        ) : (
          <div className={classes.alignCenter}>
            <Typography>No results found</Typography>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );

  const selectionChip = (
    <div className={classes.selectionButtonContainer}>
      <Button
        onClick={openDialog}
        disabled={disabled}
        variant="outlined"
        endIcon={<KeyboardArrowDownIcon />}
        className={classes.selectionButton}
      >
        {value ? (
          <RenderOption account={value} />
        ) : (
          <Typography color="textSecondary">Select a token</Typography>
        )}
      </Button>
    </div>
  );

  return (
    <>
      {dialog}
      {value && nft ? <NFTViewer value={value} chainId={chainId} /> : null}
      {selectionChip}
    </>
  );
}
