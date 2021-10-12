import { PriceTicker } from '../common/priceTicker'
import { StrategyBase } from './strategyBase'
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
export class StrategyLastPrice extends StrategyBase {
  getPrice (): PriceTicker | undefined {
    const ret = this.buffer[this.buffer.length - 1]
    this.clearBuffer()
    return ret
  }
}
