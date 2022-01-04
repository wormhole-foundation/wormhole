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

let env: RelayerEnvironment;
const logger = getLogger();

export type WalletBalance = {
  chainId: ChainId;
  balanceAbs: string;
  balanceFormatted?: string;
  currencyName: string;
  currencyAddressNative: string;
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
  try {
    for (const chainInfo of env.supportedChains) {
      if (!chainInfo) break;
      for (const privateKey of chainInfo.walletPrivateKey || []) {
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
      }

      for (const solanaPrivateKey of chainInfo.solanaPrivateKey || []) {
        if (chainInfo.chainId === CHAIN_ID_SOLANA) {
          logger.info("pullBalances() - calling pullSolanaNativeBalance...");
          balances.push(
            await pullSolanaNativeBalance(chainInfo, solanaPrivateKey)
          );
          logger.info("pullBalances() - calling pullSolanaTokanBalances...");
          balances = balances.concat(
            await pullSolanaTokenBalances(chainInfo, solanaPrivateKey)
          );
        }
      }
    }
  } catch (e) {
    logger.error("pullBalance() - for loop failed: " + e);
  }
  // logger.debug("returning balances:  %o", balances);
  return balances;
}

async function pullEVMBalance(
  chainInfo: ChainConfigInfo,
  privateKey: string,
  tokenAddress: string
): Promise<WalletBalance> {
  if (parseInt(tokenAddress) === 0) {
    throw new Error("tokenAddress is 0");
  }
  let provider = new ethers.providers.WebSocketProvider(chainInfo.nodeUrl);
  const signer: Signer = new ethers.Wallet(privateKey, provider);

  logger.debug("About to get token for address: " + tokenAddress);
  const token = await getEthereumToken(tokenAddress, provider);
  // logger.debug("About to get decimals...");
  const decimals = await token.decimals();
  // logger.debug("About to get balance...");
  const balance = await token.balanceOf(await signer.getAddress());
  // logger.debug("About to get symbol...");
  const symbol = await token.symbol();
  //const name = await token.name();
  const balanceFormatted = formatUnits(balance, decimals);

  return {
    chainId: chainInfo.chainId,
    balanceAbs: balance.toString(),
    balanceFormatted: balanceFormatted,
    currencyName: symbol,
    currencyAddressNative: tokenAddress,
  };
}

async function pullTerraBalance(
  chainInfo: ChainConfigInfo,
  walletPrivateKey: string,
  tokenAddress: string
): Promise<WalletBalance> {
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

  return {
    chainId: CHAIN_ID_TERRA,
    balanceAbs: balanceInfo.balance.toString(),
    balanceFormatted: formatUnits(
      balanceInfo.balance.toString(),
      tokenInfo.decimals
    ),
    currencyName: tokenInfo.symbol,
    currencyAddressNative: tokenAddress,
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
    logger.debug(
      "pullSolanaTokenBalances() - mdArray.size = " + mdArray.length
    );
    mdArray.forEach((el) => {
      if (el && el.data && el?.data.symbol) {
        logger.debug(
          "pullSolanaTokenBalances() - el.data.symbol: " + el.data.symbol
        );
      }
    });

    for (const account of allAccounts.value) {
      let mintAddress: string[] = [];
      mintAddress.push(account.account.data.parsed?.info?.mint);
      const mdArray = await getMetaplexData(mintAddress, chainInfo);
      logger.debug(
        "pullSolanaTokenBalances() - mdArray.size = " + mdArray.length
      );
      let cName: string = "";
      if (mdArray && mdArray[0] && mdArray[0].data && mdArray[0].data.symbol) {
        const encoded = mdArray[0].data.symbol;
        cName = encodeURIComponent(encoded);
        cName = cName.replace(/%/g, "_");
        logger.debug(
          "pullSolanaTokenBalances() - encoded: " +
            encoded +
            ", cName: " +
            cName
        );
      }

      logger.debug("pullSolanaTokenBalances() - pushing output...");
      output.push({
        chainId: CHAIN_ID_SOLANA,
        balanceAbs: account.account.data.parsed?.info?.tokenAmount?.amount,
        balanceFormatted:
          account.account.data.parsed?.info?.tokenAmount?.uiAmount,
        currencyName: cName,
        currencyAddressNative: account.account.data.parsed?.info?.mint,
      });
    }
  } catch (e) {
    logger.error("pullSolanaTokenBalances() - ", e);
  }
  logger.debug("pullSolanaTokenBalances() - output: %o", output);

  return output;
}

async function pullEVMNativeBalance(
  chainInfo: ChainConfigInfo,
  privateKey: string
): Promise<WalletBalance> {
  if (!privateKey || !chainInfo.nodeUrl) {
    throw new Error("Bad chainInfo config for EVM chain: " + chainInfo.chainId);
  }

  let provider = await new ethers.providers.WebSocketProvider(
    chainInfo.nodeUrl
  );
  if (!provider) throw new Error("bad provider");
  const signer: Signer = new ethers.Wallet(privateKey, provider);
  const addr: string = await signer.getAddress();
  const weiAmount = await provider.getBalance(addr);
  const balanceInEth = ethers.utils.formatEther(weiAmount);
  await provider.destroy();

  logger.debug(
    "chainId: " +
      chainInfo.chainId +
      ", balanceAbs: " +
      weiAmount.toString() +
      ", balanceFormatted: " +
      balanceInEth.toString() +
      ", currencyName: " +
      chainInfo.chainName
  );
  return {
    chainId: chainInfo.chainId,
    balanceAbs: weiAmount.toString(),
    balanceFormatted: balanceInEth.toString(),
    currencyName: chainInfo.chainName,
    currencyAddressNative: chainInfo.chainName,
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
  // logger.debug("Terra wallet address: " + walletAddress);

  await lcd.bank.balance(walletAddress).then((coins) => {
    // coins doesn't support reduce
    const balancePairs = coins.map(({ amount, denom }) => [denom, amount]);
    const balance = balancePairs.reduce((obj, current) => {
      obj[current[0].toString()] = current[1].toString();
      // logger.debug("Terra coins thingy: " + current[0] + ", => " + current[1]);
      // logger.debug("TerraBalance returning reduced obj: %o", obj);
      return obj;
    }, {} as TerraNativeBalances);
    Object.keys(balance).forEach((key) => {
      logger.debug(
        "chainId: " +
          chainInfo.chainId +
          ", balanceAbs: " +
          balance[key] +
          ", balanceFormatted: " +
          formatUnits(balance[key], 6).toString() +
          ", currencyName: " +
          key
      );
      output.push({
        chainId: chainInfo.chainId,
        balanceAbs: balance[key],
        balanceFormatted: formatUnits(balance[key], 6).toString(),
        currencyName: key,
        currencyAddressNative: key,
      });
    });
  });
  // logger.debug("TerraBalance returning: %o", output);
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
    throw new Error("Failed to fetch native wallet balance for solana");
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
    currencyName: chainInfo.chainName,
    currencyAddressNative: chainInfo.chainName,
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

async function calcForeignAddressesEVM(
  supportedTokens: SupportedToken[],
  chainConfigInfo: ChainConfigInfo
): Promise<string[]> {
  let provider = await new ethers.providers.WebSocketProvider(
    chainConfigInfo.nodeUrl
  );

  // logger.debug("calcForeignAddressesEVM() - entered.");
  let output: string[] = [];
  for (const supportedToken of supportedTokens) {
    const hexAddress = nativeToHexString(
      supportedToken.address,
      supportedToken.chainId
    );
    if (!hexAddress) {
      logger.debug(
        "calcForeignAddressesEVM() - no hexAddress for chainId: " +
          supportedToken.chainId +
          ", address: " +
          supportedToken.address
      );
      continue;
    }
    // logger.debug("calcForeignAddressesEVM() - got hex address: " + hexAddress);
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
      logger.log("Exception thrown from getForeignAssetEth");
    }

    if (!foreignAddress) {
      continue;
    }
    output.push(foreignAddress);
  }

  provider.destroy();
  return output;
}

async function calcForeignAddressesTerra(
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
      logger.log("Foreign address exception.");
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
  const foreignAddresses = await calcForeignAddressesEVM(
    supportedTokens,
    chainConfig
  );
  // logger.debug("foreignAddress: %o", foreignAddresses);
  const output: WalletBalance[] = [];
  if (!chainConfig.walletPrivateKey) {
    return output;
  }
  for (const privateKey of chainConfig.walletPrivateKey) {
    for (const address of foreignAddresses) {
      try {
        const balance = await pullEVMBalance(chainConfig, privateKey, address);
        if (balance) {
          output.push(balance);
        }
      } catch (e) {
        logger.error("pullEVMBalance failed: " + e);
      }
    }
  }

  return output;
}

async function pullAllTerraTokens(
  supportedTokens: SupportedToken[],
  chainConfig: ChainConfigInfo
) {
  const foreignAddresses = await calcForeignAddressesTerra(
    supportedTokens,
    chainConfig
  );
  logger.debug("pullAllTerraTokens() - foreignAddresses: %o", foreignAddresses);
  const output: WalletBalance[] = [];
  if (!chainConfig.walletPrivateKey) {
    return output;
  }
  for (const privateKey of chainConfig.walletPrivateKey) {
    for (const address of foreignAddresses) {
      const balance = await pullTerraBalance(chainConfig, privateKey, address);
      if (balance) {
        output.push(balance);
      }
    }
  }
  logger.debug("pullAllTerraTokens() - returning %o", output);

  return output;
}
