import { PoxInfo, Pox4SignatureTopic } from '@stacks/stacking';
import crypto from 'crypto';
import {
  Account,
  getAccounts,
  maxAmount,
  parseEnvInt,
  waitForSetup,
  logger,
  burnBlockToRewardCycle,
} from './common';

const randInt = () => crypto.randomInt(0, 0xffffffffffff);
const stackingInterval = parseEnvInt('STACKING_INTERVAL', true);
const postTxWait = parseEnvInt('POST_TX_WAIT', true);
const stackingCycles = parseEnvInt('STACKING_CYCLES', true);

const SLOT_MULTIPLIER = 2.5;
const DEFAULT_NUM_SLOTS = 2;

let startTxFee = 1000;
const getNextTxFee = () => startTxFee++;

type RewardCycleId = number;
type AmountToStack = bigint;

// Map to store fixed stacking amounts per reward cycle to ensure consistent
// proportional weights based on target slots. Maps target reward cycle ID to
// fixed amount per slot for that cycle.
//
// This prevents dynamic threshold changes from causing unexpected weight
// distribution as stackers join throughout the cycle and affect the
// participation-based threshold.
const cycleStackingAmounts = new Map<RewardCycleId, AmountToStack>();

/**
 * Compute and store the fixed stacking amount for a given target reward cycle.
 * This ensures all stackers have expected weights regardless of the stacking
 * order within the cycle.
 *
 * @param targetRewardCycle The reward cycle ID for which the fixed amount is
 *                          computed
 * @param currentThreshold The current minimum threshold for the cycle
 * @param multiplier Optional multiplier for the starting threshold
 *                   (default: SLOT_MULTIPLIER)
 * @returns The fixed stacking amount for this cycle
 */
function getFixedStackingAmount(
  targetRewardCycle: number,
  currentThreshold: number,
  multiplier: number = SLOT_MULTIPLIER
): AmountToStack {
  if (cycleStackingAmounts.has(targetRewardCycle)) {
    return cycleStackingAmounts.get(targetRewardCycle)!;
  }

  // Use the threshold at the time this target cycle was first encountered.
  // Bump by multiplier to avoid getting stuck if threshold increases slightly
  // over time.
  const fixedAmount = BigInt(Math.floor(currentThreshold * multiplier));
  cycleStackingAmounts.set(targetRewardCycle, fixedAmount);

  logger.info(
    {
      targetRewardCycle: targetRewardCycle,
      currentThreshold,
      fixedAmount: fixedAmount.toString(),
      multiplier,
    },
    `Set fixed stacking amount for target reward cycle ${targetRewardCycle}`
  );

  return fixedAmount;
}

async function run(stackingKeys: string[], stackingSlotDistribution: number[]) {
  const accounts = getAccounts(stackingKeys, stackingSlotDistribution);
  const poxInfo = await accounts[0].client.getPoxInfo();
  if (!poxInfo.contract_id.endsWith('.pox-4')) {
    logger.info(
      {
        poxContract: poxInfo.contract_id,
      },
      `Pox contract is not .pox-4, skipping stacking (contract=${poxInfo.contract_id})`
    );
    return;
  }

  const runLog = logger.child({
    burnHeight: poxInfo.current_burnchain_block_height,
  });

  const accountInfos = await Promise.all(
    accounts.map(async a => {
      const info = await a.client.getAccountStatus();
      const unlockHeight = Number(info.unlock_height);
      const lockedAmount = BigInt(info.locked);
      const balance = BigInt(info.balance);
      return { ...a, info, unlockHeight, lockedAmount, balance };
    })
  );

  let txSubmitted = false;

  // Bump min threshold by SLOT_MULTIPLIER to avoid getting stuck if threshold
  // increases.
  const minStx = Math.floor(poxInfo.next_cycle.min_threshold_ustx * SLOT_MULTIPLIER);
  const nextCycleStx = poxInfo.next_cycle.stacked_ustx;
  if (nextCycleStx < minStx) {
    runLog.info(`Next cycle has less than min threshold.. stacking should be performed soon`);
  }

  await Promise.all(
    accountInfos.map(async account => {
      if (account.lockedAmount === 0n) {
        runLog.info(
          {
            burnHeight: poxInfo.current_burnchain_block_height,
            unlockHeight: account.unlockHeight,
            account: account.index,
          },
          `Account ${account.index} is unlocked, stack-stx required`
        );
        await stackStx(poxInfo, account, account.balance);
        txSubmitted = true;
        return;
      }
      const unlockHeightCycle = burnBlockToRewardCycle(account.unlockHeight);
      const nowCycle = burnBlockToRewardCycle(poxInfo.current_burnchain_block_height ?? 0);
      if (unlockHeightCycle === nowCycle + 1) {
        runLog.info(
          {
            burnHeight: poxInfo.current_burnchain_block_height,
            unlockHeight: account.unlockHeight,
            account: account.index,
            nowCycle,
            unlockCycle: unlockHeightCycle,
          },
          `Account ${account.index} unlocks before next cycle ${account.unlockHeight} vs ${poxInfo.current_burnchain_block_height}, stack-extend required`
        );
        await stackExtend(poxInfo, account);
        txSubmitted = true;
        return;
      }
      runLog.info(
        {
          burnHeight: poxInfo.current_burnchain_block_height,
          unlockHeight: account.unlockHeight,
          account: account.index,
          nowCycle,
          unlockCycle: unlockHeightCycle,
        },
        `Account ${account.index} is locked for next cycle, skipping stacking`
      );
    })
  );

  if (txSubmitted) {
    await new Promise(resolve => setTimeout(resolve, postTxWait * 1000));
  }
}

async function stackStx(poxInfo: PoxInfo, account: Account, balance: bigint) {
  // Determine the fixed stacking amount per slot for the target reward cycle.
  // This ensures the stacked amount per slot is constant for the entire cycle,
  // regardless of potential increases in the minimum threshold.
  const baseStackingAmount = getFixedStackingAmount(
    poxInfo.next_cycle.id,
    poxInfo.next_cycle.min_threshold_ustx
  );

  // Calculate total amount needed based on target slots and fixed base amount.
  const amountToStack = baseStackingAmount * BigInt(account.targetSlots);

  // Compare with current threshold.
  const currentThreshold = poxInfo.next_cycle.min_threshold_ustx;
  const adjustedThreshold = Math.floor(currentThreshold * SLOT_MULTIPLIER);

  if (balance < baseStackingAmount) {
    throw new Error(
      `Insufficient balance to stack minimum amount (required=${baseStackingAmount}, balance=${balance})`
    );
  }

  if (balance < amountToStack) {
    throw new Error(
      `Insufficient balance to stack (required=${amountToStack}, balance=${balance}), this can lead to unexpected weight distribution.`
    );
  }
  const authId = randInt();
  const sigArgs = {
    topic: Pox4SignatureTopic.StackStx,
    rewardCycle: poxInfo.reward_cycle_id,
    poxAddress: account.btcAddr,
    period: stackingCycles,
    signerPrivateKey: account.signerPrivKey,
    authId,
    maxAmount,
  } as const;
  const signerSignature = account.client.signPoxSignature(sigArgs);
  const stackingArgs = {
    poxAddress: account.btcAddr,
    privateKey: account.privKey,
    amountMicroStx: amountToStack,
    burnBlockHeight: poxInfo.current_burnchain_block_height,
    cycles: stackingCycles,
    fee: getNextTxFee(),
    signerKey: account.signerPubKey,
    signerSignature,
    authId,
    maxAmount,
  };
  account.logger.debug(
    {
      ...stackingArgs,
      ...sigArgs,
      // The total amount to stack.
      stackedAmount: amountToStack.toString(),
      // The fixed amount per slot for the target reward cycle.
      baseStackingAmount: baseStackingAmount.toString(),
      // How many slots the account is targeting to stack. Will stack this
      // amount multiplied by a constant multiplier to avoid getting locked out
      // if the threshold increases.
      targetSlots: account.targetSlots,
      // The current minimum threshold for the cycle.
      currentThreshold,
      // The threshold after applying the multiplier.
      adjustedThreshold,
    },
    `Stack-stx with args:`
  );
  const stackResult = await account.client.stack(stackingArgs);
  account.logger.info(
    {
      ...stackResult,
    },
    `Stack-stx tx result`
  );
}

async function stackExtend(
  poxInfo: PoxInfo,
  account: Account & { lockedAmount: bigint; balance: bigint }
) {
  const authId = randInt();
  const sigArgs = {
    topic: Pox4SignatureTopic.StackExtend,
    rewardCycle: poxInfo.reward_cycle_id,
    poxAddress: account.btcAddr,
    period: stackingCycles,
    signerPrivateKey: account.signerPrivKey,
    authId,
    maxAmount,
  } as const;
  const signerSignature = account.client.signPoxSignature(sigArgs);
  const stackingArgs = {
    poxAddress: account.btcAddr,
    privateKey: account.privKey,
    extendCycles: stackingCycles,
    fee: getNextTxFee(),
    signerKey: account.signerPubKey,
    signerSignature,
    authId,
    maxAmount,
  };
  account.logger.debug(
    {
      stxAddress: account.stxAddress,
      account: account.index,
      ...stackingArgs,
      ...sigArgs,
    },
    `Stack-extend with args:`
  );
  const stackResult = await account.client.stackExtend(stackingArgs);
  account.logger.info(
    {
      stxAddress: account.stxAddress,
      account: account.index,
      ...stackResult,
    },
    `Stack-extend tx result`
  );
}

async function loop() {
  const stackingKeys = process.env.STACKING_KEYS?.split(',') || [];

  if (stackingKeys.length === 0) {
    throw new Error('No stacking keys provided using STACKING_KEYS.');
  }

  const envStackingSlotDistribution =
    process.env.STACKING_SLOT_DISTRO?.split(',').map(Number) || [];
  const stackingSlotDistribution: number[] = Array(stackingKeys.length)
    .fill(DEFAULT_NUM_SLOTS)
    .map((defaultValue, index) => envStackingSlotDistribution[index] ?? defaultValue);

  logger.info(
    {
      stackingKeys: stackingKeys.length,
      stackingSlotDistribution,
      stackingInterval,
      postTxWait,
      stackingCycles,
    },
    `Starting stacker with configuration:`
  );

  await waitForSetup(stackingKeys, stackingSlotDistribution);

  while (true) {
    try {
      await run(stackingKeys, stackingSlotDistribution);
    } catch (e) {
      console.error('Error running stacking:', e);
    }
    await new Promise(resolve => setTimeout(resolve, stackingInterval * 1000));
  }
}
loop();
