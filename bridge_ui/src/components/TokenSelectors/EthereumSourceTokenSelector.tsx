import { Button, TextField, Typography } from "@material-ui/core";
import React, { useCallback, useEffect, useState } from "react";
import { useEthereumProvider } from "../../contexts/EthereumProviderContext";
import { ParsedTokenAccount } from "../../store/transferSlice";
import {
  ethTokenToParsedTokenAccount,
  getEthereumToken,
  isValidEthereumAddress,
} from "../../utils/ethereum";

type EthereumSourceTokenSelectorProps = {
  value: ParsedTokenAccount | null;
  onChange: (newValue: ParsedTokenAccount | null) => void;
};

export default function EthereumSourceTokenSelector(
  props: EthereumSourceTokenSelectorProps
) {
  const { onChange, value } = props;
  const advancedMode = true; //const [advancedMode, setAdvancedMode] = useState(true);
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

  const symbolString = advancedModeSymbol ? advancedModeSymbol + " " : "";

  const handleClick = useCallback(() => {
    onChange(null);
    setAdvancedModeHolderString("");
  }, [onChange]);

  const handleOnChange = useCallback(
    (event) => setAdvancedModeHolderString(event.target.value),
    []
  );

  const content = value ? (
    <>
      <Typography>{symbolString + value.mintKey}</Typography>
      <Button onClick={handleClick}>Clear</Button>
    </>
  ) : (
    <>
      <TextField
        fullWidth
        label="Asset Address"
        value={advancedModeHolderString}
        onChange={handleOnChange}
        error={
          (advancedModeHolderString !== "" &&
            !isValidEthereumAddress(advancedModeHolderString)) ||
          !!advancedModeError
        }
        helperText={advancedModeError === "" ? undefined : advancedModeError}
        disabled={advancedModeLoading}
      />
    </>
  );

  return <React.Fragment>{content}</React.Fragment>;
}
