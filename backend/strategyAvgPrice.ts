/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { StrategyBaseQueue } from './strategyqueuebase'

class StrategyAveragePrice extends StrategyBaseQueue {
  getPrice (): number {
    return 0
  }
}
