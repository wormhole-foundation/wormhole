import {
  CircularProgress,
  createStyles,
  makeStyles,
  TextField,
  Typography,
} from "@material-ui/core";
import { Autocomplete, createFilterOptions } from "@material-ui/lab";
import React, { useCallback, useEffect, useState } from "react";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import { CovalentData } from "../../hooks/useGetSourceParsedTokenAccounts";
import { DataWrapper } from "../../store/helpers";
import { ParsedTokenAccount } from "../../store/transferSlice";
import {
  ethTokenToParsedTokenAccount,
  getEthereumToken,
  isValidEthereumAddress,
} from "../../utils/ethereum";
import { shortenAddress } from "../../utils/solana";
import OffsetButton from "./OffsetButton";

const useStyles = makeStyles(() =>
  createStyles({
    selectInput: { minWidth: "10rem" },
    tokenOverviewContainer: {
      display: "flex",
      "& div": {
        margin: ".5rem",
      },
    },
    tokenImage: {
      maxHeight: "2.5rem",
    },
  })
);

type EthereumSourceTokenSelectorProps = {
  value: ParsedTokenAccount | null;
  onChange: (newValue: ParsedTokenAccount | null) => void;
  covalent: DataWrapper<CovalentData[]> | undefined;
  tokenAccounts: DataWrapper<ParsedTokenAccount[]> | undefined;
  disabled: boolean;
};

const renderAccount = (
  account: ParsedTokenAccount,
  covalentData: CovalentData | undefined,
  classes: any
) => {
  const mintPrettyString = shortenAddress(account.mintKey);
  const uri = covalentData?.logo_url;
  const symbol = covalentData?.contract_ticker_symbol || "Unknown";
  return (
    <div className={classes.tokenOverviewContainer}>
      <div>
        {uri && <img alt="" className={classes.tokenImage} src={uri} />}
      </div>
      <div>
        <Typography variant="subtitle1">{symbol}</Typography>
      </div>
      <div>
        <Typography variant="body1">{mintPrettyString}</Typography>
      </div>
      <div>
        <Typography variant="body2">{"Balance"}</Typography>
        <Typography variant="h6">{account.uiAmountString}</Typography>
      </div>
    </div>
  );
};

export default function EthereumSourceTokenSelector(
  props: EthereumSourceTokenSelectorProps
) {
  const { value, onChange, covalent, tokenAccounts, disabled } = props;
  const classes = useStyles();
  const [advancedMode, setAdvancedMode] = useState(false);
  const [advancedModeLoading, setAdvancedModeLoading] = useState(false);
  const [advancedModeSymbol, setAdvancedModeSymbol] = useState("");
  const [advancedModeHolderString, setAdvancedModeHolderString] = useState("");
  const [advancedModeError, setAdvancedModeError] = useState("");
  const { provider, signerAddress } = useEthereumProvider();

  useEffect(() => {
    //If we receive a push from our parent, usually on component mount, we set the advancedModeString to synchronize.
    //This also kicks off the metadata load.
    if (advancedMode && value && advancedModeHolderString !== value.mintKey) {
      setAdvancedModeHolderString(value.mintKey);
    }
  }, [value, advancedMode, advancedModeHolderString]);

  //This loads the parsedTokenAccount & symbol from the advancedModeString
  //TODO move to util or hook
  useEffect(() => {
    let cancelled = false;
    if (!advancedMode || !isValidEthereumAddress(advancedModeHolderString)) {
      return;
    } else {
      //TODO get a bit smarter about setting & clearing errors
      if (provider === undefined || signerAddress === undefined) {
        !cancelled &&
          setAdvancedModeError("Your Ethereum wallet is no longer connected.");
        return;
      }
      !cancelled && setAdvancedModeLoading(true);
      !cancelled && setAdvancedModeError("");
      !cancelled && setAdvancedModeSymbol("");
      try {
        getEthereumToken(advancedModeHolderString, provider).then((token) => {
          ethTokenToParsedTokenAccount(token, signerAddress).then(
            (parsedTokenAccount) => {
              !cancelled && onChange(parsedTokenAccount);
              !cancelled && setAdvancedModeLoading(false);
            },
            (error) => {
              //These errors can maybe be consolidated
              !cancelled &&
                setAdvancedModeError("Failed to find the specified address");
              !cancelled && setAdvancedModeLoading(false);
            }
          );

          token.symbol().then(
            (result) => {
              !cancelled && setAdvancedModeSymbol(result);
            },
            (error) => {
              !cancelled &&
                setAdvancedModeError("Failed to find the specified address");
              !cancelled && setAdvancedModeLoading(false);
            }
          );
        });
      } catch (error) {
        !cancelled &&
          setAdvancedModeError("Failed to find the specified address");
        !cancelled && setAdvancedModeLoading(false);
      }
    }
    return () => {
      cancelled = true;
    };
  }, [
    advancedModeHolderString,
    advancedMode,
    provider,
    signerAddress,
    onChange,
  ]);

  const handleClick = useCallback(() => {
    onChange(null);
    setAdvancedModeHolderString("");
  }, [onChange]);

  const handleOnChange = useCallback(
    (event) => setAdvancedModeHolderString(event.target.value),
    []
  );

  const getSymbol = (account: ParsedTokenAccount | null) => {
    if (!account) {
      return undefined;
    }
    return covalent?.data?.find((x) => x.contract_address === account.mintKey);
  };

  const filterConfig = createFilterOptions({
    matchFrom: "any",
    stringify: (option: ParsedTokenAccount) => {
      const symbol = getSymbol(option) + " " || "";
      const mint = option.mintKey + " ";

      return symbol + mint;
    },
  });

  const toggleAdvancedMode = () => {
    setAdvancedMode(!advancedMode);
  };

  const isLoading =
    props.covalent?.isFetching || props.tokenAccounts?.isFetching;

  const symbolString = advancedModeSymbol
    ? advancedModeSymbol + " "
    : getSymbol(value)
    ? getSymbol(value)?.contract_ticker_symbol + " "
    : "";

  const autoComplete = (
    <Autocomplete
      autoComplete
      autoHighlight
      autoSelect
      blurOnSelect
      clearOnBlur
      fullWidth={false}
      filterOptions={filterConfig}
      value={value}
      onChange={(event, newValue) => {
        onChange(newValue);
      }}
      disabled={disabled}
      noOptionsText={"No ERC20 tokens found at the moment."}
      options={tokenAccounts?.data || []}
      renderInput={(params) => (
        <TextField {...params} label="Token Account" variant="outlined" />
      )}
      renderOption={(option) => {
        return renderAccount(
          option,
          covalent?.data?.find((x) => x.contract_address === option.mintKey),
          classes
        );
      }}
      getOptionLabel={(option) => {
        const symbol = getSymbol(option);
        return `${symbol ? symbol : "Unknown"} (Account: ${shortenAddress(
          option.publicKey
        )}, Address: ${shortenAddress(option.mintKey)})`;
      }}
    />
  );

  const advancedModeToggleButton = (
    <OffsetButton onClick={toggleAdvancedMode} disabled={disabled}>
      {advancedMode ? "Toggle Token Picker" : "Toggle Override"}
    </OffsetButton>
  );

  const content = value ? (
    <>
      <Typography>{symbolString + value.mintKey}</Typography>
      <OffsetButton onClick={handleClick} disabled={disabled}>
        Clear
      </OffsetButton>
    </>
  ) : isLoading ? (
    <CircularProgress />
  ) : advancedMode ? (
    <>
      <TextField
        fullWidth
        label="Enter an asset address"
        value={advancedModeHolderString}
        onChange={handleOnChange}
        error={
          (advancedModeHolderString !== "" &&
            !isValidEthereumAddress(advancedModeHolderString)) ||
          !!advancedModeError
        }
        helperText={advancedModeError === "" ? undefined : advancedModeError}
        disabled={disabled || advancedModeLoading}
      />
    </>
  ) : (
    autoComplete
  );

  return (
    <React.Fragment>
      {content}
      {!value && !isLoading && advancedModeToggleButton}
    </React.Fragment>
  );
}
