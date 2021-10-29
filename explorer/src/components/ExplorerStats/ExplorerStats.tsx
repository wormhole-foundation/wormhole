import React, { useContext, useEffect, useState } from 'react';
import { Spin, } from 'antd'
import { useIntl, } from 'gatsby-plugin-intl'
import { BigTableMessage } from '~/components/ExplorerQuery/ExplorerQuery';

import RecentMessages from './RecentMessages';
import DailyCountLineChart from './DailyCountLineChart';
import { NetworkContext } from '~/components/NetworkSelect';

import { ReactComponent as BinanceChainIcon } from '~/icons/binancechain.svg';
import { ReactComponent as EthereumIcon } from '~/icons/ethereum.svg';
import { ReactComponent as SolanaIcon } from '~/icons/solana.svg';
import { ReactComponent as TerraIcon } from '~/icons/terra.svg';
import { ReactComponent as PolygonIcon } from '~/icons/polygon.svg';
import ChainOverviewCard from './ChainOverviewCard';
import { ChainID } from '~/utils/misc/constants';
import { contractNameFormatter } from './utils';

export interface Totals {
    LastDayCount: { [groupByKey: string]: number }
    TotalCount: { [groupByKey: string]: number }
    DailyTotals: {
        // "2021-08-22": { "*": 0 },
        [date: string]: { [groupByKey: string]: number }
    }
}
// type GroupByKey = "*" | "emitterChain" | "emitterChain:emitterAddress"
export interface Recent {
    [groupByKey: string]: Array<BigTableMessage>
}
type GroupBy = undefined | "chain" | "address"
type ForChain = undefined | StatsProps["emitterChain"]
type ForAddress = undefined | StatsProps["emitterAddress"]


interface StatsProps {
    emitterChain?: number,
    emitterAddress?: string
}

const Stats: React.FC<StatsProps> = ({ emitterChain, emitterAddress }) => {
    const intl = useIntl()
    const { activeNetwork } = useContext(NetworkContext)

    const [totals, setTotals] = useState<Totals>()
    const [recent, setRecent] = useState<Recent>()
    const [polling, setPolling] = useState(true);
    const [lastFetched, setLastFetched] = useState<number>()
    const [pollInterval, setPollInterval] = useState<NodeJS.Timeout>()
    const [controller, setController] = useState<AbortController>(new AbortController())

    const daysSinceDataStart = 30

    const fetchTotals = (baseUrl: string, groupBy: GroupBy, forChain: ForChain, forAddress: ForAddress, signal: AbortSignal) => {
        const totalsUrl = `${baseUrl}/totals`
        let url = `${totalsUrl}?numDays=${daysSinceDataStart}`
        if (groupBy) { url = `${url}&groupBy=${groupBy}` }
        if (forChain) { url = `${url}&forChain=${forChain}` }
        if (forAddress) { url = `${url}&forAddress=${forAddress}` }

        return fetch(url, { signal })
            .then<Totals>(res => {
                if (res.ok) return res.json()
                // throw an error with specific message, rather than letting the json decoding throw.
                throw 'explorer.stats.failedFetchingTotals'
            })
            .then(result => {
                setTotals(result)
                setLastFetched(Date.now())

            }, error => {
                if (error.name !== "AbortError") {
                    //  handle errors here instead of a catch(), so that we don't swallow exceptions from components
                    console.error('failed fetching totals. error: ', error)
                }
            })
    }
    const fetchRecent = (baseUrl: string, groupBy: GroupBy, forChain: ForChain, forAddress: ForAddress, signal: AbortSignal) => {
        const recentUrl = `${baseUrl}/recent`
        let url = `${recentUrl}?numRows=24`
        if (groupBy) { url = `${url}&groupBy=${groupBy}` }
        if (forChain) { url = `${url}&forChain=${forChain}` }
        if (forAddress) { url = `${url}&forAddress=${forAddress}` }

        return fetch(url, { signal })
            .then<Recent>(res => {
                if (res.ok) return res.json()
                // throw an error with specific message, rather than letting the json decoding throw.
                throw 'explorer.stats.failedFetchingRecent'
            })
            .then(result => {
                setRecent(result)
                setLastFetched(Date.now())

            }, error => {
                if (error.name !== "AbortError") {
                    //  handle errors here instead of a catch(), so that we don't swallow exceptions from components
                    console.error('failed fetching recent. error: ', error)
                }
            })
    }

    const getData = (props: StatsProps, baseUrl: string, signal: AbortSignal) => {
        let forChain: ForChain = undefined
        let forAddress: ForAddress = undefined
        let recentGroupBy: GroupBy = undefined
        let totalsGroupBy: GroupBy = "chain"
        if (props.emitterChain) {
            forChain = props.emitterChain
            totalsGroupBy = 'address'
            recentGroupBy = 'address'
        }
        if (props.emitterChain && props.emitterAddress) {
            forAddress = props.emitterAddress
        }
        return Promise.all([
            fetchTotals(baseUrl, totalsGroupBy, forChain, forAddress, signal),
            fetchRecent(baseUrl, recentGroupBy, forChain, forAddress, signal)
        ])
    }

    const pollingController = (emitterChain: StatsProps["emitterChain"], emitterAddress: StatsProps["emitterAddress"], baseUrl: string) => {
        // clear any ongoing intervals
        if (pollInterval) {
            clearInterval(pollInterval)
            setPollInterval(undefined)
        }
        if (polling) {
            // abort any in-flight requests
            controller.abort()
            // create a new controller for the new fetches, add it to state
            const newController = new AbortController();
            setController(newController)
            // create a signal for requests
            const { signal } = newController;
            // start polling
            let interval = setInterval(() => {
                getData({ emitterChain, emitterAddress }, baseUrl, signal).catch(err => {
                    console.error('failed fetching data. err: ', err)
                })
            }, 5000)
            setPollInterval(interval)
        }
    }

    useEffect(() => {
        controller.abort()
        setTotals(undefined)
        setRecent(undefined)
        pollingController(emitterChain, emitterAddress, activeNetwork.endpoints.bigtableFunctionsBase)
    }, [emitterChain, emitterAddress, activeNetwork.endpoints.bigtableFunctionsBase])

    useEffect(() => {
        return function cleanup() {
            if (pollInterval) {
                clearInterval(pollInterval)
            }
        };
    }, [polling, pollInterval, activeNetwork.endpoints.bigtableFunctionsBase])

    let title = "Recent messages"
    let hideTableTitles = false
    if (emitterChain) {
        title = `Recent ${ChainID[Number(emitterChain)]} messages`
    }
    if (emitterChain && emitterAddress) {
        title = `Recent ${contractNameFormatter(emitterAddress, emitterChain, activeNetwork)} messages`
        hideTableTitles = true
    }

    return (
        <>
            {!emitterChain && !emitterAddress &&
                <div style={{ display: 'flex', justifyContent: 'space-around', alignItems: 'flex-end', flexWrap: 'wrap', marginBottom: 40 }}>
                    <ChainOverviewCard totalDays={daysSinceDataStart} totals={totals} dataKey="1" title={ChainID[1]} Icon={SolanaIcon} iconStyle={{ height: 120, margin: '10px 0' }} />
                    <ChainOverviewCard totalDays={daysSinceDataStart} totals={totals} dataKey="2" title={ChainID[2]} Icon={EthereumIcon} />
                    <ChainOverviewCard totalDays={daysSinceDataStart} totals={totals} dataKey="3" title={ChainID[3]} Icon={TerraIcon} />
                    <ChainOverviewCard totalDays={daysSinceDataStart} totals={totals} dataKey="4" title={ChainID[4]} Icon={BinanceChainIcon} />
                    <ChainOverviewCard totalDays={daysSinceDataStart} totals={totals} dataKey="5" title={ChainID[5]} Icon={PolygonIcon} />
                </div>
            }
            <Spin spinning={!totals && !recent} style={{ width: '100%', height: 500 }} />
            <div>
                {totals?.DailyTotals &&
                    <DailyCountLineChart
                        dailyCount={totals?.DailyTotals}
                        lastFetched={lastFetched}
                        title="messages/day"
                        emitterChain={emitterChain}
                        emitterAddress={emitterAddress}
                    />}
            </div>

            {recent && <RecentMessages recent={recent} lastFetched={lastFetched} title={title} hideTableTitles={hideTableTitles} />}

        </>
    )
}

export default Stats
