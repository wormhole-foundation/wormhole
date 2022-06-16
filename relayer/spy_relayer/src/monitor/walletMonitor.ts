import {
  Bridge__factory,
  ChainId,
  CHAIN_ID_SOLANA,
  CHAIN_ID_TERRA,
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
import { getScopedLogger } from "../helpers/logHelper";
import { PromHelper } from "../helpers/promHelpers";
import { getMetaplexData, sleep } from "../helpers/utils";
import { getEthereumToken } from "../utils/ethereum";
import { getMultipleAccountsRPC } from "../utils/solana";
import { formatNativeDenom } from "../utils/terra";
import { newProvider } from "../relayer/evm";

let env: RelayerEnvironment;
const logger = getScopedLogger(["walletMonitor"]);

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

async function pullBalances(metrics: PromHelper): Promise<WalletBalance[]> {
  env = getRelayerEnvironment();
  //TODO loop through all the chain configs, calc the public keys, pull their balances, and push to a combo of the loggers and prmometheus
  if (!env) {
    logger.error("pullBalances() - no env");
    return [];
  }
  if (!env.supportedChains) {
    logger.error("pullBalances() - no supportedChains");
    return [];
  }
  const balancePromises: Promise<WalletBalance[]>[] = [];
  for (const chainInfo of env.supportedChains) {
    if (!chainInfo) continue;
    try {
      if (chainInfo.chainId === CHAIN_ID_SOLANA) {
        for (const solanaPrivateKey of chainInfo.solanaPrivateKey || []) {
          try {
            balancePromises.push(
              pullSolanaNativeBalance(chainInfo, solanaPrivateKey)
            );
            balancePromises.push(
              pullSolanaTokenBalances(chainInfo, solanaPrivateKey)
            );
          } catch (e: any) {
            logger.error(
              "pulling balances failed failed for chain: " + chainInfo.chainName
            );
            if (e && e.stack) {
              logger.error(e.stack);
            }
          }
        }
      } else if (isEVMChain(chainInfo.chainId)) {
        for (const privateKey of chainInfo.walletPrivateKey || []) {
          try {
            balancePromises.push(pullEVMNativeBalance(chainInfo, privateKey));
          } catch (e) {
            logger.error("pullEVMNativeBalance() failed: " + e);
          }
        }
        // TODO one day this will spin up independent watchers that time themselves
        // purposefully not awaited
        pullAllEVMTokens(env.supportedTokens, chainInfo, metrics);
      } else if (chainInfo.chainId === CHAIN_ID_TERRA) {
        // TODO one day this will spin up independent watchers that time themselves
        // purposefully not awaited
        pullAllTerraBalances(env.supportedTokens, chainInfo, metrics);
      } else {
        logger.error("Invalid chain ID in wallet monitor " + chainInfo.chainId);
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

  const balancesArrays = await Promise.all(balancePromises);
  const balances = balancesArrays.reduce(
    (prev, curr) => [...prev, ...curr],
    []
  );

  return balances;
}

export async function pullTerraBalance(
  lcd: LCDClient,
  walletAddress: string,
  tokenAddress: string
): Promise<WalletBalance | undefined> {
  try {
    const tokenInfo: any = await lcd.wasm.contractQuery(tokenAddress, {
      token_info: {},
    });
    const balanceInfo: any = await lcd.wasm.contractQuery(tokenAddress, {
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
  } catch (e) {
    logger.error("Failed to fetch terra balance for %s", tokenAddress);
  }
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
): Promise<WalletBalance[]> {
  if (!privateKey || !chainInfo.nodeUrl) {
    throw new Error("Bad chainInfo config for EVM chain: " + chainInfo.chainId);
  }

  let provider = newProvider(chainInfo.nodeUrl);
  if (!provider) throw new Error("bad provider");
  const signer: Signer = new ethers.Wallet(privateKey, provider);
  const addr: string = await signer.getAddress();
  const weiAmount = await provider.getBalance(addr);
  const balanceInEth = ethers.utils.formatEther(weiAmount);

  return [
    {
      chainId: chainInfo.chainId,
      balanceAbs: weiAmount.toString(),
      balanceFormatted: balanceInEth.toString(),
      currencyName: chainInfo.nativeCurrencySymbol,
      currencyAddressNative: "",
      isNative: true,
      walletAddress: addr,
    },
  ];
}

async function pullTerraNativeBalance(
  lcd: LCDClient,
  chainInfo: ChainConfigInfo,
  walletAddress: string
): Promise<WalletBalance[]> {
  try {
    const output: WalletBalance[] = [];
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
  } catch (e) {
    logger.error(
      "Failed to fetch terra native balances for wallet %s",
      walletAddress
    );
    return [];
  }
}

async function pullSolanaNativeBalance(
  chainInfo: ChainConfigInfo,
  privateKey: Uint8Array
): Promise<WalletBalance[]> {
  const keyPair = Keypair.fromSecretKey(privateKey);
  const connection = new Connection(chainInfo.nodeUrl);
  const fetchAccounts = await getMultipleAccountsRPC(connection, [
    keyPair.publicKey,
  ]);

  if (!fetchAccounts[0]) {
    //Accounts with zero balance report as not existing.
    return [
      {
        chainId: chainInfo.chainId,
        balanceAbs: "0",
        balanceFormatted: "0",
        currencyName: chainInfo.nativeCurrencySymbol,
        currencyAddressNative: chainInfo.chainName,
        isNative: true,
        walletAddress: keyPair.publicKey.toString(),
      },
    ];
  }

  const amountLamports = fetchAccounts[0].lamports.toString();
  const amountSol = formatUnits(
    fetchAccounts[0].lamports,
    WSOL_DECIMALS
  ).toString();

  return [
    {
      chainId: chainInfo.chainId,
      balanceAbs: amountLamports,
      balanceFormatted: amountSol,
      currencyName: chainInfo.nativeCurrencySymbol,
      currencyAddressNative: "",
      isNative: true,
      walletAddress: keyPair.publicKey.toString(),
    },
  ];
}

export async function collectWallets(metrics: PromHelper) {
  const scopedLogger = getScopedLogger(["collectWallets"], logger);
  const ONE_MINUTE: number = 60000;
  scopedLogger.info("Starting up.");
  while (true) {
    scopedLogger.debug("Pulling balances.");
    let wallets: WalletBalance[] = [];
    try {
      wallets = await pullBalances(metrics);
    } catch (e) {
      scopedLogger.error("Failed to pullBalances: " + e);
    }
    scopedLogger.debug("Done pulling balances.");
    metrics.handleWalletBalances(wallets);
    await sleep(ONE_MINUTE);
  }
}

async function calcLocalAddressesEVM(
  provider: ethers.providers.JsonRpcBatchProvider,
  supportedTokens: SupportedToken[],
  chainConfigInfo: ChainConfigInfo
): Promise<string[]> {
  const tokenBridge = Bridge__factory.connect(
    chainConfigInfo.tokenBridgeAddress,
    provider
  );
  let tokenAddressPromises: Promise<string>[] = [];
  for (const supportedToken of supportedTokens) {
    if (supportedToken.chainId === chainConfigInfo.chainId) {
      tokenAddressPromises.push(Promise.resolve(supportedToken.address));
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
    tokenAddressPromises.push(
      tokenBridge.wrappedAsset(
        supportedToken.chainId,
        hexToUint8Array(hexAddress)
      )
    );
  }
  return (await Promise.all(tokenAddressPromises)).filter(
    (tokenAddress) =>
      tokenAddress && tokenAddress !== ethers.constants.AddressZero
  );
}

export async function calcLocalAddressesTerra(
  lcd: LCDClient,
  supportedTokens: SupportedToken[],
  chainConfigInfo: ChainConfigInfo
) {
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
  chainConfig: ChainConfigInfo,
  metrics: PromHelper
) {
  try {
    let provider = newProvider(
      chainConfig.nodeUrl,
      true
    ) as ethers.providers.JsonRpcBatchProvider;
    const localAddresses = await calcLocalAddressesEVM(
      provider,
      supportedTokens,
      chainConfig
    );
    if (!chainConfig.walletPrivateKey) {
      return;
    }
    for (const privateKey of chainConfig.walletPrivateKey) {
      try {
        const publicAddress = await new ethers.Wallet(privateKey).getAddress();
        const tokens = await Promise.all(
          localAddresses.map((tokenAddress) =>
            getEthereumToken(tokenAddress, provider)
          )
        );
        const tokenInfos = await Promise.all(
          tokens.map((token) =>
            Promise.all([
              token.decimals(),
              token.balanceOf(publicAddress),
              token.symbol(),
            ])
          )
        );
        const balances = tokenInfos.map(([decimals, balance, symbol], idx) => ({
          chainId: chainConfig.chainId,
          balanceAbs: balance.toString(),
          balanceFormatted: formatUnits(balance, decimals),
          currencyName: symbol,
          currencyAddressNative: localAddresses[idx],
          isNative: false,
          walletAddress: publicAddress,
        }));
        metrics.handleWalletBalances(balances);
      } catch (e) {
        logger.error(
          "pullAllEVMTokens failed: for tokens " +
            JSON.stringify(localAddresses) +
            " on chain " +
            chainConfig.chainId +
            ", error: " +
            e
        );
      }
    }
  } catch (e) {
    logger.error(
      "pullAllEVMTokens failed: for chain " +
        chainConfig.chainId +
        ", error: " +
        e
    );
  }
}

async function pullAllTerraBalances(
  supportedTokens: SupportedToken[],
  chainConfig: ChainConfigInfo,
  metrics: PromHelper
) {
  let balances: WalletBalance[] = [];
  if (!chainConfig.walletPrivateKey) {
    return balances;
  }
  if (
    !(
      chainConfig.terraChainId &&
      chainConfig.terraCoin &&
      chainConfig.terraGasPriceUrl &&
      chainConfig.terraName
    )
  ) {
    logger.error("Terra relay was called without proper instantiation.");
    throw new Error("Terra relay was called without proper instantiation.");
  }
  const lcdConfig = {
    URL: chainConfig.nodeUrl,
    chainID: chainConfig.terraChainId,
    name: chainConfig.terraName,
  };
  const lcd = new LCDClient(lcdConfig);
  const localAddresses = await calcLocalAddressesTerra(
    lcd,
    supportedTokens,
    chainConfig
  );
  for (const privateKey of chainConfig.walletPrivateKey) {
    const mk = new MnemonicKey({
      mnemonic: privateKey,
    });
    const wallet = lcd.wallet(mk);
    const walletAddress = wallet.key.accAddress;
    balances = [
      ...balances,
      ...(await pullTerraNativeBalance(lcd, chainConfig, walletAddress)),
    ];
    for (const address of localAddresses) {
      const balance = await pullTerraBalance(lcd, walletAddress, address);
      if (balance) {
        balances.push(balance);
      }
    }
  }

  metrics.handleWalletBalances(balances);
}
