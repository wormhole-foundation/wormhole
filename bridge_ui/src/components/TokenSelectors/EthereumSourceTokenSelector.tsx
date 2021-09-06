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
import { WormholeAbi__factory } from "@certusone/wormhole-sdk/lib/ethers-contracts/abi";
import { WORMHOLE_V1_ETH_ADDRESS } from "../../utils/consts";

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

const isWormholev1 = (provider: any, address: string) => {
  const connection = WormholeAbi__factory.connect(
    WORMHOLE_V1_ETH_ADDRESS,
    provider
  );
  return connection.isWrappedAsset(address);
};

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

  const [autocompleteHolder, setAutocompleteHolder] =
    useState<ParsedTokenAccount | null>(null);
  const [autocompleteError, setAutocompleteError] = useState("");

  const { provider, signerAddress } = useEthereumProvider();

  // const wrappedTestToken = "0x8bf3c393b588bb6ad021e154654493496139f06d";
  // const notWrappedTestToken = "0xaaaebe6fe48e54f431b0c390cfaf0b017d09d42d";

  useEffect(() => {
    //If we receive a push from our parent, usually on component mount, we set our internal value to synchronize
    //This also kicks off the metadata load.
    if (advancedMode && value && advancedModeHolderString !== value.mintKey) {
      setAdvancedModeHolderString(value.mintKey);
    }
    if (!advancedMode && value && !autocompleteHolder) {
      setAutocompleteHolder(value);
    }
  }, [value, advancedMode, advancedModeHolderString, autocompleteHolder]);

  //This effect is watching the autocomplete selection.
  //It checks to make sure the token is a valid choice before putting it on the state.
  //At present, that just means it can't be wormholev1
  useEffect(() => {
    if (advancedMode || !autocompleteHolder || !provider) {
      return;
    } else {
      let cancelled = false;
      setAutocompleteError("");
      isWormholev1(provider, autocompleteHolder.mintKey).then(
        (result) => {
          if (!cancelled) {
            result
              ? setAutocompleteError(
                  "Wormhole v1 tokens cannot be transferred with this bridge."
                )
              : onChange(autocompleteHolder);
          }
        },
        (error) => {
          console.log(error);
          if (!cancelled) {
            setAutocompleteError(
              "Warning: please verify if this is a Wormhole v1 token address. V1 tokens should not be transferred with this bridge"
            );
            onChange(autocompleteHolder);
          }
        }
      );
      return () => {
        cancelled = true;
      };
    }
  }, [autocompleteHolder, provider, advancedMode, onChange]);

  //This effect watches the advancedModeString, and checks that the selected asset is valid before putting
  // it on the state.
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
        //Validate that the token is not a wormhole v1 asset
        const isWormholePromise = isWormholev1(
          provider,
          advancedModeHolderString
        ).then(
          (result) => {
            if (result && !cancelled) {
              setAdvancedModeError(
                "Wormhole v1 assets are not eligible for transfer."
              );
              setAdvancedModeLoading(false);
              return Promise.reject();
            } else {
              return Promise.resolve();
            }
          },
          (error) => {
            !cancelled &&
              setAdvancedModeError(
                "Warning: please verify if this is a Wormhole v1 token address. V1 tokens should not be transferred with this bridge"
              );
            !cancelled && setAdvancedModeLoading(false);
            return Promise.resolve(); //Don't allow an error here to tank the workflow
          }
        );

        //Then fetch the asset's information & transform to a parsed token account
        isWormholePromise.then(() =>
          getEthereumToken(advancedModeHolderString, provider).then(
            (token) => {
              ethTokenToParsedTokenAccount(token, signerAddress).then(
                (parsedTokenAccount) => {
                  !cancelled && onChange(parsedTokenAccount);
                  !cancelled && setAdvancedModeLoading(false);
                },
                (error) => {
                  //These errors can maybe be consolidated
                  !cancelled &&
                    setAdvancedModeError(
                      "Failed to find the specified address"
                    );
                  !cancelled && setAdvancedModeLoading(false);
                }
              );

              //Also attempt to store off the symbol
              token.symbol().then(
                (result) => {
                  !cancelled && setAdvancedModeSymbol(result);
                },
                (error) => {
                  !cancelled &&
                    setAdvancedModeError(
                      "Failed to find the specified address"
                    );
                  !cancelled && setAdvancedModeLoading(false);
                }
              );
            },
            (error) => {}
          )
        );
      } catch (e) {
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
    const item = covalent?.data?.find(
      (x) => x.contract_address === account.mintKey
    );
    return item ? item.contract_ticker_symbol : undefined;
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
    setAdvancedModeHolderString("");
    setAdvancedModeError("");
    setAdvancedModeSymbol("");
    setAdvancedMode(!advancedMode);
  };

  const handleAutocompleteChange = (newValue: ParsedTokenAccount | null) => {
    setAutocompleteHolder(newValue);
  };

  const isLoading =
    props.covalent?.isFetching || props.tokenAccounts?.isFetching;

  const autoComplete = (
    <>
      <Autocomplete
        autoComplete
        autoHighlight
        autoSelect
        blurOnSelect
        clearOnBlur
        fullWidth={false}
        filterOptions={filterConfig}
        value={autocompleteHolder}
        onChange={(event, newValue) => {
          handleAutocompleteChange(newValue);
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
          return `${symbol ? symbol : "Unknown"} (Address: ${shortenAddress(
            option.mintKey
          )})`;
        }}
      />
      {autocompleteError && (
        <Typography color="error">{autocompleteError}</Typography>
      )}
    </>
  );

  const advancedModeToggleButton = (
    <OffsetButton onClick={toggleAdvancedMode} disabled={disabled}>
      {advancedMode ? "Toggle Token Picker" : "Toggle Override"}
    </OffsetButton>
  );

  const symbol = getSymbol(value) || advancedModeSymbol;

  const content = value ? (
    <>
      <Typography>{(symbol ? symbol + " " : "") + value.mintKey}</Typography>
      <OffsetButton onClick={handleClick} disabled={disabled}>
        Clear
      </OffsetButton>
      {!advancedMode && autocompleteError ? (
        <Typography color="error">{autocompleteError}</Typography>
      ) : advancedMode && advancedModeError ? (
        <Typography color="error">{advancedModeError}</Typography>
      ) : null}
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
