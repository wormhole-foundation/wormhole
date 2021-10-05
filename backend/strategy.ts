/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PriceData } from '@pythnetwork/client'

/**
 * Implements a strategy for obtaining an asset price from
 * a set of received prices in a buffer.
 */
export interface IStrategy {
    /**
     *
     * @param size The size of the buffer
     */
    createBuffer(size: number): void

    /**
     * Clear price buffer
     */
    clearBuffer(): void

    /**
     * Put a new price in buffer.
     * @param priceData  The price data to put
     * @returns true if successful.
     */
    put(priceData: PriceData): boolean

    /**
     * Get the calculated price according to selected strategy.
     */
    getPrice(): number
}
