import {
  ChainId,
  CHAIN_ID_ALGORAND,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  getForeignAssetEth,
  getForeignAssetSolana,
  getForeignAssetTerra,
  hexToNativeString,
  hexToUint8Array,
  isEVMChain,
} from "@certusone/wormhole-sdk";
import {
  accountExists,
  decodeLocalState,
  SEED_AMT,
} from "@certusone/wormhole-sdk/lib/esm/algorand/Algorand";
import {
  PopulateData,
  TmplSig,
  uint8ArrayToHexString,
} from "@certusone/wormhole-sdk/lib/esm/algorand/TmplSig";
import MyAlgoConnect from "@randlabs/myalgo-connect";
import algosdk from "algosdk";
import {
  getForeignAssetEth as getForeignAssetEthNFT,
  getForeignAssetSol as getForeignAssetSolNFT,
} from "@certusone/wormhole-sdk/lib/esm/nft_bridge";
import { BigNumber } from "@ethersproject/bignumber";
import { arrayify } from "@ethersproject/bytes";
import { Connection } from "@solana/web3.js";
import { LCDClient } from "@terra-money/terra.js";
import { ethers } from "ethers";
import { useCallback, useEffect, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { useAlgorandContext } from "../contexts/AlgorandWalletContext";
import { useEthereumProvider } from "../contexts/EthereumProviderContext";
import {
  errorDataWrapper,
  fetchDataWrapper,
  receiveDataWrapper,
} from "../store/helpers";
import { setTargetAsset as setNFTTargetAsset } from "../store/nftSlice";
import {
  selectNFTIsSourceAssetWormholeWrapped,
  selectNFTOriginAsset,
  selectNFTOriginChain,
  selectNFTOriginTokenId,
  selectNFTTargetChain,
  selectTransferIsSourceAssetWormholeWrapped,
  selectTransferOriginAsset,
  selectTransferOriginChain,
  selectTransferTargetChain,
} from "../store/selectors";
import { setTargetAsset as setTransferTargetAsset } from "../store/transferSlice";
import {
  ALGORAND_HOST,
  ALGORAND_TOKEN_BRIDGE_ID,
  getEvmChainId,
  getNFTBridgeAddressForChain,
  getTokenBridgeAddressForChain,
  SOLANA_HOST,
  SOL_NFT_BRIDGE_ADDRESS,
  SOL_TOKEN_BRIDGE_ADDRESS,
  TERRA_HOST,
  TERRA_TOKEN_BRIDGE_ADDRESS,
} from "../utils/consts";

export async function optin(
  client: algosdk.Algodv2,
  senderAddr: string,
  appId: number,
  appIndex: number,
  emitterId: string,
  why: string
): Promise<string> {
  console.log("optin called with ", appIndex, emitterId, why);

  // This is the application address associated with the application ID
  const appAddr: string = algosdk.getApplicationAddress(appId);
  const decAppAddr: Uint8Array = algosdk.decodeAddress(appAddr).publicKey;
  const aa: string = uint8ArrayToHexString(decAppAddr, false);

  let data: PopulateData = {
    addrIdx: appIndex,
    appAddress: aa,
    appId: appId,
    emitterId: emitterId,
    seedAmt: SEED_AMT,
  };

  console.log("YYY", JSON.stringify(data));

  const ts: TmplSig = new TmplSig(client);
  const lsa: algosdk.LogicSigAccount = await ts.populate(data);
  const sigAddr: string = lsa.address();

  // Check to see if we need to create this
  console.log("Checking to see if account exists...", appIndex, "-", emitterId);
  const retval: boolean = await accountExists(client, appId, sigAddr);
  if (!retval) {
    // console.log("Account does not exist.");
    // These are the suggested params from the system
    // console.log("Getting parms...");
    const params = await client.getTransactionParams().do();
    // console.log("Creating payment txn...");
    const seedTxn = algosdk.makePaymentTxnWithSuggestedParamsFromObject({
      from: senderAddr,
      to: sigAddr,
      amount: SEED_AMT,
      suggestedParams: params,
    });
    // console.log("Creating optin txn...");
    const optinTxn = algosdk.makeApplicationOptInTxnFromObject({
      from: sigAddr,
      suggestedParams: params,
      appIndex: appId,
    });
    // console.log("Creating rekey txn...");
    const rekeyTxn = algosdk.makePaymentTxnWithSuggestedParamsFromObject({
      from: sigAddr,
      to: sigAddr,
      amount: 0,
      suggestedParams: params,
      rekeyTo: appAddr,
    });

    // console.log("Assigning group ID...");
    let txns = [seedTxn, optinTxn, rekeyTxn];
    algosdk.assignGroupID(txns);

    // console.log("Signing seed for optin...");
    const myAlgoConnect = new MyAlgoConnect();
    const signedSeedTxn = await myAlgoConnect.signTransaction(seedTxn.toByte());
    // console.log("Signing optin for optin...");
    const signedOptinTxn = algosdk.signLogicSigTransaction(optinTxn, lsa);
    // console.log("Signing rekey for optin...");
    const signedRekeyTxn = algosdk.signLogicSigTransaction(rekeyTxn, lsa);

    // console.log(
    //     "Sending txns for optin...",
    //     appIndex,
    //     "-",
    //     emitterId,
    //     "-",
    //     sigAddr
    // );
    await client
      .sendRawTransaction([
        signedSeedTxn.blob,
        signedOptinTxn.blob,
        signedRekeyTxn.blob,
      ])
      .do();

    // console.log(
    //     "Awaiting confirmation for optin...",
    //     appIndex,
    //     "-",
    //     emitterId
    // );
    const confirmedTxns = await algosdk.waitForConfirmation(
      client,
      txns[txns.length - 1].txID(),
      1
    );
    console.log("optin confirmation", confirmedTxns);
  }
  return sigAddr;
}

// TODO: make this usable in sdk
export async function getForeignAssetAlgo(
  client: algosdk.Algodv2,
  senderAddr: string,
  chain: number,
  contract: string
): Promise<number> {
  if (chain === 8) {
    return parseInt(contract, 16);
  } else {
    let chainAddr = await optin(
      client,
      senderAddr,
      ALGORAND_TOKEN_BRIDGE_ID,
      chain,
      contract,
      "getForeignAssetAlgo"
    );
    let asset: Uint8Array = await decodeLocalState(
      client,
      ALGORAND_TOKEN_BRIDGE_ID,
      chainAddr
    );
    if (asset.length > 8) {
      const tmp = Buffer.from(asset.slice(0, 8));
      return Number(tmp.readBigUInt64BE(0));
    } else throw new Error("unknownForeignAsset");
  }
}

function useFetchTargetAsset(nft?: boolean) {
  const dispatch = useDispatch();
  const isSourceAssetWormholeWrapped = useSelector(
    nft
      ? selectNFTIsSourceAssetWormholeWrapped
      : selectTransferIsSourceAssetWormholeWrapped
  );
  const originChain = useSelector(
    nft ? selectNFTOriginChain : selectTransferOriginChain
  );
  const originAsset = useSelector(
    nft ? selectNFTOriginAsset : selectTransferOriginAsset
  );
  const originTokenId = useSelector(selectNFTOriginTokenId);
  const tokenId = originTokenId || ""; // this should exist by this step for NFT transfers
  const targetChain = useSelector(
    nft ? selectNFTTargetChain : selectTransferTargetChain
  );
  const setTargetAsset = nft ? setNFTTargetAsset : setTransferTargetAsset;
  const { provider, chainId: evmChainId } = useEthereumProvider();
  const correctEvmNetwork = getEvmChainId(targetChain);
  const hasCorrectEvmNetwork = evmChainId === correctEvmNetwork;
  const { accounts: algorandAccounts } = useAlgorandContext();
  const [lastSuccessfulArgs, setLastSuccessfulArgs] = useState<{
    isSourceAssetWormholeWrapped: boolean | undefined;
    originChain: ChainId | undefined;
    originAsset: string | undefined;
    targetChain: ChainId;
    nft?: boolean;
    tokenId?: string;
  } | null>(null);
  const argsMatchLastSuccess =
    !!lastSuccessfulArgs &&
    lastSuccessfulArgs.isSourceAssetWormholeWrapped ===
      isSourceAssetWormholeWrapped &&
    lastSuccessfulArgs.originChain === originChain &&
    lastSuccessfulArgs.originAsset === originAsset &&
    lastSuccessfulArgs.targetChain === targetChain &&
    lastSuccessfulArgs.nft === nft &&
    lastSuccessfulArgs.tokenId === tokenId;
  const setArgs = useCallback(
    () =>
      setLastSuccessfulArgs({
        isSourceAssetWormholeWrapped,
        originChain,
        originAsset,
        targetChain,
        nft,
        tokenId,
      }),
    [
      isSourceAssetWormholeWrapped,
      originChain,
      originAsset,
      targetChain,
      nft,
      tokenId,
    ]
  );
  useEffect(() => {
    if (argsMatchLastSuccess) {
      return;
    }
    setLastSuccessfulArgs(null);
    if (isSourceAssetWormholeWrapped && originChain === targetChain) {
      dispatch(
        setTargetAsset(
          receiveDataWrapper({
            doesExist: true,
            address: hexToNativeString(originAsset, originChain) || null,
          })
        )
      );
      setArgs();
      return;
    }
    let cancelled = false;
    (async () => {
      if (
        isEVMChain(targetChain) &&
        provider &&
        hasCorrectEvmNetwork &&
        originChain &&
        originAsset
      ) {
        dispatch(setTargetAsset(fetchDataWrapper()));
        try {
          const asset = await (nft
            ? getForeignAssetEthNFT(
                getNFTBridgeAddressForChain(targetChain),
                provider,
                originChain,
                hexToUint8Array(originAsset)
              )
            : getForeignAssetEth(
                getTokenBridgeAddressForChain(targetChain),
                provider,
                originChain,
                hexToUint8Array(originAsset)
              ));
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                receiveDataWrapper({
                  doesExist: asset !== ethers.constants.AddressZero,
                  address: asset,
                })
              )
            );
            setArgs();
          }
        } catch (e) {
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                errorDataWrapper(
                  "Unable to determine existence of wrapped asset"
                )
              )
            );
          }
        }
      }
      if (targetChain === CHAIN_ID_SOLANA && originChain && originAsset) {
        dispatch(setTargetAsset(fetchDataWrapper()));
        try {
          const connection = new Connection(SOLANA_HOST, "confirmed");
          const asset = await (nft
            ? getForeignAssetSolNFT(
                SOL_NFT_BRIDGE_ADDRESS,
                originChain,
                hexToUint8Array(originAsset),
                arrayify(BigNumber.from(tokenId || "0"))
              )
            : getForeignAssetSolana(
                connection,
                SOL_TOKEN_BRIDGE_ADDRESS,
                originChain,
                hexToUint8Array(originAsset)
              ));
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                receiveDataWrapper({ doesExist: !!asset, address: asset })
              )
            );
            setArgs();
          }
        } catch (e) {
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                errorDataWrapper(
                  "Unable to determine existence of wrapped asset"
                )
              )
            );
          }
        }
      }
      if (targetChain === CHAIN_ID_TERRA && originChain && originAsset) {
        dispatch(setTargetAsset(fetchDataWrapper()));
        try {
          const lcd = new LCDClient(TERRA_HOST);
          const asset = await getForeignAssetTerra(
            TERRA_TOKEN_BRIDGE_ADDRESS,
            lcd,
            originChain,
            hexToUint8Array(originAsset)
          );
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                receiveDataWrapper({ doesExist: !!asset, address: asset })
              )
            );
            setArgs();
          }
        } catch (e) {
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                errorDataWrapper(
                  "Unable to determine existence of wrapped asset"
                )
              )
            );
          }
        }
      }
      if (
        targetChain === CHAIN_ID_ALGORAND &&
        originChain &&
        originAsset &&
        algorandAccounts[0]
      ) {
        dispatch(setTargetAsset(fetchDataWrapper()));
        try {
          const algodClient = new algosdk.Algodv2(
            ALGORAND_HOST.algodToken,
            ALGORAND_HOST.algodServer,
            ALGORAND_HOST.algodPort
          );
          const asset = await getForeignAssetAlgo(
            algodClient,
            algorandAccounts[0].address,
            originChain,
            originAsset
          );
          console.log("foreign asset algo:", asset);
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                receiveDataWrapper({
                  doesExist: !!asset,
                  address: asset.toString(),
                })
              )
            );
            setArgs();
          }
        } catch (e) {
          console.error(e);
          if (!cancelled) {
            dispatch(
              setTargetAsset(
                errorDataWrapper(
                  "Unable to determine existence of wrapped asset"
                )
              )
            );
          }
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [
    dispatch,
    isSourceAssetWormholeWrapped,
    originChain,
    originAsset,
    targetChain,
    provider,
    nft,
    setTargetAsset,
    tokenId,
    hasCorrectEvmNetwork,
    argsMatchLastSuccess,
    setArgs,
    algorandAccounts,
  ]);
}

export default useFetchTargetAsset;
