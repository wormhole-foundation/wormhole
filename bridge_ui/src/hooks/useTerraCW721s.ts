import { LCDClient } from "@terra-money/terra.js";
import axios, { AxiosResponse } from "axios";
import { useEffect, useLayoutEffect, useMemo, useState } from "react";
import { DataWrapper } from "../store/helpers";
import { CLUSTER, TERRA_HOST, TERRA_NFT_BRIDGE_ADDRESS } from "../utils/consts";
import { alias } from "../utils/alias";
import { ApolloClient, InMemoryCache } from "@apollo/client";
import { NFTParsedTokenAccount } from "../store/nftSlice";
import { createNFTParsedTokenAccount } from "./useGetSourceParsedTokenAccounts";

const safeIPFS = (uri: string) =>
    uri.startsWith("ipfs://ipfs/")
        ? uri.replace("ipfs://", "https://ipfs.io/")
        : uri.startsWith("ipfs://")
        ? uri.replace("ipfs://", "https://ipfs.io/ipfs/")
        : uri.startsWith("https://cloudflare-ipfs.com/ipfs/") // no CORS support?
        ? uri.replace("https://cloudflare-ipfs.com/ipfs/", "https://ipfs.io/ipfs/")
        : uri;

type TerraCW721Address = {
    contract: string,
}

type TerraCW721 = TerraCW721Address & {
    name: string,
    symbol: string
}

type TerraCW721Token = TerraCW721 & {
    token: {
        id: string,
    }
};

export type TerraCW721TokenWithMeta = TerraCW721Token & {
    token: {
        image: string,
        name?: string
    }
};

export const enum ShouldIncludeWrappedAssets {
    Include,
    Exclude
}


function useWhitelist(include_wrapped: ShouldIncludeWrappedAssets): TerraCW721Address[] | undefined {
    const [result, setResult] = useState<TerraCW721Address[]>();
    useEffect(() => {
        const load = async () => {
            let contracts: TerraCW721Address[];
            try {
                const axios_result =
                    await axios.get("http://assets.terra.money/cw721/contracts.json");
                const whitelist: { [key: string]: any } = axios_result.data.mainnet;
                contracts = Object.entries(whitelist).map(([_, v]) => (
                    { contract: v.contract }
                ));
            } catch (e) {
                console.error(e);
                contracts = [];
            }

            let wrapped: TerraCW721Address[] = [];
            try {
                if (include_wrapped === ShouldIncludeWrappedAssets.Include) {
                    const lcd = new LCDClient(TERRA_HOST);
                    const wrapped_tokens = await lcd.wasm.contractQuery<string[]>(TERRA_NFT_BRIDGE_ADDRESS, {all_wrapped_assets: {}});
                    wrapped = wrapped_tokens.map(contract => ({contract}));

                }
            } catch (e) {
                console.error(e);
            }
            setResult([...contracts, ...wrapped]);
        };
        load();
    }, [])
    return result;
}

export function useTerraNFTBalance(owner_address: string | undefined, include_wrapped: ShouldIncludeWrappedAssets): NFTParsedTokenAccount[] | undefined {
    const [result, setResult] = useState<NFTParsedTokenAccount[] | undefined>(undefined);

    const mantle = "https://mantle.terra.dev";
    const whitelist = useWhitelist(include_wrapped);
    useEffect(() => {
        if (owner_address && whitelist) {
            const load = async () => {
                const client = new ApolloClient({
                    uri: mantle,
                    cache: new InMemoryCache()
                });

                const contract_info_queries =
                    whitelist.map((token) => ({
                        contract: token.contract,
                        msg: { contract_info: {} }
                    })) ;

                if (contract_info_queries.length === 0) {
                    setResult([]);
                    return;
                }


                const { data: contract_infos } = await client.query<{ [key: string]: { Result: string } }>({
                    query: alias(contract_info_queries),
                    errorPolicy: "ignore"
                });

                const with_infos: TerraCW721[] = whitelist.flatMap((token) => {
                    if (!contract_infos[token.contract]) { return [] };
                    const info: any = JSON.parse(contract_infos[token.contract].Result);
                    return { ...token, name: info.name, symbol: info.symbol };
                });

                const owned_queries =
                    with_infos.map((token) => ({
                        contract: token.contract,
                        msg: { tokens: { owner: owner_address } }
                    }));

                if (owned_queries.length === 0) {
                    setResult([]);
                    return;
                }

                const { data: owned_tokens } = await client.query({
                    query: alias(owned_queries),
                    errorPolicy: "ignore"
                });

                const owned: TerraCW721Token[] = with_infos.flatMap((token) => {
                    if (!owned_tokens[token.contract]) { return [] };
                    const tokens: string[] = JSON.parse(owned_tokens[token.contract].Result).tokens;
                    return tokens.map((id) => ({
                        ...token, token: { id }
                    }))
                });

                const mk_unique = (token: TerraCW721Token) =>
                    token.contract + token.token.id

                const meta_queries =
                    owned.map((token) => ({
                        alias: mk_unique(token),
                        contract: token.contract,
                        msg: { nft_info: { token_id: token.token.id } }
                    }));

                if (meta_queries.length === 0) {
                    setResult([]);
                    return;
                }

                const { data: metas } = await client.query<{ [key: string]: { Result: string } }>({
                    query: alias(meta_queries),
                    errorPolicy: "ignore"
                });

                let nfts: NFTParsedTokenAccount[] = [];

                for (const token of owned) {
                    if (!metas[mk_unique(token)]) { continue };
                    const meta = JSON.parse(metas[mk_unique(token)].Result);
                    let uri = meta.token_uri ? await axios.get(safeIPFS(meta.token_uri)) : null;
                    const uri_data = uri?.data.result?.data || uri?.data;
                    const image = uri_data?.image || meta.extension?.image;
                    const name = uri_data?.name || meta.extension?.name;
                    const description = uri_data?.description || meta.extension?.description;

                    nfts.push({
                        publicKey: owner_address,
                        mintKey: token.contract,
                        amount: "1",
                        decimals: 0,
                        uiAmount: 1,
                        uiAmountString: "1",
                        symbol: token.symbol,
                        name: token.name,
                        tokenId: token.token.id,
                        uri: meta.token_uri,
                        animation_url: meta.extension?.animation_url,
                        external_url: meta.extension?.external_url,
                        image: image ? safeIPFS(image) : undefined,
                        image_256: image ? safeIPFS(image) : undefined,
                        nftName: name,
                        description,
                    });
                }

                setResult(nfts);

            };

            load();
        }
    }, [owner_address, whitelist]);

    return result;
}
