/* eslint-disable camelcase */
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

/**
 * A generic Price ticker information class
 */
export class PriceTicker {
  constructor (
    symbol: string,
    price: BigInt,
    price_type: number,
    confidence: BigInt,
    exponent: number,
    twap: BigInt,
    twac: BigInt,
    timestamp: BigInt,
    user_data?: any) {
    this._symbol = symbol
    this._price = price
    this._price_type = price_type
    this._confidence = confidence
    this._exponent = exponent
    this._timestamp = timestamp
    this._twap = twap
    this._twac = twac
    this._user_data = user_data
  }

  private _symbol: string
  public get symbol (): string {
    return this._symbol
  }

  public set symbol (value: string) {
    this._symbol = value
  }

  /** price */
  private _price: BigInt;
  public get price (): BigInt {
    return this._price
  }

  public set price (value: BigInt) {
    this._price = value
  }

  /** price_type */
  private _price_type: number
  public get price_type (): number {
    return this._price_type
  }

  public set price_type (value: number) {
    this._price_type = value
  }

  /** a confidence interval */
  private _confidence: BigInt;
  public get confidence (): BigInt {
    return this._confidence
  }

  public set confidence (value: BigInt) {
    this._confidence = value
  }

  /** exponent (fixed point) */
  private _exponent: number;
  public get exponent (): number {
    return this._exponent
  }

  public set exponent (value: number) {
    this._exponent = value
  }

  /** time in blockchain network units */
  private _timestamp: BigInt;
  public get timestamp (): BigInt {
    return this._timestamp
  }

  public set timestamp (value: BigInt) {
    this._timestamp = value
  }

  private _twac: BigInt
  public get twac (): BigInt {
    return this._twac
  }

  public set twac (value: BigInt) {
    this._twac = value
  }

  private _twap: BigInt
  public get twap (): BigInt {
    return this._twap
  }

  public set twap (value: BigInt) {
    this._twap = value
  }

  private _user_data: any
  public get user_data (): any {
    return this._user_data
  }

  public set user_data (value: any) {
    this._user_data = value
  }
}
