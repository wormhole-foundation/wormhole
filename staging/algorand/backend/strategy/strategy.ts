/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * Copyright 2022 Wormhole Project Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { PriceTicker } from '../common/priceTicker'

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
     * Returns the current number of items in buffer
     */
    bufferCount(): number

    /**
     * Put a new price in buffer.
     * @param priceData  The price data to put
     * @returns true if successful.
     */
    put(ticker: PriceTicker): boolean

    /**
     * Get the calculated price according to selected strategy.
     */
    getPrice(): PriceTicker | undefined
}
