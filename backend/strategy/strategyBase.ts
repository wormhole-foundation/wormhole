/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PriceTicker } from '../common/priceTicker'
import { IStrategy } from './strategy'

/**
 * A base class for queue-based buffer strategies
 */
export abstract class StrategyBase implements IStrategy {
  protected buffer!: PriceTicker[]
  protected bufSize!: number

  constructor (bufSize: number = 10) {
    this.createBuffer(bufSize)
  }

  createBuffer (maxSize: number): void {
    this.buffer = []
    this.bufSize = maxSize
  }

  clearBuffer (): void {
    this.buffer.length = 0
  }

  bufferCount (): number {
    return this.buffer.length
  }

  put (ticker: PriceTicker): boolean {
    if (this.buffer.length === this.bufSize) {
      this.buffer.shift()
    }
    this.buffer.push(ticker)
    return true
  }

  abstract getPrice(): PriceTicker | undefined
}
