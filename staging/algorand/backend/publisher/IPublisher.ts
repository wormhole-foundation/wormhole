/* eslint-disable no-unused-vars */
/**
 * Pricecaster Service.
 *
 * Fetcher backend component.
 *
 * (c) 2021 Randlabs, Inc.
 */

import { PriceTicker } from '../common/priceTicker'
import { StatusCode } from '../common/statusCodes'

export type PublishInfo = {
    status: StatusCode,
    reason?: '',
    msgb64?: '',
    block?: BigInt
    txid?: string
}

export interface IPublisher {
    start(): void
    stop(): void
    publish(data: any): Promise<PublishInfo>
}
