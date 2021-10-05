/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PriceData } from '@pythnetwork/client'
import { IStrategy } from './strategy'

export abstract class StrategyBase implements IStrategy {
    protected symbol: string
    constructor (symbol: string, bufSize: number) {
      this.symbol = symbol
      this.createBuffer(bufSize)
    }

    abstract put(priceData: PriceData): boolean
    abstract createBuffer(size: number): void
    abstract clearBuffer(): void
    abstract getPrice(): number
}
