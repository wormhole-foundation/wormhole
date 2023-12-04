import { ChainId, tryNativeToHexString } from "@certusone/wormhole-sdk";
import { BigNumber, BigNumberish, utils, ContractReceipt } from 'ethers';
import {
  init,
  ChainInfo,
  loadScriptConfig,
  getDeliveryProvider,
  getOperatingChains,
  loadChains,
  getDeliveryProviderAddress,
} from "../helpers/env";
import { buildOverrides } from "../helpers/deployments";

import type {
  DeliveryProvider,
  DeliveryProviderStructs,
} from "../../../ethers-contracts/DeliveryProvider";

type AwaitedProperties<T> = {
  [K in keyof T]: Awaited<T[K]>;
};
type UpdateStruct = AwaitedProperties<DeliveryProviderStructs.UpdateStruct>;
type CoreConfigStruct = AwaitedProperties<DeliveryProviderStructs.CoreConfigStruct>;

interface Config {
  pricingInfo: PricingInfo[];
  deliveryGasOverheads: DeliveryGasOverhead[];
  maximumBudgets: MaximumBudget[];
  conversionBuffers: AssetConversionBuffer[];
  rewardAddresses: RewardAddress[];
  supportedChains: SupportedChain[];
  supportedKeysByChain: SupportedKeys[];
}

interface PricingInfo {
  chainId: ChainId;
  updatePriceGas: BigNumberish;
  updatePriceNative: BigNumberish;
}

interface DeliveryGasOverhead {
  chainId: ChainId;
  updateGasOverhead: BigNumberish;
}

interface MaximumBudget {
  chainId: ChainId;
  updateMaximumBudget: BigNumberish;
}

interface AssetConversionBuffer {
  chainId: ChainId;
  buffer: bigint;
  bufferDenominator: bigint;
}

interface SupportedKeys {
  chainId: ChainId;
  supportedKeys: number[];
}

interface RewardAddress {
  chainId: ChainId;
  rewardAddress: string;
}

interface SupportedChain {
  chainId: ChainId;
  isSupported: boolean;
}

const zeroAddress = "0x0000000000000000000000000000000000000000";
const whZeroAddress =
"0x0000000000000000000000000000000000000000000000000000000000000000";

const processName = "configureDeliveryProvider";
init();
const operatingChains = getOperatingChains();
const allChains = loadChains();
const config: Config = loadScriptConfig(processName);

async function run() {
  console.log(`Start! ${processName}`);
  
  const updateTasks = operatingChains.map((chain) =>
    updateDeliveryProviderConfiguration(config, chain)
  );
  const results = await Promise.allSettled(updateTasks);
  for (const result of results) {
    if (result.status === "rejected") {
      console.log(
        `Updates processing failed: ${result.reason?.stack || result.reason}`
      );
    } else {
      // Print update details; this reflects the exact updates requested to the contract.
      // Note that we assume that this update element was added because
      // some modification was requested to the contract.
      // This depends on the behaviour of the process functions.
      printUpdate(result.value.updates, result.value.coreConfig, result.value.chain);
    }
  }
}

function printUpdate(updates: UpdateStruct[], coreConfig: CoreConfigStruct, { chainId }: ChainInfo) {
  const messages: string[] = [];

  if (coreConfig.updateRewardAddress || coreConfig.updateSupportedKeyTypes) {
    messages.push(`Core configuration updates for operating chain ${chainId}:`);
  }
  if (coreConfig.updateRewardAddress) {
    messages.push(`  Reward address: ${coreConfig.rewardAddress}`);
  }
  if (coreConfig.updateSupportedKeyTypes) {
    const bitmap = BigNumber.isBigNumber(coreConfig.supportedKeyTypesBitmap)
      ? coreConfig.supportedKeyTypesBitmap.toBigInt()
      : BigNumber.from(coreConfig.supportedKeyTypesBitmap).toBigInt();
    const supportedKeys = extractKeys(bitmap);
    messages.push(`  Supported key types: [${supportedKeys.join(", ")}]`);
  }

  for (const update of updates) {
    messages.push(`Updates for operating chain ${chainId} and target chain ${update.chainId}:`);
    if (update.updatePrice) {
      const assetPrice = utils.formatUnits(update.nativeCurrencyPrice, 6);
      const gasPrice = utils.formatUnits(update.gasPrice, "gwei");
      messages.push(`  Asset price update: $${assetPrice}`);
      messages.push(`  Gas price update: ${gasPrice} gwei`);
    }
    if (update.updateDeliverGasOverhead) {
      messages.push(`  Deliver gas overhead update: ${update.newGasOverhead}`);
    }
    if (update.updateMaximumBudget) {
      const maximumBudget = utils.formatEther(update.maximumTotalBudget);
      messages.push(`  Maximum budget update: ${maximumBudget}`);
    }
    if (update.updateTargetChainAddress) {
      messages.push(`  Target chain address update: ${update.targetChainAddress}`);
    }
    if (update.updateSupportedChain) {
      messages.push(`  Supported chain update: ${update.isSupported}`);
    }
    if (update.updateAssetConversionBuffer) {
      const bufferDenominator = BigNumber.isBigNumber(update.bufferDenominator) ? update.bufferDenominator : BigNumber.from(update.bufferDenominator);
      const buffer = BigNumber.isBigNumber(update.buffer) ? update.buffer : BigNumber.from(update.buffer);
      messages.push(`  Asset conversion buffer ratio: (${bufferDenominator.toBigInt()} / ${bufferDenominator.add(buffer).toBigInt()})`)
    }
    if (update.updateSupportedChain) {
      messages.push(`  Supported chain update: ${update.isSupported}`);
    }
    if (update.updateTargetChainAddress) {
      messages.push(`  Target chain address update: ${utils.hexlify(update.targetChainAddress)}`);
    }
    if (update.updateAssetConversionBuffer) {
      messages.push(`  Asset conversion buffer update:`);
      messages.push(`    buffer: ${update.buffer}`);
      messages.push(`    buffer denominator: ${update.bufferDenominator}`);
    }
  }

  console.log(messages.join("\n"));
}

async function updateDeliveryProviderConfiguration(config: Config, chain: ChainInfo) {
  const deliveryProvider = await getDeliveryProvider(chain);
  const updates: UpdateStruct[] = [];

  for (const priceUpdate of config.pricingInfo) {
    console.log(
      `Processing price update for operating chain ${chain.chainId} and target chain ${priceUpdate.chainId}`
    );
    await processPriceUpdate(updates, deliveryProvider, priceUpdate);
  }

  for (const gasOverheadUpdate of config.deliveryGasOverheads) {
    console.log(
      `Processing gas overhead update for operating chain ${chain.chainId} and target chain ${gasOverheadUpdate.chainId}`
    );
    await processGasOverheadUpdate(
      updates,
      deliveryProvider,
      gasOverheadUpdate
    );
  }

  for (const maximumBudgetUpdate of config.maximumBudgets) {
    console.log(
      `Processing maximum budget update for operating chain ${chain.chainId} and target chain ${maximumBudgetUpdate.chainId}`
    );
    await processMaximumBudgetUpdate(
      updates,
      deliveryProvider,
      maximumBudgetUpdate
    );
  }

  for (const conversionBuffer of config.conversionBuffers) {
    console.log(
      `Processing asset conversion buffer update for operating chain ${chain.chainId} and target chain ${conversionBuffer.chainId}`
    );
    await processConversionBufferUpdate(
      updates,
      deliveryProvider,
      conversionBuffer
    );
  }

  for (const targetChain of allChains) {
    console.log(
      `Processing targetChainAddress update for operating chain ${chain.chainId} and target chain ${targetChain.chainId}`
    );

    await processTargetChainAddressUpdate(
      updates,
      deliveryProvider,
      targetChain,
    );
  }

  for (const supportedChain of config.supportedChains) {
    console.log(
      `Processing supported chain update for operating chain ${chain.chainId} and target chain ${supportedChain.chainId}`
    );

    await processSupportedChainUpdate(
      updates,
      deliveryProvider,
      supportedChain,
    );
  }

  for (const conversionBufferConfig of config.conversionBuffers) {
    console.log(
      `Processing supported chain update for operating chain ${chain.chainId} and target chain ${conversionBufferConfig.chainId}`
    );

    await processAssetConversionBufferUpdates(
      updates,
      deliveryProvider,
      conversionBufferConfig,
    );
  }

  const coreConfig = await processCoreConfigUpdates(
    config.rewardAddresses,
    config.supportedKeysByChain,
    deliveryProvider,
    chain,
  );

  const overrides = await buildOverrides(
    () => deliveryProvider.estimateGas.updateConfig(updates, coreConfig),
    chain
  );

  // `coreConfig.updateWormholeRelayer` is an obsolete configuration parameter so we ignore it here.
  if (updates.length === 0 && !coreConfig.updateRewardAddress && !coreConfig.updateSupportedKeyTypes) {
    console.log(`No updates for operating chain ${chain.chainId}`);
    return { updates, coreConfig, chain };
  }

  console.log(`Sending update tx for operating chain ${chain.chainId}. Updates: ${JSON.stringify(updates)}`);

  let receipt: ContractReceipt;
  try {
    const tx = await deliveryProvider.updateConfig(
      updates,
      coreConfig,
      overrides
    );
    receipt = await tx.wait();
  } catch (error) {
    console.error(
      `Updates failed on operating chain ${chain.chainId}. Error: ${error}`
    );
    throw error;
  }

  if (receipt.status !== 1) {
    throw new Error(
      `Updates failed on operating chain ${chain.chainId}. Tx id ${receipt.transactionHash}`
    );
  }

  return { updates, coreConfig, chain };
}

async function processPriceUpdate(
  updates: UpdateStruct[],
  deliveryProvider: DeliveryProvider,
  { chainId, updatePriceGas, updatePriceNative }: PricingInfo
) {
  const currentGasPrice = await deliveryProvider.gasPrice(chainId);
  const currentNativeAssetPrice = await deliveryProvider.nativeCurrencyPrice(
    chainId
  );

  if (
    !currentGasPrice.eq(updatePriceGas) ||
    !currentNativeAssetPrice.eq(updatePriceNative)
  ) {
    const update = getUpdateConfig(updates, chainId);
    update.updatePrice = true;
    update.nativeCurrencyPrice = updatePriceNative;
    update.gasPrice = updatePriceGas;
  }
}

async function processGasOverheadUpdate(
  updates: UpdateStruct[],
  deliveryProvider: DeliveryProvider,
  { chainId, updateGasOverhead }: DeliveryGasOverhead
) {
  const currentGasOverhead = await deliveryProvider.deliverGasOverhead(chainId);

  if (!currentGasOverhead.eq(updateGasOverhead)) {
    const update = getUpdateConfig(updates, chainId);
    update.updateDeliverGasOverhead = true;
    update.newGasOverhead = updateGasOverhead;
  }
}

async function processMaximumBudgetUpdate(
  updates: UpdateStruct[],
  deliveryProvider: DeliveryProvider,
  { chainId, updateMaximumBudget }: MaximumBudget
) {
  const currentMaximumBudget = await deliveryProvider.maximumBudget(chainId);
  if (!currentMaximumBudget.eq(updateMaximumBudget)) {
    const update = getUpdateConfig(updates, chainId);
    update.updateMaximumBudget = true;
    update.maximumTotalBudget = updateMaximumBudget;
  }
}

async function processConversionBufferUpdate(
  updates: UpdateStruct[],
  deliveryProvider: DeliveryProvider,
  { chainId, buffer, bufferDenominator }: AssetConversionBuffer
) {
  const currentBuffer = await deliveryProvider.assetConversionBuffer(chainId);

  if (BigInt(currentBuffer.buffer) !== buffer || BigInt(currentBuffer.bufferDenominator) !== bufferDenominator) {
    const update = getUpdateConfig(updates, chainId);
    update.updateAssetConversionBuffer = true;
    update.buffer = buffer;
    update.bufferDenominator = bufferDenominator;
  }
}

async function processTargetChainAddressUpdate(
  updates: UpdateStruct[],
  deliveryProvider: DeliveryProvider,
  chain: ChainInfo,
) {
  const currentTargetChainAddress = await deliveryProvider.getTargetChainAddress(chain.chainId);
  const targetChainAddress =
  "0x" + tryNativeToHexString(getDeliveryProviderAddress(chain), "ethereum");

  if (currentTargetChainAddress !== targetChainAddress) {
    const update = getUpdateConfig(updates, chain.chainId);
    update.updateTargetChainAddress = true;
    update.targetChainAddress = targetChainAddress;
  }
}

async function processSupportedChainUpdate(
  updates: UpdateStruct[],
  deliveryProvider: DeliveryProvider,
  { chainId, isSupported }: SupportedChain,
) {
  const currentIsSupported = await deliveryProvider.isChainSupported(chainId);

  if (currentIsSupported !== isSupported) {
    const update = getUpdateConfig(updates, chainId);
    update.updateSupportedChain = true;
    update.isSupported = isSupported;
  }
}

async function processCoreConfigUpdates(
  rewardAddresses: RewardAddress[],
  supportedKeysByChain: SupportedKeys[],
  deliveryProvider: DeliveryProvider,
  chain: ChainInfo,
) {
  const coreConfig: CoreConfigStruct = {
    updateRewardAddress: false,
    updateWormholeRelayer: false,
    updateSupportedKeyTypes: false,
    coreRelayer: zeroAddress,
    rewardAddress: zeroAddress,
    supportedKeyTypesBitmap: 0,
  };

  const rewardAddress = rewardAddresses.find((element) => {
    return element.chainId === chain.chainId;
  })?.rewardAddress;

  if (rewardAddress !== undefined) {
    const currentRewardAddress = await deliveryProvider.rewardAddress();

    if (currentRewardAddress !== rewardAddress) {
      coreConfig.updateRewardAddress = true;
      coreConfig.rewardAddress = rewardAddress;
    }
  }

  const supportedKeys = supportedKeysByChain.find((element) => {
    return element.chainId === chain.chainId;
  })?.supportedKeys;

  if (supportedKeys !== undefined) {
    const currentSupportedKeys = await deliveryProvider.getSupportedKeys();
    const newSupportedKeys = generateBitmap(supportedKeys);

    if (currentSupportedKeys.toBigInt() !== newSupportedKeys) {
      coreConfig.updateSupportedKeyTypes = true;
      coreConfig.supportedKeyTypesBitmap = newSupportedKeys;
    }
  }

  return coreConfig;
}

async function processAssetConversionBufferUpdates(
  updates: UpdateStruct[],
  deliveryProvider: DeliveryProvider,
  conversionBufferConfig: AssetConversionBuffer,
) {
  const { chainId, buffer, bufferDenominator } = conversionBufferConfig;
  const update = getUpdateConfig(updates, chainId);

  const { 
    buffer: currentBuffer,
    bufferDenominator: currentBufferDenominator
  } = await deliveryProvider.assetConversionBuffer(chainId);

  if (buffer !== BigInt(currentBuffer) || bufferDenominator !== BigInt(currentBufferDenominator)) {
    update.updateAssetConversionBuffer = true;
    update.buffer = buffer;
    update.bufferDenominator = bufferDenominator;
  }
}

function getUpdateConfig(
  updates: UpdateStruct[],
  chainId: ChainId
): UpdateStruct {
  let update = updates.find((element) => {
    return element.chainId === chainId;
  });

  if (update === undefined) {
    update = createEmptyUpdateConfig(chainId);
    updates.push(update);
  }

  return update;
}

function createEmptyUpdateConfig(chainId: ChainId): UpdateStruct {
  return {
    chainId,
    updateAssetConversionBuffer: false,
    updateDeliverGasOverhead: false,
    updatePrice: false,
    updateMaximumBudget: false,
    updateTargetChainAddress: false,
    updateSupportedChain: false,
    isSupported: false,
    buffer: 0,
    bufferDenominator: 0,
    newGasOverhead: 0,
    gasPrice: 0,
    nativeCurrencyPrice: 0,
    targetChainAddress: whZeroAddress,
    maximumTotalBudget: 0,
  };
}

// We assume the bitmap is 256 bits wide which is accurate at the time of writing.
function extractKeys(bitmap: bigint): bigint[] {
  const keys = [];
  for (let i = 0n; i < 256n; ++i) {
    if ((bitmap & (1n << i)) > 0n) {
      keys.push(i);
    }
  }
  return keys;
}

function generateBitmap(keys: number[]): bigint {
  let bitmap = 0n;
  for (const key of keys) {
    bitmap |= 1n << BigInt(key);
  }
  return bitmap;
}

run().then(() => console.log(`Done! ${processName}`));
