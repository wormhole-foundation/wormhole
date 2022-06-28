import { ChainId, isNativeDenom, TerraChainId } from "@certusone/wormhole-sdk";
import { formatUnits } from "@ethersproject/units";
import { LCDClient } from "@terra-money/terra.js";
import React, { useCallback, useMemo, useRef } from "react";
import { createParsedTokenAccount } from "../../hooks/useGetSourceParsedTokenAccounts";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import useTerraNativeBalances from "../../hooks/useTerraNativeBalances";
import { DataWrapper } from "../../store/helpers";
import { NFTParsedTokenAccount } from "../../store/nftSlice";
import { ParsedTokenAccount } from "../../store/transferSlice";
import { SUPPORTED_TERRA_TOKENS, getTerraConfig } from "../../utils/consts";
import {
  formatNativeDenom,
  getNativeTerraIcon,
  isValidTerraAddress,
  NATIVE_TERRA_DECIMALS,
} from "../../utils/terra";
import TokenPicker, { BasicAccountRender } from "./TokenPicker";

type TerraTokenPickerProps = {
  value: ParsedTokenAccount | null;
  onChange: (newValue: ParsedTokenAccount | null) => void;
  tokenAccounts: DataWrapper<ParsedTokenAccount[]> | undefined;
  disabled: boolean;
  resetAccounts: (() => void) | undefined;
  chainId: TerraChainId;
};

const returnsFalse = () => false;

export default function TerraTokenPicker(props: TerraTokenPickerProps) {
  const { value, onChange, disabled, chainId } = props;
  const { walletAddress } = useIsWalletReady(chainId);
  const nativeRefresh = useRef<() => void>(() => {});
  const { balances, isLoading: nativeIsLoading } = useTerraNativeBalances(
    chainId,
    walletAddress,
    nativeRefresh
  );

  const resetAccountWrapper = useCallback(() => {
    //we can currently skip calling this as we don't read from sourceParsedTokenAccounts
    //resetAccounts && resetAccounts();
    nativeRefresh.current();
  }, []);
  const isLoading = nativeIsLoading; // || (tokenMap?.isFetching || false);

  const onChangeWrapper = useCallback(
    async (account: NFTParsedTokenAccount | null) => {
      if (account === null) {
        onChange(null);
        return Promise.resolve();
      }
      onChange(account);
      return Promise.resolve();
    },
    [onChange]
  );

  const terraTokenArray = useMemo(() => {
    const balancesItems =
      balances && walletAddress
        ? Object.keys(balances).map((denom) =>
            // ({
            //   protocol: "native",
            //   symbol: formatNativeDenom(denom),
            //   token: denom,
            //   icon: getNativeTerraIcon(formatNativeDenom(denom)),
            //   balance: balances[denom],
            // } as TerraTokenMetadata)

            //TODO support non-natives in the SUPPORTED_TERRA_TOKENS
            //This token account makes a lot of assumptions
            createParsedTokenAccount(
              walletAddress,
              denom,
              balances[denom], //amount
              NATIVE_TERRA_DECIMALS, //TODO actually get decimals rather than hardcode
              0, //uiAmount is unused
              formatUnits(balances[denom], NATIVE_TERRA_DECIMALS), //uiAmountString
              formatNativeDenom(denom, props.chainId), // symbol
              undefined, //name
              getNativeTerraIcon(formatNativeDenom(denom, props.chainId)), //logo
              true //is native asset
            )
          )
        : [];
    return balancesItems.filter((metadata) =>
      SUPPORTED_TERRA_TOKENS.includes(metadata.mintKey)
    );
    // const values = tokenMap.data?.mainnet;
    // const tokenMapItems = Object.values(values || {}) || [];
    // return [...balancesItems, ...tokenMapItems];
  }, [
    walletAddress,
    balances,
    props.chainId,
    // tokenMap
  ]);

  //TODO this only supports non-native assets. Native assets come from the hook.
  //TODO correlate against token list to get metadata
  const lookupTerraAddress = useCallback(
    (lookupAsset: string) => {
      if (!walletAddress) {
        return Promise.reject("Wallet not connected");
      }
      const lcd = new LCDClient(getTerraConfig(chainId));
      return lcd.wasm
        .contractQuery(lookupAsset, {
          token_info: {},
        })
        .then((info: any) =>
          lcd.wasm
            .contractQuery(lookupAsset, {
              balance: {
                address: walletAddress,
              },
            })
            .then((balance: any) => {
              if (balance && info) {
                return createParsedTokenAccount(
                  walletAddress,
                  lookupAsset,
                  balance.balance.toString(),
                  info.decimals,
                  Number(formatUnits(balance.balance, info.decimals)),
                  formatUnits(balance.balance, info.decimals),
                  info.symbol,
                  info.name
                );
              } else {
                throw new Error("Failed to retrieve Terra account.");
              }
            })
        )
        .catch(() => {
          return Promise.reject();
        });
    },
    [walletAddress, chainId]
  );

  const isSearchableAddress = useCallback(
    (address: string, chainId: ChainId) => {
      return (
        isValidTerraAddress(address, chainId as TerraChainId) &&
        !isNativeDenom(address)
      );
    },
    []
  );

  const RenderComp = useCallback(
    ({ account }: { account: NFTParsedTokenAccount }) => {
      return BasicAccountRender(account, returnsFalse, false);
    },
    []
  );

  return (
    <TokenPicker
      value={value}
      options={terraTokenArray || []}
      RenderOption={RenderComp}
      onChange={onChangeWrapper}
      isValidAddress={isSearchableAddress}
      getAddress={lookupTerraAddress}
      disabled={disabled}
      resetAccounts={resetAccountWrapper}
      error={""}
      showLoader={isLoading}
      nft={false}
      chainId={chainId}
    />
  );
}
