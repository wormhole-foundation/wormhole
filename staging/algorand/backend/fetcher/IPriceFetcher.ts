/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PriceTicker } from '../common/priceTicker'
import { IStrategy } from '../strategy/strategy'

export interface IPriceFetcher {
    start(): void
    stop(): void
    hasData(): boolean

    /**
     * Set price aggregation strategy for this fetcher.
     * @param IStrategy The local price aggregation strategy
     */
    setStrategy(s: IStrategy): void

    /**
     * Get the current price, according to running strategy.
     */
    queryTicker(): PriceTicker | undefined
}
