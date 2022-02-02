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
