/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

/**
 * A generic Price ticker information class
 */
export class PriceTicker {
  constructor (price: BigInt, confidence: BigInt, exponent: number, networkTime: BigInt) {
    this._price = price
    this._confidence = confidence
    this._exponent = exponent
    this._networkTime = networkTime
  }

    /** price */
    private _price: BigInt;
    public get price (): BigInt {
      return this._price
    }

    public set price (value: BigInt) {
      this._price = value
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
    private _networkTime: BigInt;
    public get networkTime (): BigInt {
      return this._networkTime
    }

    public set networkTime (value: BigInt) {
      this._networkTime = value
    }
}
