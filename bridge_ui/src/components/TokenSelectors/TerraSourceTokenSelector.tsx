import { TextField, Typography } from "@material-ui/core";
import { LCDClient } from "@terra-money/terra.js";
import {
  ConnectedWallet,
  useConnectedWallet,
} from "@terra-money/wallet-provider";
import { formatUnits } from "ethers/lib/utils";
import React, { useCallback, useState } from "react";
import { createParsedTokenAccount } from "../../hooks/useGetSourceParsedTokenAccounts";
import { ParsedTokenAccount } from "../../store/transferSlice";
import { TERRA_HOST } from "../../utils/consts";
import OffsetButton from "./OffsetButton";

type TerraSourceTokenSelectorProps = {
  value: ParsedTokenAccount | null;
  onChange: (newValue: ParsedTokenAccount | null) => void;
  disabled: boolean;
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
  const { onChange, value, disabled } = props;
  //const advancedMode = true; //const [advancedMode, setAdvancedMode] = useState(true);
  const [advancedModeHolderString, setAdvancedModeHolderString] = useState("");
  const [advancedModeError, setAdvancedModeError] = useState("");
  const terraWallet = useConnectedWallet();

  const handleClick = useCallback(() => {
    onChange(null);
    setAdvancedModeHolderString("");
  }, [onChange]);

  const handleOnChange = useCallback(
    (event) => setAdvancedModeHolderString(event.target.value),
    []
  );

  const handleConfirm = () => {
    if (terraWallet === undefined) {
      setAdvancedModeError("Terra wallet not connected.");
      return;
    }
    lookupTerraAddress(advancedModeHolderString, terraWallet).then(
      (result) => {
        onChange(result);
      },
      (error) => {
        setAdvancedModeError("Unable to retrieve address.");
      }
    );
    setAdvancedModeError("");
  };

  const content = value ? (
    <>
      <Typography>{value.mintKey}</Typography>
      <OffsetButton onClick={handleClick} disabled={disabled}>
        Clear
      </OffsetButton>
    </>
  ) : (
    <>
      <TextField
        fullWidth
        label="Asset Address"
        value={advancedModeHolderString}
        onChange={handleOnChange}
        disabled={disabled}
        error={advancedModeHolderString !== "" && !!advancedModeError}
        helperText={advancedModeError === "" ? undefined : advancedModeError}
      />
      <OffsetButton onClick={handleConfirm} disabled={disabled}>
        Confirm
      </OffsetButton>
    </>
  );

  return <React.Fragment>{content}</React.Fragment>;
}
