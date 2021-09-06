import { CircularProgress, TextField, Typography } from "@material-ui/core";
import { createStyles, makeStyles, Theme } from "@material-ui/core/styles";
import { Autocomplete } from "@material-ui/lab";
import { createFilterOptions } from "@material-ui/lab/Autocomplete";
import { TokenInfo } from "@solana/spl-token-registry";
import React, { useCallback, useMemo } from "react";
import { DataWrapper } from "../../store/helpers";
import { ParsedTokenAccount } from "../../store/transferSlice";
import { WORMHOLE_V1_MINT_AUTHORITY } from "../../utils/consts";
import { Metadata } from "../../utils/metaplex";
import { shortenAddress } from "../../utils/solana";

const useStyles = makeStyles((theme: Theme) =>
  createStyles({
    selectInput: { minWidth: "10rem" },
    tokenOverviewContainer: {
      display: "flex",
      "& div": {
        margin: ".5rem",
      },
    },
    tokenImage: {
      maxHeight: "2.5rem", //Eyeballing this based off the text size
    },
  })
);

type SolanaSourceTokenSelectorProps = {
  value: ParsedTokenAccount | null;
  onChange: (newValue: ParsedTokenAccount | null) => void;
  accounts: ParsedTokenAccount[];
  solanaTokenMap: DataWrapper<TokenInfo[]> | undefined;
  metaplexData: any; //DataWrapper<(Metadata | undefined)[]> | undefined | null;
  disabled: boolean;
  mintAccounts: DataWrapper<Map<String, string | null>> | undefined;
};

export default function SolanaSourceTokenSelector(
  props: SolanaSourceTokenSelectorProps
) {
  const { value, onChange, disabled } = props;
  const classes = useStyles();

  const memoizedTokenMap: Map<String, TokenInfo> = useMemo(() => {
    const output = new Map<String, TokenInfo>();

    if (props.solanaTokenMap?.data) {
      for (const data of props.solanaTokenMap.data) {
        if (data && data.address) {
          output.set(data.address, data);
        }
      }
    }

    return output;
  }, [props.solanaTokenMap]);

  const memoizedMetaplex: Map<String, Metadata> = useMemo(() => {
    const output = new Map<String, Metadata>();

    if (props.metaplexData.data) {
      for (const data of props.metaplexData.data) {
        if (data && data.mint) {
          output.set(data.mint, data);
        }
      }
    }

    return output;
  }, [props.metaplexData]);

  const getSymbol = (account: ParsedTokenAccount) => {
    return (
      memoizedTokenMap.get(account.mintKey)?.symbol ||
      memoizedMetaplex.get(account.mintKey)?.data?.symbol ||
      undefined
    );
  };

  const getName = (account: ParsedTokenAccount) => {
    return (
      memoizedTokenMap.get(account.mintKey)?.name ||
      memoizedMetaplex.get(account.mintKey)?.data?.name ||
      undefined
    );
  };

  //I wish there was a way  to make this more intelligent,
  //but the autocomplete filterConfig options seem pretty limiting.
  const filterConfig = createFilterOptions({
    matchFrom: "any",
    stringify: (option: ParsedTokenAccount) => {
      const symbol = getSymbol(option) + " " || "";
      const name = getName(option) + " " || "";
      const mint = option.mintKey + " ";
      const pubkey = option.publicKey + " ";

      return symbol + name + mint + pubkey;
    },
  });

  const isWormholev1 = useCallback(
    (address: string) => {
      //This is a v1 wormhole token on testnet
      //const testAddress = "4QixXecTZ4zdZGa39KH8gVND5NZ2xcaB12wiBhE4S7rn";

      if (!props.mintAccounts?.data) {
        return true; //These should never be null by this point
      }
      const mintInfo = props.mintAccounts.data.get(address);

      if (!mintInfo) {
        return true; //We should never fail to pull the mint of an account.
      }

      if (mintInfo === WORMHOLE_V1_MINT_AUTHORITY) {
        return true; //This means the mint was created by the wormhole v1 contract, and we want to disallow its transfer.
      }

      return false;
    },
    [props.mintAccounts]
  );

  const renderAccount = (
    account: ParsedTokenAccount,
    solanaTokenMap: Map<String, TokenInfo>,
    metaplexData: Map<String, Metadata>,
    classes: any
  ) => {
    const tokenMapData = solanaTokenMap.get(account.mintKey);
    const metaplexValue = metaplexData.get(account.mintKey);

    const mintPrettyString = shortenAddress(account.mintKey);
    const accountAddressPrettyString = shortenAddress(account.publicKey);
    const uri = tokenMapData?.logoURI || metaplexValue?.data?.uri || undefined;
    const symbol =
      tokenMapData?.symbol || metaplexValue?.data.symbol || "Unknown";
    const name = tokenMapData?.name || metaplexValue?.data?.name || "--";

    const content = (
      <>
        <div className={classes.tokenOverviewContainer}>
          <div>
            {uri && <img alt="" className={classes.tokenImage} src={uri} />}
          </div>
          <div>
            <Typography variant="subtitle1">{symbol}</Typography>
            <Typography variant="subtitle2">{name}</Typography>
          </div>
          <div>
            <Typography variant="body1">
              {"Mint : " + mintPrettyString}
            </Typography>
            <Typography variant="body1">
              {"Account :" + accountAddressPrettyString}
            </Typography>
          </div>
          <div>
            <Typography variant="body2">{"Balance"}</Typography>
            <Typography variant="h6">{account.uiAmountString}</Typography>
          </div>
        </div>
      </>
    );

    const v1Warning = (
      <div>
        <Typography variant="body2">
          Wormhole v1 tokens are not eligible for transfer.
        </Typography>
        <div>{content}</div>
      </div>
    );

    return isWormholev1(account.mintKey) ? v1Warning : content;
  };

  //The autocomplete doesn't rerender the option label unless the value changes.
  //Thus we should wait for the metadata to arrive before rendering it.
  //TODO This can flicker dependent on how fast the useEffects in the getSourceAccounts hook complete.
  const isLoading =
    props.metaplexData.isFetching ||
    props.solanaTokenMap?.isFetching ||
    props.mintAccounts?.isFetching;

  const accountLoadError =
    !(props.mintAccounts?.isFetching || props.mintAccounts?.data) &&
    "Unable to retrieve your token accounts";
  const error = accountLoadError;

  //This exists to remove NFTs from the list of potential options. It requires reading the metaplex data, so it would be
  //difficult to do before this point.
  const filteredOptions = useMemo(() => {
    return props.accounts.filter((x) => {
      //TODO, do a better check which likely involves supply or checking masterEdition.
      const isNFT =
        x.decimals === 0 && memoizedMetaplex.get(x.mintKey)?.data?.uri;
      return !isNFT;
    });
  }, [memoizedMetaplex, props.accounts]);

  const isOptionDisabled = useMemo(() => {
    return (value: ParsedTokenAccount) => isWormholev1(value.mintKey);
  }, [isWormholev1]);

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
      options={filteredOptions}
      renderInput={(params) => (
        <TextField {...params} label="Token Account" variant="outlined" />
      )}
      renderOption={(option) => {
        return renderAccount(
          option,
          memoizedTokenMap,
          memoizedMetaplex,
          classes
        );
      }}
      getOptionDisabled={isOptionDisabled}
      getOptionLabel={(option) => {
        const symbol = getSymbol(option);
        return `${symbol ? symbol : "Unknown"} (Account: ${shortenAddress(
          option.publicKey
        )}, Mint: ${shortenAddress(option.mintKey)})`;
      }}
    />
  );

  return (
    <React.Fragment>
      {isLoading ? <CircularProgress /> : autoComplete}
      {error && <Typography color="error">{error}</Typography>}
    </React.Fragment>
  );
}
