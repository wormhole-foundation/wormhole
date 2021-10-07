/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PriceTicker } from '../PriceTicker'
import { IStrategy } from './strategy'

export abstract class StrategyBase implements IStrategy {
  constructor (bufSize: number = 10) {
    this.createBuffer(bufSize)
  }

  abstract put(priceData: PriceTicker): boolean
  abstract createBuffer(size: number): void
  abstract clearBuffer(): void
  abstract getPrice(): PriceTicker
}
