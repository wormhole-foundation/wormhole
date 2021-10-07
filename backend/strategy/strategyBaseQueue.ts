/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PriceTicker } from '../PriceTicker'
import { StrategyBase } from './strategyBase'

/**
 * A base class for queue-based buffer strategies
 */
export abstract class StrategyBaseQueue extends StrategyBase {
  protected buffer: PriceTicker[] = []
  private bufSize: number = 0

  createBuffer (maxSize: number): void {
    this.bufSize = maxSize
  }

  clearBuffer (): void {
    this.buffer.length = 0
  }

  put (ticker: PriceTicker): boolean {
    if (this.buffer.length === this.bufSize) {
      this.buffer.shift()
    }
    this.buffer.push(ticker)
    return true
  }

  abstract getPrice(): PriceTicker
}
