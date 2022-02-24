import axios from  "axios";
import fs from "fs";
import { SigningCosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import { GasPrice, calculateFee, StdFee } from "@cosmjs/stargate";
import {  DirectSecp256k1HdWallet, makeCosmoshubPath } from "@cosmjs/proto-signing";
import { Slip10RawIndex } from "@cosmjs/crypto";
import path from "path";
/*
 * This is a set of helpers meant for use with @cosmjs/cli
 * With these you can easily use the cw721 contract without worrying about forming messages and parsing queries.
 * 
 * Usage: npx @cosmjs/cli@^0.26 --init https://raw.githubusercontent.com/CosmWasm/cosmwasm-plus/master/contracts/cw721-base/helpers.ts
 * 
 * Create a client:
 *   const [addr, client] = await useOptions(pebblenetOptions).setup('password');
 * 
 * Get the mnemonic:
 *   await useOptions(pebblenetOptions).recoverMnemonic(password);
 * 
 * Create contract:
 *   const contract = CW721(client, pebblenetOptions.fees);
 * 
 * Upload contract:
 *   const codeId = await contract.upload(addr);
 *
 * Instantiate contract example:
 *   const initMsg = {
 *     name: "Potato Coin",
 *     symbol: "TATER",
 *     minter: addr
 *   };
 *   const instance = await contract.instantiate(addr, codeId, initMsg, 'Potato Coin!');
 * If you want to use this code inside an app, you will need several imports from https://github.com/CosmWasm/cosmjs
 */

interface Options {
  readonly httpUrl: string
  readonly networkId: string
  readonly feeToken: string
  readonly bech32prefix: string
  readonly hdPath: readonly Slip10RawIndex[]
  readonly faucetUrl?: string
  readonly defaultKeyFile: string,
  readonly fees: {
    upload: StdFee,
    init: StdFee,
    exec: StdFee
  }
}

const pebblenetGasPrice = GasPrice.fromString("0.01upebble");
const pebblenetOptions: Options = {
  httpUrl: 'https://rpc.pebblenet.cosmwasm.com',
  networkId: 'pebblenet-1',
  bech32prefix: 'wasm',
  feeToken: 'upebble',
  faucetUrl: 'https://faucet.pebblenet.cosmwasm.com/credit',
  hdPath: makeCosmoshubPath(0),
  defaultKeyFile: path.join(process.env.HOME, ".pebblenet.key"),
  fees: {
    upload: calculateFee(1500000, pebblenetGasPrice),
    init: calculateFee(500000, pebblenetGasPrice),
    exec: calculateFee(200000, pebblenetGasPrice),
  },
}

interface Network {
  setup: (password: string, filename?: string) => Promise<[string, SigningCosmWasmClient]>
  recoverMnemonic: (password: string, filename?: string) => Promise<string>
}

const useOptions = (options: Options): Network => {

  const loadOrCreateWallet = async (options: Options, filename: string, password: string): Promise<DirectSecp256k1HdWallet> => {
    let encrypted: string;
    try {
      encrypted = fs.readFileSync(filename, 'utf8');
    } catch (err) {
      // generate if no file exists
      const wallet = await DirectSecp256k1HdWallet.generate(12, {hdPaths: [options.hdPath], prefix: options.bech32prefix});
      const encrypted = await wallet.serialize(password);
      fs.writeFileSync(filename, encrypted, 'utf8');
      return wallet;
    }
    // otherwise, decrypt the file (we cannot put deserialize inside try or it will over-write on a bad password)
    const wallet = await DirectSecp256k1HdWallet.deserialize(encrypted, password);
    return wallet;
  };

  const connect = async (
    wallet: DirectSecp256k1HdWallet,
    options: Options
  ): Promise<SigningCosmWasmClient> => {
    const clientOptions = {
      prefix: options.bech32prefix
    }
    return await SigningCosmWasmClient.connectWithSigner(options.httpUrl, wallet, clientOptions)
  };

  const hitFaucet = async (
    faucetUrl: string,
    address: string,
    denom: string
  ): Promise<void> => {
    await axios.post(faucetUrl, { denom, address });
  }

  const setup = async (password: string, filename?: string): Promise<[string, SigningCosmWasmClient]> => {
    const keyfile = filename || options.defaultKeyFile;
    const wallet = await loadOrCreateWallet(pebblenetOptions, keyfile, password);
    const client = await connect(wallet, pebblenetOptions);

    const [account] = await wallet.getAccounts();
    // ensure we have some tokens
    if (options.faucetUrl) {
      const tokens = await client.getBalance(account.address, options.feeToken)
      if (tokens.amount === '0') {
        console.log(`Getting ${options.feeToken} from faucet`);
        await hitFaucet(options.faucetUrl, account.address, options.feeToken);
      }
    }

    return [account.address, client];
  }

  const recoverMnemonic = async (password: string, filename?: string): Promise<string> => {
    const keyfile = filename || options.defaultKeyFile;
    const wallet = await loadOrCreateWallet(pebblenetOptions, keyfile, password);
    return wallet.mnemonic;
  }

  return { setup, recoverMnemonic };
}

type TokenId = string

interface Balances {
  readonly address: string
  readonly amount: string  // decimal as string
}

interface MintInfo {
  readonly minter: string
  readonly cap?: string // decimal as string
}

interface ContractInfo {
  readonly name: string
  readonly symbol: string
}

interface NftInfo {
  readonly name: string,
  readonly description: string,
  readonly image: any
}

interface Access {
  readonly owner: string,
  readonly approvals: []
}
interface AllNftInfo {
  readonly access: Access,
  readonly info: NftInfo
}

interface Operators {
  readonly operators: []
}

interface Count {
  readonly count: number
}

interface InitMsg {
  readonly name: string
  readonly symbol: string
  readonly minter: string
}
// Better to use this interface?
interface MintMsg {
  readonly token_id: TokenId
  readonly owner: string
  readonly name: string
  readonly description?: string
  readonly image?: string
}

type Expiration = { readonly at_height: number } | { readonly at_time: number } | { readonly never: {} };

interface AllowanceResponse {
  readonly allowance: string;  // integer as string
  readonly expires: Expiration;
}

interface AllowanceInfo {
  readonly allowance: string;  // integer as string
  readonly spender: string; // bech32 address
  readonly expires: Expiration;
}

interface AllAllowancesResponse {
  readonly allowances: readonly AllowanceInfo[];
}

interface AllAccountsResponse {
  // list of bech32 address that have a balance
  readonly accounts: readonly string[];
}

interface TokensResponse {
  readonly tokens: readonly string[];
}

interface CW721Instance {
  readonly contractAddress: string

  // queries
  allowance: (owner: string, spender: string) => Promise<AllowanceResponse>
  allAllowances: (owner: string, startAfter?: string, limit?: number) => Promise<AllAllowancesResponse>
  allAccounts: (startAfter?: string, limit?: number) => Promise<readonly string[]>
  minter: () => Promise<MintInfo>
  contractInfo: () => Promise<ContractInfo>
  nftInfo: (tokenId: TokenId) => Promise<NftInfo>
  allNftInfo: (tokenId: TokenId) => Promise<AllNftInfo>
  ownerOf: (tokenId: TokenId) => Promise<Access>
  approvedForAll: (owner: string, include_expired?: boolean, start_after?: string, limit?: number) => Promise<Operators>
  numTokens: () => Promise<Count>
  tokens: (owner: string, startAfter?: string, limit?: number) => Promise<TokensResponse>
  allTokens: (startAfter?: string, limit?: number) => Promise<TokensResponse>

  // actions
  mint: (senderAddress: string, tokenId: TokenId, owner: string, name: string, level: number, description?: string, image?: string) => Promise<string>
  transferNft: (senderAddress: string, recipient: string, tokenId: TokenId) => Promise<string>
  sendNft: (senderAddress: string, contract: string, token_id: TokenId, msg?: BinaryType) => Promise<string>
  approve: (senderAddress: string, spender: string, tokenId: TokenId, expires?: Expiration) => Promise<string>
  approveAll: (senderAddress: string, operator: string, expires?: Expiration) => Promise<string>
  revoke: (senderAddress: string, spender: string, tokenId: TokenId) => Promise<string>
  revokeAll: (senderAddress: string, operator: string) => Promise<string>
}

interface CW721Contract {
  // upload a code blob and returns a codeId
  upload: (senderAddress: string) => Promise<number>

  // instantiates a cw721 contract
  // codeId must come from a previous deploy
  // label is the public name of the contract in listing
  // if you set admin, you can run migrations on this contract (likely client.senderAddress)
  instantiate: (senderAddress: string, codeId: number, initMsg: Record<string, unknown>, label: string, admin?: string) => Promise<CW721Instance>

  use: (contractAddress: string) => CW721Instance
}


export const CW721 = (client: SigningCosmWasmClient, fees: Options['fees']): CW721Contract => {
  const use = (contractAddress: string): CW721Instance => {
    
    const allowance = async (owner: string, spender: string): Promise<AllowanceResponse> => {
      return client.queryContractSmart(contractAddress, { allowance: { owner, spender } });
    };

    const allAllowances = async (owner: string, startAfter?: string, limit?: number): Promise<AllAllowancesResponse> => {
      return client.queryContractSmart(contractAddress, { all_allowances: { owner, start_after: startAfter, limit } });
    };

    const allAccounts = async (startAfter?: string, limit?: number): Promise<readonly string[]> => {
      const accounts: AllAccountsResponse = await client.queryContractSmart(contractAddress, { all_accounts: { start_after: startAfter, limit } });
      return accounts.accounts;
    };

    const minter = async (): Promise<MintInfo> => {
      return client.queryContractSmart(contractAddress, { minter: {} });
    };

    const contractInfo = async (): Promise<ContractInfo> => {
      return client.queryContractSmart(contractAddress, { contract_info: {} });
    };

    const nftInfo = async (token_id: TokenId): Promise<NftInfo> => {
      return client.queryContractSmart(contractAddress, { nft_info: { token_id } });
    }

    const allNftInfo = async (token_id: TokenId): Promise<AllNftInfo> => {
      return client.queryContractSmart(contractAddress, { all_nft_info: { token_id } });
    }

    const ownerOf = async (token_id: TokenId): Promise<Access> => {
      return await client.queryContractSmart(contractAddress, { owner_of: { token_id } });
    }

    const approvedForAll = async (owner: string, include_expired?: boolean, start_after?: string, limit?: number): Promise<Operators> => {
      return await client.queryContractSmart(contractAddress, { approved_for_all: { owner, include_expired, start_after, limit } })
    }

    // total number of tokens issued
    const numTokens = async (): Promise<Count> => {
      return client.queryContractSmart(contractAddress, { num_tokens: {} });
    }

    // list all token_ids that belong to a given owner
    const tokens = async (owner: string, start_after?: string, limit?: number): Promise<TokensResponse> => {
      return client.queryContractSmart(contractAddress, { tokens: { owner, start_after, limit } });
    }

    const allTokens = async (start_after?: string, limit?: number): Promise<TokensResponse> => {
      return client.queryContractSmart(contractAddress, { all_tokens: { start_after, limit } });
    }

    // actions 
    const mint = async (senderAddress: string, token_id: TokenId, owner: string, name: string, level: number, description?: string, image?: string): Promise<string> => {
      const result = await client.execute(senderAddress, contractAddress, { mint: { token_id, owner, name, level, description, image } }, fees.exec);
      return result.transactionHash;
    }

    // transfers ownership, returns transactionHash
    const transferNft = async (senderAddress: string, recipient: string, token_id: TokenId): Promise<string> => {
      const result = await client.execute(senderAddress, contractAddress, { transfer_nft: { recipient, token_id } }, fees.exec);
      return result.transactionHash;
    }

    // sends an nft token to another contract (TODO: msg type any needs to be revisited once receiveNft is implemented)
    const sendNft = async (senderAddress: string, contract: string, token_id: TokenId, msg?: any): Promise<string> => {
      const result = await client.execute(senderAddress, contractAddress, { send_nft: { contract, token_id, msg } }, fees.exec)
      return result.transactionHash;
    }

    const approve = async (senderAddress: string, spender: string, token_id: TokenId, expires?: Expiration): Promise<string> => {
      const result = await client.execute(senderAddress, contractAddress, { approve: { spender, token_id, expires } }, fees.exec);
      return result.transactionHash;
    }

    const approveAll = async (senderAddress: string, operator: string, expires?: Expiration): Promise<string> => {
      const result = await client.execute(senderAddress, contractAddress, { approve_all: { operator, expires } }, fees.exec)
      return result.transactionHash
    }

    const revoke = async (senderAddress: string, spender: string, token_id: TokenId): Promise<string> => {
      const result = await client.execute(senderAddress, contractAddress, { revoke: { spender, token_id } }, fees.exec);
      return result.transactionHash;
    }

    const revokeAll = async (senderAddress: string, operator: string): Promise<string> => {
      const result = await client.execute(senderAddress, contractAddress, { revoke_all: { operator } }, fees.exec)
      return result.transactionHash;
    }

    return {
      contractAddress,
      allowance,
      allAllowances,
      allAccounts,
      minter,
      contractInfo,
      nftInfo,
      allNftInfo,
      ownerOf,
      approvedForAll,
      numTokens,
      tokens,
      allTokens,
      mint,
      transferNft,
      sendNft,
      approve,
      approveAll,
      revoke,
      revokeAll
    };
  }

  const downloadWasm = async (url: string): Promise<Uint8Array> => {
    const r = await axios.get(url, { responseType: 'arraybuffer' })
    if (r.status !== 200) {
      throw new Error(`Download error: ${r.status}`)
    }
    return r.data
  }

  const upload = async (senderAddress: string): Promise<number> => {
    const sourceUrl = "https://github.com/CosmWasm/cosmwasm-plus/releases/download/v0.9.0/cw721_base.wasm";
    const wasm = await downloadWasm(sourceUrl);
    const result = await client.upload(senderAddress, wasm, fees.upload);
    return result.codeId;
  }

  const instantiate = async (senderAddress: string, codeId: number, initMsg: Record<string, unknown>, label: string, admin?: string): Promise<CW721Instance> => {
    const result = await client.instantiate(senderAddress, codeId, initMsg, label, fees.init, { memo: `Init ${label}`, admin });
    return use(result.contractAddress);
  }

  return { upload, instantiate, use };
}
