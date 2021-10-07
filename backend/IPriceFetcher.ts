/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PriceTicker } from './PriceTicker'

export interface IPriceFetcher {
    start(): void
    stop(): void

    /**
     * Set price aggregation strategy for this fetcher.
     * @param IStrategy The local price aggregation strategy
     */
    setStrategy(IStrategy)

    /**
     * Get the current price, according to running strategy.
     */
    queryTicker(): PriceTicker
}
