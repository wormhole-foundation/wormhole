import { PriceTicker } from '../PriceTicker'
import { StrategyBaseQueue } from './strategyBaseQueue'
/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

/**
 * This strategy just caches the last provided price,
 * acting as a single-item buffer.
 */
export class StrategyLastPrice extends StrategyBaseQueue {
  constructor () {
    super(1)
  }

  getPrice (): PriceTicker {
    return this.buffer[0]
  }
}
