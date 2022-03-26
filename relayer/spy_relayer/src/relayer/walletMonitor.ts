import {
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
  getForeignAssetEth,
  getForeignAssetTerra,
  hexToUint8Array,
  isEVMChain,
  nativeToHexString,
  WSOL_DECIMALS,
} from "@certusone/wormhole-sdk";
import { TOKEN_PROGRAM_ID } from "@solana/spl-token";
import { Connection, Keypair } from "@solana/web3.js";
import { LCDClient, MnemonicKey } from "@terra-money/terra.js";
import { ethers, Signer } from "ethers";
import { formatUnits } from "ethers/lib/utils";
import {
  ChainConfigInfo,
  getRelayerEnvironment,
  RelayerEnvironment,
  SupportedToken,
} from "../configureEnv";
import { getLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";
import { getMetaplexData, sleep } from "../helpers/utils";
import { getEthereumToken } from "../utils/ethereum";
import { getMultipleAccountsRPC } from "../utils/solana";
import { formatNativeDenom } from "../utils/terra";
import { newProvider } from "./evm";

let env: RelayerEnvironment;
const logger = getLogger();

export type WalletBalance = {
  chainId: ChainId;
  balanceAbs: string;
  balanceFormatted?: string;
  currencyName: string;
  currencyAddressNative: string;
  isNative: boolean;
  walletAddress: string;
};

export interface TerraNativeBalances {
  [index: string]: string;
}

function init() {
  try {
    env = getRelayerEnvironment();
  } catch (e) {
    logger.error("Unable to instantiate the relayerEnv in wallet monitor");
  }
}

async function pullBalances(): Promise<WalletBalance[]> {
  //TODO loop through all the chain configs, calc the public keys, pull their balances, and push to a combo of the loggers and prmometheus

  let balances: WalletBalance[] = [];

  logger.debug("pulling balances...");
  if (!env) {
    logger.error("pullBalances() - no env");
    return balances;
  }
  if (!env.supportedChains) {
    logger.error("pullBalances() - no supportedChains");
    return balances;
  }
  for (const chainInfo of env.supportedChains) {
    if (!chainInfo) break;
    for (const privateKey of chainInfo.walletPrivateKey || []) {
      try {
        if (!privateKey) break;
        logger.debug(
          "Attempting to pull native balance for chainId: " + chainInfo.chainId
        );
        if (isEVMChain(chainInfo.chainId)) {
          logger.info("Attempting to pull EVM native balance...");
          try {
            balances.push(await pullEVMNativeBalance(chainInfo, privateKey));
          } catch (e) {
            logger.error("pullEVMNativeBalance() failed: " + e);
          }
          logger.info("Attempting to pull EVM non-native balance...");
          balances = balances.concat(
            await pullAllEVMTokens(env.supportedTokens, chainInfo)
          );
        } else if (chainInfo.chainId === CHAIN_ID_TERRA) {
          logger.info("Attempting to pull TERRA native balance...");
          balances = balances.concat(
            await pullTerraNativeBalance(chainInfo, privateKey)
          );
          logger.info("Attempting to pull TERRA non-native balance...");
          balances = balances.concat(
            await pullAllTerraTokens(env.supportedTokens, chainInfo)
          );
        } else {
          logger.error(
            "Invalid chain ID in wallet monitor " + chainInfo.chainId
          );
        }
      } catch (e: any) {
        logger.error(
          "pulling balances failed failed for chain: " + chainInfo.chainName
        );
        if (e && e.stack) {
          logger.error(e.stack);
        }
      }
    }

    for (const solanaPrivateKey of chainInfo.solanaPrivateKey || []) {
      try {
        if (chainInfo.chainId === CHAIN_ID_SOLANA) {
          logger.info("pullBalances() - calling pullSolanaNativeBalance...");
          balances.push(
            await pullSolanaNativeBalance(chainInfo, solanaPrivateKey)
          );
          logger.info("pullBalances() - calling pullSolanaTokenBalances...");
          balances = balances.concat(
            await pullSolanaTokenBalances(chainInfo, solanaPrivateKey)
          );
        }
      } catch (e: any) {
        logger.error(
          "pulling balances failed failed for chain: " + chainInfo.chainName
        );
        if (e && e.stack) {
          logger.error(e.stack);
        }
      }
    }
  }

  // logger.debug("returning balances:  %o", balances);
  return balances;
}

export async function pullEVMBalance(
  chainInfo: ChainConfigInfo,
  publicAddress: string,
  tokenAddress: string
): Promise<WalletBalance> {
  let provider = newProvider(chainInfo.nodeUrl);

  const token = await getEthereumToken(tokenAddress, provider);
  const decimals = await token.decimals();
  const balance = await token.balanceOf(publicAddress);
  const symbol = await token.symbol();
  const balanceFormatted = formatUnits(balance, decimals);

  if (provider instanceof ethers.providers.WebSocketProvider) {
    await provider.destroy();
  }

  return {
    chainId: chainInfo.chainId,
    balanceAbs: balance.toString(),
    balanceFormatted: balanceFormatted,
    currencyName: symbol,
    currencyAddressNative: tokenAddress,
    isNative: false,
    walletAddress: publicAddress,
  };
}

async function pullTerraBalance(
  chainInfo: ChainConfigInfo,
  walletPrivateKey: string,
  tokenAddress: string
): Promise<WalletBalance | undefined> {
  if (
    !(
      chainInfo.terraChainId &&
      chainInfo.terraCoin &&
      chainInfo.terraGasPriceUrl &&
      chainInfo.terraName
    )
  ) {
    logger.error("Terra relay was called without proper instantiation.");
    throw new Error("Terra relay was called without proper instantiation.");
  }
  const lcdConfig = {
    URL: chainInfo.nodeUrl,
    chainID: chainInfo.terraChainId,
    name: chainInfo.terraName,
  };
  const lcd = new LCDClient(lcdConfig);
  const mk = new MnemonicKey({
    mnemonic: walletPrivateKey,
  });
  const wallet = lcd.wallet(mk);
  const walletAddress = wallet.key.accAddress;

  const tokenInfo: any = await lcd.wasm.contractQuery(tokenAddress, {
    token_info: {},
  });
  const balanceInfo: any = lcd.wasm.contractQuery(tokenAddress, {
    balance: {
      address: walletAddress,
    },
  });

  if (!tokenInfo || !balanceInfo) {
    return undefined;
  }

  return {
    chainId: CHAIN_ID_TERRA,
    balanceAbs: balanceInfo?.balance?.toString() || "0",
    balanceFormatted: formatUnits(
      balanceInfo?.balance?.toString() || "0",
      tokenInfo.decimals
    ),
    currencyName: tokenInfo.symbol,
    currencyAddressNative: tokenAddress,
    isNative: false,
    walletAddress: walletAddress,
  };
}

async function pullSolanaTokenBalances(
  chainInfo: ChainConfigInfo,
  privateKey: Uint8Array
): Promise<WalletBalance[]> {
  const keyPair = Keypair.fromSecretKey(privateKey);
  const connection = new Connection(chainInfo.nodeUrl);
  const output: WalletBalance[] = [];

  try {
    const allAccounts = await connection.getParsedTokenAccountsByOwner(
      keyPair.publicKey,
      { programId: TOKEN_PROGRAM_ID },
      "confirmed"
    );
    let mintAddresses: string[] = [];
    allAccounts.value.forEach((account) => {
      mintAddresses.push(account.account.data.parsed?.info?.mint);
    });
    const mdArray = await getMetaplexData(mintAddresses, chainInfo);

    for (const account of allAccounts.value) {
      let mintAddress: string[] = [];
      mintAddress.push(account.account.data.parsed?.info?.mint);
      const mdArray = await getMetaplexData(mintAddress, chainInfo);
      let cName: string = "";
      if (mdArray && mdArray[0] && mdArray[0].data && mdArray[0].data.symbol) {
        const encoded = mdArray[0].data.symbol;
        cName = encodeURIComponent(encoded);
        cName = cName.replace(/%/g, "_");
      }

      output.push({
        chainId: CHAIN_ID_SOLANA,
        balanceAbs: account.account.data.parsed?.info?.tokenAmount?.amount,
        balanceFormatted:
          account.account.data.parsed?.info?.tokenAmount?.uiAmount,
        currencyName: cName,
        currencyAddressNative: account.account.data.parsed?.info?.mint,
        isNative: false,
        walletAddress: account.pubkey.toString(),
      });
    }
  } catch (e) {
    logger.error("pullSolanaTokenBalances() - ", e);
  }

  return output;
}

async function pullEVMNativeBalance(
  chainInfo: ChainConfigInfo,
  privateKey: string
): Promise<WalletBalance> {
  if (!privateKey || !chainInfo.nodeUrl) {
    throw new Error("Bad chainInfo config for EVM chain: " + chainInfo.chainId);
  }

  let provider = newProvider(chainInfo.nodeUrl);
  if (!provider) throw new Error("bad provider");
  const signer: Signer = new ethers.Wallet(privateKey, provider);
  const addr: string = await signer.getAddress();
  const weiAmount = await provider.getBalance(addr);
  const balanceInEth = ethers.utils.formatEther(weiAmount);
  if (provider instanceof ethers.providers.WebSocketProvider) {
    await provider.destroy();
  }

  return {
    chainId: chainInfo.chainId,
    balanceAbs: weiAmount.toString(),
    balanceFormatted: balanceInEth.toString(),
    currencyName: chainInfo.nativeCurrencySymbol,
    currencyAddressNative: "",
    isNative: true,
    walletAddress: addr,
  };
}

async function pullTerraNativeBalance(
  chainInfo: ChainConfigInfo,
  privateKey: string
): Promise<WalletBalance[]> {
  const output: WalletBalance[] = [];
  if (
    !(
      chainInfo.terraChainId &&
      chainInfo.terraCoin &&
      chainInfo.terraGasPriceUrl &&
      chainInfo.terraName
    )
  ) {
    logger.error(
      "Terra wallet balance was called without proper instantiation."
    );
    throw new Error(
      "Terra wallet balance was called without proper instantiation."
    );
  }
  const lcdConfig = {
    URL: chainInfo.nodeUrl,
    chainID: chainInfo.terraChainId,
    name: chainInfo.terraName,
  };
  const lcd = new LCDClient(lcdConfig);
  const mk = new MnemonicKey({
    mnemonic: privateKey,
  });
  const wallet = lcd.wallet(mk);
  const walletAddress = wallet.key.accAddress;

  const [coins] = await lcd.bank.balance(walletAddress);
  // coins doesn't support reduce
  const balancePairs = coins.map(({ amount, denom }) => [denom, amount]);
  const balance = balancePairs.reduce((obj, current) => {
    obj[current[0].toString()] = current[1].toString();
    return obj;
  }, {} as TerraNativeBalances);
  Object.keys(balance).forEach((key) => {
    output.push({
      chainId: chainInfo.chainId,
      balanceAbs: balance[key],
      balanceFormatted: formatUnits(balance[key], 6).toString(),
      currencyName: formatNativeDenom(key),
      currencyAddressNative: key,
      isNative: true,
      walletAddress: walletAddress,
    });
  });
  return output;
}

async function pullSolanaNativeBalance(
  chainInfo: ChainConfigInfo,
  privateKey: Uint8Array
): Promise<WalletBalance> {
  const keyPair = Keypair.fromSecretKey(privateKey);
  const connection = new Connection(chainInfo.nodeUrl);
  const fetchAccounts = await getMultipleAccountsRPC(connection, [
    keyPair.publicKey,
  ]);

  if (!fetchAccounts[0]) {
    //Accounts with zero balance report as not existing.
    return {
      chainId: chainInfo.chainId,
      balanceAbs: "0",
      balanceFormatted: "0",
      currencyName: chainInfo.nativeCurrencySymbol,
      currencyAddressNative: chainInfo.chainName,
      isNative: true,
      walletAddress: keyPair.publicKey.toString(),
    };
  }

  const amountLamports = fetchAccounts[0].lamports.toString();
  const amountSol = formatUnits(
    fetchAccounts[0].lamports,
    WSOL_DECIMALS
  ).toString();

  return {
    chainId: chainInfo.chainId,
    balanceAbs: amountLamports,
    balanceFormatted: amountSol,
    currencyName: chainInfo.nativeCurrencySymbol,
    currencyAddressNative: "",
    isNative: true,
    walletAddress: keyPair.publicKey.toString(),
  };
}

export async function collectWallets(metrics: PromHelper) {
  const ONE_MINUTE: number = 60000;
  logger.info("collectWallets() - starting up...");
  init();
  while (true) {
    // get wallet amounts
    logger.debug("collectWallets() - pulling balances...");
    let wallets: WalletBalance[] = [];
    try {
      wallets = await pullBalances();
    } catch (e) {
      logger.error("Failed to pullBalances: " + e);
    }
    logger.debug("collectWallets() - done pulling balances...");
    // peg prometheus metrics
    // logger.debug("collectWallets() - Destined for Prometheus: %o", wallets);
    metrics.handleWalletBalances(wallets);
    logger.debug("collectWallets() - Finished metrics call.");
    await sleep(ONE_MINUTE);
  }
}

async function calcLocalAddressesEVM(
  supportedTokens: SupportedToken[],
  chainConfigInfo: ChainConfigInfo
): Promise<string[]> {
  let provider = newProvider(chainConfigInfo.nodeUrl);

  // logger.debug("calcLocalAddressesEVM() - entered.");
  let output: string[] = [];
  for (const supportedToken of supportedTokens) {
    if (supportedToken.chainId === chainConfigInfo.chainId) {
      output.push(supportedToken.address);
      continue;
    }
    const hexAddress = nativeToHexString(
      supportedToken.address,
      supportedToken.chainId
    );
    if (!hexAddress) {
      logger.debug(
        "calcLocalAddressesEVM() - no hexAddress for chainId: " +
          supportedToken.chainId +
          ", address: " +
          supportedToken.address
      );
      continue;
    }
    // logger.debug("calcLocalAddressesEVM() - got hex address: " + hexAddress);
    //This returns a native address
    let foreignAddress;
    try {
      foreignAddress = await getForeignAssetEth(
        chainConfigInfo.tokenBridgeAddress,
        provider as any, //why does this typecheck work elsewhere?
        supportedToken.chainId,
        hexToUint8Array(hexAddress)
      );
    } catch (e) {
      logger.error("Exception thrown from getForeignAssetEth");
    }

    if (!foreignAddress || foreignAddress === ethers.constants.AddressZero) {
      continue;
    }
    output.push(foreignAddress);
  }

  if (provider instanceof ethers.providers.WebSocketProvider) {
    await provider.destroy();
  }
  return output;
}

async function calcLocalAddressesTerra(
  supportedTokens: SupportedToken[],
  chainConfigInfo: ChainConfigInfo
) {
  if (
    !(
      chainConfigInfo.terraChainId &&
      chainConfigInfo.terraCoin &&
      chainConfigInfo.terraGasPriceUrl &&
      chainConfigInfo.terraName
    )
  ) {
    logger.error(
      "Terra wallet balance was called without proper instantiation."
    );
    throw new Error(
      "Terra wallet balance was called without proper instantiation."
    );
  }
  const lcdConfig = {
    URL: chainConfigInfo.nodeUrl,
    chainID: chainConfigInfo.terraChainId,
    name: chainConfigInfo.terraName,
  };
  const lcd = new LCDClient(lcdConfig);

  const output: string[] = [];
  for (const supportedToken of supportedTokens) {
    if (supportedToken.chainId === chainConfigInfo.chainId) {
      // skip natives, like uluna and uusd
      if (supportedToken.address.startsWith("terra")) {
        output.push(supportedToken.address);
      }
      continue;
    }
    const hexAddress = nativeToHexString(
      supportedToken.address,
      supportedToken.chainId
    );
    if (!hexAddress) {
      continue;
    }
    //This returns a native address
    let foreignAddress;
    try {
      foreignAddress = await getForeignAssetTerra(
        chainConfigInfo.tokenBridgeAddress,
        lcd,
        supportedToken.chainId,
        hexToUint8Array(hexAddress)
      );
    } catch (e) {
      logger.error("Foreign address exception.");
    }

    if (!foreignAddress) {
      continue;
    }
    output.push(foreignAddress);
  }

  return output;
}

async function pullAllEVMTokens(
  supportedTokens: SupportedToken[],
  chainConfig: ChainConfigInfo
) {
  const localAddresses = await calcLocalAddressesEVM(
    supportedTokens,
    chainConfig
  );
  const output: WalletBalance[] = [];
  if (!chainConfig.walletPrivateKey) {
    return output;
  }
  for (const privateKey of chainConfig.walletPrivateKey) {
    const publicAddress = await new ethers.Wallet(privateKey).getAddress();
    for (const address of localAddresses) {
      try {
        const balance = await pullEVMBalance(
          chainConfig,
          publicAddress,
          address
        );
        if (balance) {
          output.push(balance);
        }
      } catch (e) {
        logger.error(
          "pullEVMBalance failed: for token " +
            address +
            " on chain " +
            chainConfig.chainId +
            ", error: " +
            e
        );
      }
    }
  }

  return output;
}

async function pullAllTerraTokens(
  supportedTokens: SupportedToken[],
  chainConfig: ChainConfigInfo
) {
  const localAddresses = await calcLocalAddressesTerra(
    supportedTokens,
    chainConfig
  );
  const output: WalletBalance[] = [];
  if (!chainConfig.walletPrivateKey) {
    return output;
  }
  for (const privateKey of chainConfig.walletPrivateKey) {
    for (const address of localAddresses) {
      const balance = await pullTerraBalance(chainConfig, privateKey, address);
      if (balance) {
        output.push(balance);
      }
    }
  }
  // logger.debug("pullAllTerraTokens() - returning %o", output);

  return output;
}
