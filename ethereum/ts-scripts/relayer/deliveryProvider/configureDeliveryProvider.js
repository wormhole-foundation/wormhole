"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
const wormhole_sdk_1 = require("@certusone/wormhole-sdk");
const ethers_1 = require("ethers");
const env_1 = require("../helpers/env");
const deployments_1 = require("../helpers/deployments");
const zeroAddress = "0x0000000000000000000000000000000000000000";
const whZeroAddress = "0x0000000000000000000000000000000000000000000000000000000000000000";
const processName = "configureDeliveryProvider";
(0, env_1.init)();
const operatingChains = (0, env_1.getOperatingChains)();
const allChains = (0, env_1.loadChains)();
const config = (0, env_1.loadScriptConfig)(processName);
async function run() {
    console.log(`Start! ${processName}`);
    const updateTasks = operatingChains.map((chain) => updateDeliveryProviderConfiguration(config, chain));
    const results = await Promise.allSettled(updateTasks);
    for (const result of results) {
        if (result.status === "rejected") {
            console.log(`Price update failed: ${result.reason?.stack || result.reason}`);
        }
        else {
            // Print update details; this reflects the exact updates requested to the contract.
            // Note that we assume that this update element was added because
            // some modification was requested to the contract.
            // This depends on the behaviour of the process functions.
            for (const update of result.value?.updates || []) {
                if (result.value?.chain) {
                    printUpdate(update, result.value.chain);
                }
            }
        }
    }
}
function printUpdate(update, { chainId }) {
    let messages = [
        `Updates for operating chain ${chainId} and target chain ${update.chainId}:`,
    ];
    if (update.updatePrice) {
        const assetPrice = ethers_1.utils.formatUnits(update.nativeCurrencyPrice, 6);
        const gasPrice = ethers_1.utils.formatUnits(update.gasPrice, "gwei");
        messages.push(`  Asset price update: $${assetPrice}`);
        messages.push(`  Gas price update: ${gasPrice} gwei`);
    }
    if (update.updateDeliverGasOverhead) {
        messages.push(`  Deliver gas overhead update: ${update.newGasOverhead}`);
    }
    if (update.updateMaximumBudget) {
        const maximumBudget = ethers_1.utils.formatEther(update.maximumTotalBudget);
        messages.push(`  Maximum budget update: ${maximumBudget}`);
    }
    console.log(messages.join("\n"));
}
async function updateDeliveryProviderConfiguration(config, chain) {
    const deliveryProvider = await (0, env_1.getDeliveryProvider)(chain);
    const updates = [];
    for (const priceUpdate of config.pricingInfo) {
        console.log(`Processing price update for operating chain ${chain.chainId} and target chain ${priceUpdate.chainId}`);
        await processPriceUpdate(updates, deliveryProvider, priceUpdate);
    }
    for (const gasOverheadUpdate of config.deliveryGasOverheads) {
        console.log(`Processing gas overhead update for operating chain ${chain.chainId} and target chain ${gasOverheadUpdate.chainId}`);
        await processGasOverheadUpdate(updates, deliveryProvider, gasOverheadUpdate);
    }
    for (const maximumBudgetUpdate of config.maximumBudgets) {
        console.log(`Processing maximum budget update for operating chain ${chain.chainId} and target chain ${maximumBudgetUpdate.chainId}`);
        await processMaximumBudgetUpdate(updates, deliveryProvider, maximumBudgetUpdate);
    }
    for (const targetChain of allChains) {
        console.log(`Processing targetChainAddress update for operating chain ${chain.chainId} and target chain ${targetChain.chainId}`);
        await processTargetChainAddressUpdate(updates, deliveryProvider, targetChain);
    }
    const coreConfig = await processCoreConfigUpdates(config.rewardAddresses, deliveryProvider, chain);
    const overrides = await (0, deployments_1.buildOverrides)(() => deliveryProvider.estimateGas.updateConfig(updates, coreConfig), chain);
    console.log(`Sending update tx for operating chain ${chain.chainId}`);
    console.log(JSON.stringify(updates));
    const tx = await deliveryProvider.updateConfig(updates, coreConfig, overrides);
    const receipt = await tx.wait();
    if (receipt.status !== 1) {
        throw new Error(`Updates failed on operating chain ${chain.chainId}. Tx id ${receipt.transactionHash}`);
    }
    return { updates, chain };
}
async function processPriceUpdate(updates, deliveryProvider, { chainId, updatePriceGas, updatePriceNative }) {
    const currentGasPrice = await deliveryProvider.gasPrice(chainId);
    const currentNativeAssetPrice = await deliveryProvider.nativeCurrencyPrice(chainId);
    if (!currentGasPrice.eq(updatePriceGas) ||
        !currentNativeAssetPrice.eq(updatePriceNative)) {
        const update = getUpdateConfig(updates, chainId);
        update.updatePrice = true;
        update.nativeCurrencyPrice = updatePriceNative;
        update.gasPrice = updatePriceGas;
    }
}
async function processGasOverheadUpdate(updates, deliveryProvider, { chainId, updateGasOverhead }) {
    const currentGasOverhead = await deliveryProvider.deliverGasOverhead(chainId);
    if (!currentGasOverhead.eq(updateGasOverhead)) {
        const update = getUpdateConfig(updates, chainId);
        update.updateDeliverGasOverhead = true;
        update.newGasOverhead = updateGasOverhead;
    }
}
async function processMaximumBudgetUpdate(updates, deliveryProvider, { chainId, updateMaximumBudget }) {
    const currentMaximumBudget = await deliveryProvider.maximumBudget(chainId);
    if (!currentMaximumBudget.eq(updateMaximumBudget)) {
        const update = getUpdateConfig(updates, chainId);
        update.updateMaximumBudget = true;
        update.maximumTotalBudget = updateMaximumBudget;
    }
}
async function processTargetChainAddressUpdate(updates, deliveryProvider, chain) {
    const currentTargetChainAddress = await deliveryProvider.getTargetChainAddress(chain.chainId);
    const targetChainAddress = "0x" + (0, wormhole_sdk_1.tryNativeToHexString)((0, env_1.getDeliveryProviderAddress)(chain), "ethereum");
    if (currentTargetChainAddress !== targetChainAddress) {
        const update = getUpdateConfig(updates, chain.chainId);
        update.updateTargetChainAddress = true;
        update.targetChainAddress = targetChainAddress;
    }
}
async function processCoreConfigUpdates(rewardAddresses, deliveryProvider, chain) {
    const coreConfig = {
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
    if (!rewardAddress) {
        throw new Error("Failed to find reward address info for chain " + chain.chainId);
    }
    const currentRewardAddress = await deliveryProvider.rewardAddress();
    console.log("currentRewardAddress: " + currentRewardAddress);
    console.log("rewardAddress: " + rewardAddress);
    if (currentRewardAddress !== rewardAddress) {
        coreConfig.updateRewardAddress = true;
        coreConfig.rewardAddress = rewardAddress;
    }
    return coreConfig;
}
function getUpdateConfig(updates, chainId) {
    let update = updates.find((element) => {
        return element.chainId === chainId;
    });
    if (update === undefined) {
        update = createEmptyUpdateConfig(chainId);
        updates.push(update);
    }
    return update;
}
function createEmptyUpdateConfig(chainId) {
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
run().then(() => console.log(`Done! ${processName}`));
