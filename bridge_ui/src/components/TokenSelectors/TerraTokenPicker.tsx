import { CHAIN_ID_TERRA, isNativeDenom } from "@certusone/wormhole-sdk";
import { formatUnits } from "@ethersproject/units";
import { LCDClient } from "@terra-money/terra.js";
import axios from "axios";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  createNFTParsedTokenAccount,
  createParsedTokenAccount,
} from "../../hooks/useGetSourceParsedTokenAccounts";
import useIsWalletReady from "../../hooks/useIsWalletReady";
import { ShouldIncludeWrappedAssets, useTerraNFTBalance } from "../../hooks/useTerraCW721s";
import useTerraNativeBalances from "../../hooks/useTerraNativeBalances";
import { DataWrapper } from "../../store/helpers";
import { NFTParsedTokenAccount } from "../../store/nftSlice";
import { ParsedTokenAccount } from "../../store/transferSlice";
import { SUPPORTED_TERRA_TOKENS, TERRA_HOST, TERRA_NFT_BRIDGE_ADDRESS } from "../../utils/consts";
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
  nft?: boolean;
};

const returnsFalse = () => false;

export default function TerraTokenPicker(props: TerraTokenPickerProps) {
  const { value, onChange, disabled, nft } = props;
  const { walletAddress } = useIsWalletReady(CHAIN_ID_TERRA);
  const nativeRefresh = useRef<() => void>(() => { });
  const { balances, isLoading: nativeIsLoading } = useTerraNativeBalances(
    walletAddress,
    nativeRefresh
  );

  const nfts = useTerraNFTBalance(walletAddress, ShouldIncludeWrappedAssets.Include);

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
      !nft && balances && walletAddress
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
            formatNativeDenom(denom), // symbol
            undefined, //name
            getNativeTerraIcon(formatNativeDenom(denom)), //logo
            true //is native asset
          )
        )
        : [];
    return balancesItems.filter((metadata) =>
      SUPPORTED_TERRA_TOKENS.includes(metadata.mintKey)
    ).concat(nfts ?? []);
    // const values = tokenMap.data?.mainnet;
    // const tokenMapItems = Object.values(values || {}) || [];
    // return [...balancesItems, ...tokenMapItems];
  }, [
    walletAddress,
    balances,
    nfts,
    // tokenMap
  ]);

  //TODO this only supports non-native assets. Native assets come from the hook.
  //TODO correlate against token list to get metadata
  const lookupTerraAddress = useCallback(
    async (lookupAsset: string, tokenId?: string) => {
      if (!walletAddress) {
        return Promise.reject("Wallet not connected");
      }
      if (nft && !tokenId) {
        return Promise.reject("Token ID required");
      }
      const lcd = new LCDClient(TERRA_HOST);
      try {
        if (nft) {
          const info: any = await lcd.wasm
            .contractQuery(lookupAsset, {
              nft_info: {
                token_id: tokenId,
              },
            });
          const ownerInfo: any = await lcd.wasm
            .contractQuery(lookupAsset, {
              owner_of: {
                token_id: tokenId,
              },
            });
          const contractInfo: any = await lcd.wasm
            .contractQuery(lookupAsset, {
              contract_info: {}
            });
          if (ownerInfo && info) {
            return createNFTParsedTokenAccount(
              walletAddress,
              lookupAsset,
              ownerInfo.owner === walletAddress ? "1" : "0",
              0,
              Number(ownerInfo.owner === walletAddress ? "1" : "0"),
              ownerInfo.owner === walletAddress ? "1" : "0",
              tokenId || "",
              contractInfo.symbol,
              contractInfo.name,
              info.token_uri,
              info.extension?.animation_url,
              info.extension?.external_url,
              info.extension?.image,
              undefined, // image_256
              info.extension?.name,
              info.extension?.description,
            );
          } else {
            throw new Error("Failed to retrieve Terra account.");
          }
        } else {
          const info: any = await lcd.wasm
            .contractQuery(lookupAsset, {
              token_info: {},
            });
          const balance: any = await lcd.wasm
            .contractQuery(lookupAsset, {
              balance: {
                address: walletAddress,
              },
            });

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
        };
      } catch (e) {
        return Promise.reject(e);
      }
    },
    [walletAddress, nft]
  );

  const isSearchableAddress = useCallback((address: string) => {
    return isValidTerraAddress(address) && !isNativeDenom(address);
  }, []);

  const RenderComp = useCallback(
    ({ account }: { account: NFTParsedTokenAccount }) => {
      return BasicAccountRender(account, returnsFalse, nft ? nft : false);
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
      nft={nft || false}
      useTokenId={nft}
      chainId={CHAIN_ID_TERRA}
    />
  );
}
