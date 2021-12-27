import {
  ChainId,
  CHAIN_ID_AVAX,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
} from "@certusone/wormhole-sdk";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import {
  AccountInfo,
  Connection,
  ParsedAccountData,
  PublicKey,
} from "@solana/web3.js";
import axios from "axios";
import { useEffect, useMemo, useState } from "react";
import { DataWrapper } from "../store/helpers";
import { NFTParsedTokenAccount } from "../store/nftSlice";
import {
  BSC_NFT_BRIDGE_ADDRESS,
  COVALENT_GET_TOKENS_URL,
  ETH_NFT_BRIDGE_ADDRESS,
  getNFTBridgeAddressForChain,
  POLYGON_NFT_BRIDGE_ADDRESS,
  SOLANA_HOST,
  SOL_NFT_CUSTODY_ADDRESS,
} from "../utils/consts";
import { Metadata } from "../utils/metaplex";
import useMetadata, { GenericMetadata } from "./useMetadata";

export type NFTTVL = NFTParsedTokenAccount & { chainId: ChainId };

const calcEvmTVL = (covalentReport: any, chainId: ChainId): NFTTVL[] => {
  const output: NFTTVL[] = [];
  if (!covalentReport?.data?.items?.length) {
    return [];
  }

  covalentReport.data.items.forEach((item: any) => {
    //TODO remove non nfts
    if (item.balance > 0 && item.contract_address && item.nft_data) {
      item.nft_data.forEach((nftData: any) => {
        if (nftData.token_id) {
          output.push({
            amount: item.balance,
            mintKey: item.contract_address,
            tokenId: nftData.token_id,
            publicKey: getNFTBridgeAddressForChain(chainId),
            decimals: 0,
            uiAmount: 0,
            uiAmountString: item.balance.toString(),
            chainId: chainId,
            uri: nftData.token_url,
            animation_url: nftData.external_data?.animation_url,
            external_url: nftData.external_data?.external_url,
            image: nftData.external_data?.image,
            image_256: nftData.external_data?.image_256,
            nftName: nftData.external_data?.name,
            description: nftData.external_data?.description,
          });
        }
      });
    }
  });

  return output;
};
const calcSolanaTVL = (
  accounts:
    | { pubkey: PublicKey; account: AccountInfo<ParsedAccountData> }[]
    | undefined,
  metaData: DataWrapper<Map<string, GenericMetadata>>
) => {
  const output: NFTTVL[] = [];
  if (
    !accounts ||
    !accounts.length ||
    metaData.isFetching ||
    metaData.error ||
    !metaData.data
  ) {
    return output;
  }

  accounts.forEach((item) => {
    const genericMetadata = metaData.data?.get(
      item.account.data.parsed?.info?.mint?.toString()
    );
    const raw: Metadata | undefined = genericMetadata?.raw;

    if (
      item.account.data.parsed?.info?.tokenAmount?.uiAmount > 0 &&
      item.account.data.parsed?.info?.tokenAmount?.decimals === 0
    ) {
      output.push({
        amount: item.account.data.parsed?.info?.tokenAmount?.amount,
        mintKey: item.account.data.parsed?.info?.mint,
        publicKey: getNFTBridgeAddressForChain(CHAIN_ID_SOLANA),
        decimals: 0,
        uiAmount: 0,
        uiAmountString:
          item.account.data.parsed?.info?.tokenAmount?.uiAmountString,
        chainId: CHAIN_ID_SOLANA,
        uri: raw?.data?.uri,
        symbol: raw?.data?.symbol,
        // external_url: nftData.external_data?.external_url,
        // image: nftData.external_data?.image,
        // image_256: nftData.external_data?.image_256,
        // nftName: nftData.external_data?.name,
        // description: nftData.external_data?.description,
      });
    }
  });

  return output;
};

const useNFTTVL = (): DataWrapper<NFTTVL[]> => {
  const [ethCovalentData, setEthCovalentData] = useState(undefined);
  const [ethCovalentIsLoading, setEthCovalentIsLoading] = useState(false);
  const [ethCovalentError, setEthCovalentError] = useState("");

  const [bscCovalentData, setBscCovalentData] = useState(undefined);
  const [bscCovalentIsLoading, setBscCovalentIsLoading] = useState(false);
  const [bscCovalentError, setBscCovalentError] = useState("");

  const [polygonCovalentData, setPolygonCovalentData] = useState(undefined);
  const [polygonCovalentIsLoading, setPolygonCovalentIsLoading] =
    useState(false);
  const [polygonCovalentError, setPolygonCovalentError] = useState("");

  const [avaxCovalentData, setAvaxCovalentData] = useState(undefined);
  const [avaxCovalentIsLoading, setAvaxCovalentIsLoading] = useState(false);
  const [avaxCovalentError, setAvaxCovalentError] = useState("");

  const [solanaCustodyTokens, setSolanaCustodyTokens] = useState<
    { pubkey: PublicKey; account: AccountInfo<ParsedAccountData> }[] | undefined
  >(undefined);
  const [solanaCustodyTokensLoading, setSolanaCustodyTokensLoading] =
    useState(false);
  const [solanaCustodyTokensError, setSolanaCustodyTokensError] = useState("");
  const mintAddresses = useMemo(() => {
    const addresses: string[] = [];
    solanaCustodyTokens?.forEach((item) => {
      const mintKey = item.account.data.parsed?.info?.mint?.toString();
      if (mintKey) {
        addresses.push(mintKey);
      }
    });
    return addresses;
  }, [solanaCustodyTokens]);

  const solanaMetadata = useMetadata(CHAIN_ID_SOLANA, mintAddresses);

  const solanaTVL = useMemo(
    () => calcSolanaTVL(solanaCustodyTokens, solanaMetadata),
    [solanaCustodyTokens, solanaMetadata]
  );
  const ethTVL = useMemo(
    () => calcEvmTVL(ethCovalentData, CHAIN_ID_ETH),
    [ethCovalentData]
  );
  const bscTVL = useMemo(
    () => calcEvmTVL(bscCovalentData, CHAIN_ID_BSC),
    [bscCovalentData]
  );

  const polygonTVL = useMemo(
    () => calcEvmTVL(polygonCovalentData, CHAIN_ID_POLYGON),
    [polygonCovalentData]
  );

  const avaxTVL = useMemo(
    () => calcEvmTVL(avaxCovalentData, CHAIN_ID_AVAX),
    [avaxCovalentData]
  );

  useEffect(() => {
    let cancelled = false;
    setEthCovalentIsLoading(true);
    axios
      .get(
        COVALENT_GET_TOKENS_URL(
          CHAIN_ID_ETH,
          ETH_NFT_BRIDGE_ADDRESS,
          true,
          false
        )
      )
      .then(
        (results) => {
          if (!cancelled) {
            setEthCovalentData(results.data);
            setEthCovalentIsLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setEthCovalentError("Unable to retrieve Ethereum TVL.");
            setEthCovalentIsLoading(false);
          }
        }
      );
  }, []);

  useEffect(() => {
    let cancelled = false;
    setBscCovalentIsLoading(true);
    axios
      .get(
        COVALENT_GET_TOKENS_URL(
          CHAIN_ID_BSC,
          BSC_NFT_BRIDGE_ADDRESS,
          true,
          false
        )
      )
      .then(
        (results) => {
          if (!cancelled) {
            setBscCovalentData(results.data);
            setBscCovalentIsLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setBscCovalentError("Unable to retrieve BSC TVL.");
            setBscCovalentIsLoading(false);
          }
        }
      );
  }, []);

  useEffect(() => {
    let cancelled = false;
    setPolygonCovalentIsLoading(true);
    axios
      .get(
        COVALENT_GET_TOKENS_URL(
          CHAIN_ID_POLYGON,
          POLYGON_NFT_BRIDGE_ADDRESS,
          true,
          false
        )
      )
      .then(
        (results) => {
          if (!cancelled) {
            setPolygonCovalentData(results.data);
            setPolygonCovalentIsLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setPolygonCovalentError("Unable to retrieve Polygon TVL.");
            setPolygonCovalentIsLoading(false);
          }
        }
      );
  }, []);

  useEffect(() => {
    let cancelled = false;
    setAvaxCovalentIsLoading(true);
    axios
      .get(
        COVALENT_GET_TOKENS_URL(
          CHAIN_ID_AVAX,
          getNFTBridgeAddressForChain(CHAIN_ID_AVAX),
          true,
          false
        )
      )
      .then(
        (results) => {
          if (!cancelled) {
            setAvaxCovalentData(results.data);
            setAvaxCovalentIsLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setAvaxCovalentError("Unable to retrieve Polygon TVL.");
            setAvaxCovalentIsLoading(false);
          }
        }
      );
  }, []);

  useEffect(() => {
    let cancelled = false;
    const connection = new Connection(SOLANA_HOST, "confirmed");
    setSolanaCustodyTokensLoading(true);
    connection
      .getParsedTokenAccountsByOwner(new PublicKey(SOL_NFT_CUSTODY_ADDRESS), {
        programId: TOKEN_PROGRAM_ID,
      })
      .then(
        (results) => {
          if (!cancelled) {
            setSolanaCustodyTokens(results.value);
            setSolanaCustodyTokensLoading(false);
          }
        },
        (error) => {
          if (!cancelled) {
            setSolanaCustodyTokensLoading(false);
            setSolanaCustodyTokensError(
              "Unable to retrieve Solana locked tokens."
            );
          }
        }
      );
  }, []);

  return useMemo(() => {
    const tvlArray = [
      ...ethTVL,
      ...bscTVL,
      ...polygonTVL,
      ...avaxTVL,
      ...solanaTVL,
    ];

    return {
      isFetching:
        ethCovalentIsLoading ||
        bscCovalentIsLoading ||
        polygonCovalentIsLoading ||
        avaxCovalentIsLoading ||
        solanaCustodyTokensLoading,
      error:
        ethCovalentError ||
        bscCovalentError ||
        polygonCovalentError ||
        avaxCovalentError ||
        solanaCustodyTokensError,
      receivedAt: null,
      data: tvlArray,
    };
  }, [
    ethCovalentError,
    ethCovalentIsLoading,
    bscCovalentError,
    bscCovalentIsLoading,
    polygonTVL,
    polygonCovalentError,
    polygonCovalentIsLoading,
    ethTVL,
    bscTVL,
    solanaTVL,
    solanaCustodyTokensError,
    solanaCustodyTokensLoading,
    avaxTVL,
    avaxCovalentIsLoading,
    avaxCovalentError,
  ]);
};

export default useNFTTVL;
