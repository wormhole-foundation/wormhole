import {
  createStyles,
  makeStyles,
  TextField,
  Typography,
} from "@material-ui/core";
import { Autocomplete, createFilterOptions } from "@material-ui/lab";
import { LCDClient } from "@terra-money/terra.js";
import {
  ConnectedWallet,
  useConnectedWallet,
} from "@terra-money/wallet-provider";
import { formatUnits } from "ethers/lib/utils";
import React, { useCallback, useMemo, useState } from "react";
import {
  createParsedTokenAccount,
  TerraTokenMap,
  TerraTokenMetadata,
} from "../../hooks/useGetSourceParsedTokenAccounts";
import { DataWrapper } from "../../store/helpers";
import { ParsedTokenAccount } from "../../store/transferSlice";
import { TERRA_HOST } from "../../utils/consts";
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

type TerraSourceTokenSelectorProps = {
  value: ParsedTokenAccount | null;
  onChange: (newValue: ParsedTokenAccount | null) => void;
  disabled: boolean;
  tokenMap: DataWrapper<TerraTokenMap> | undefined; //TODO better type
};

//TODO move elsewhere
//TODO async
const lookupTerraAddress = (
  lookupAsset: string,
  terraWallet: ConnectedWallet
) => {
  const lcd = new LCDClient(TERRA_HOST);
  return lcd.wasm
    .contractQuery(lookupAsset, {
      token_info: {},
    })
    .then((info: any) =>
      lcd.wasm
        .contractQuery(lookupAsset, {
          balance: {
            address: terraWallet.walletAddress,
          },
        })
        .then((balance: any) => {
          if (balance && info) {
            return createParsedTokenAccount(
              terraWallet.walletAddress,
              lookupAsset,
              balance.balance.toString(),
              info.decimals,
              Number(formatUnits(balance.balance, info.decimals)),
              formatUnits(balance.balance, info.decimals)
            );
          } else {
            throw new Error("Failed to retrieve Terra account.");
          }
        })
    )
    .catch(() => {
      return Promise.reject();
    });
};

export default function TerraSourceTokenSelector(
  props: TerraSourceTokenSelectorProps
) {
  const classes = useStyles();
  const { onChange, value, disabled, tokenMap } = props;
  const [advancedMode, setAdvancedMode] = useState(false);
  const [advancedModeHolderString, setAdvancedModeHolderString] = useState("");
  const [advancedModeError, setAdvancedModeError] = useState("");
  const terraWallet = useConnectedWallet();

  const isLoading = tokenMap?.isFetching || false;

  const terraTokenArray = useMemo(() => {
    const values = props.tokenMap?.data?.mainnet;
    const items = Object.values(values || {});
    return items || [];
  }, [props.tokenMap]);

  const valueToOption = (fromProps: ParsedTokenAccount | undefined | null) => {
    if (!fromProps) return undefined;
    else {
      return terraTokenArray.find((x) => x.token === fromProps.mintKey);
    }
  };
  const handleClick = useCallback(() => {
    onChange(null);
    setAdvancedModeHolderString("");
  }, [onChange]);

  const handleOnChange = useCallback(
    (event) => setAdvancedModeHolderString(event.target.value),
    []
  );

  const handleConfirm = (address: string | undefined) => {
    if (terraWallet === undefined || address === undefined) {
      setAdvancedModeError("Terra wallet not connected.");
      return;
    }
    setAdvancedModeError("");
    lookupTerraAddress(address, terraWallet).then(
      (result) => {
        onChange(result);
      },
      (error) => {
        setAdvancedModeError("Unable to retrieve this address.");
      }
    );
    setAdvancedModeError("");
  };

  const filterConfig = createFilterOptions({
    matchFrom: "any",
    stringify: (option: TerraTokenMetadata) => {
      const symbol = option.symbol + " " || "";
      const mint = option.token + " " || "";
      const name = option.protocol + " " || "";

      return symbol + mint + name;
    },
  });

  const renderOptionLabel = (option: TerraTokenMetadata) => {
    return option.symbol + " (" + shortenAddress(option.token) + ")";
  };
  const renderOption = (option: TerraTokenMetadata) => {
    return (
      <div className={classes.tokenOverviewContainer}>
        <div>
          <img alt="" className={classes.tokenImage} src={option.icon} />
        </div>
        <div>
          <Typography variant="h6">{option.symbol}</Typography>
          <Typography variant="body2">{option.protocol}</Typography>
        </div>
        <div>
          <Typography variant="body1">{option.token}</Typography>
        </div>
      </div>
    );
  };

  const toggleAdvancedMode = () => {
    setAdvancedMode(!advancedMode);
    setAdvancedModeError("");
  };

  const advancedModeToggleButton = (
    <OffsetButton onClick={toggleAdvancedMode} disabled={disabled}>
      {advancedMode ? "Toggle Token Picker" : "Toggle Override"}
    </OffsetButton>
  );

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
        value={valueToOption(value)}
        onChange={(event, newValue) => {
          handleConfirm(newValue?.token);
        }}
        disabled={disabled}
        noOptionsText={"No CW20 tokens found at the moment."}
        options={terraTokenArray}
        renderInput={(params) => (
          <TextField {...params} label="Token" variant="outlined" />
        )}
        renderOption={renderOption}
        getOptionLabel={renderOptionLabel}
      />
      {advancedModeError && (
        <Typography color="error">{advancedModeError}</Typography>
      )}
    </>
  );

  const content = value ? (
    <>
      <Typography>{value.mintKey}</Typography>
      <OffsetButton onClick={handleClick} disabled={disabled}>
        Clear
      </OffsetButton>
    </>
  ) : !advancedMode ? (
    autoComplete
  ) : (
    <>
      <TextField
        fullWidth
        label="Enter an asset address"
        value={advancedModeHolderString}
        onChange={handleOnChange}
        disabled={disabled}
        error={advancedModeHolderString !== "" && !!advancedModeError}
        helperText={advancedModeError === "" ? undefined : advancedModeError}
      />
      <OffsetButton
        onClick={() => handleConfirm(advancedModeHolderString)}
        disabled={disabled}
      >
        Confirm
      </OffsetButton>
    </>
  );

  return (
    <React.Fragment>
      {content}
      {!value && !isLoading && advancedModeToggleButton}
    </React.Fragment>
  );
}
