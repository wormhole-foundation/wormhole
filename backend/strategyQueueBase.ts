/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PriceData } from '@pythnetwork/client'
import { StrategyBase } from './strategyBase'

/**
 * A base class for queue-based buffer strategies
 */
export abstract class StrategyBaseQueue extends StrategyBase {
  private buffer: PriceData[]
  private bufSize: number

  createBuffer (maxSize: number): void {
    this.bufSize = maxSize
  }

  clearBuffer (): void {
    this.buffer.length = 0
  }

  put (priceData: PriceData): boolean {
    if (this.buffer.length === this.bufSize) {
      this.buffer.shift()
    }
    this.buffer.push(priceData)
    return true
  }

  abstract getPrice(): number
}
