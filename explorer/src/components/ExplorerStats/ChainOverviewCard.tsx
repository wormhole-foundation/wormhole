import React, { useState, useEffect } from 'react'

import { Card, Statistic, Tooltip, Typography, } from 'antd'
const { Text } = Typography

import { useIntl, FormattedMessage } from 'gatsby-plugin-intl'
import { navigate } from 'gatsby'
import { Totals } from './ExplorerStats'
import './ChainOverviewCard.less'

interface ChainOverviewCardProps {
    Icon: React.FC<React.SVGProps<SVGSVGElement>>
    title: string
    dataKey: "*" | "1" | "2" | "3" | "4" | "5"
    totals?: Totals
    iconStyle?: { [key: string]: string | number }
    totalDays: number
}

const ChainOverviewCard: React.FC<ChainOverviewCardProps> = ({ Icon, iconStyle, title, dataKey, totals, totalDays }) => {
    const intl = useIntl()
    const [lastDayCount, setLastDayColunt] = useState<number>()
    const [totalCount, setTotalColunt] = useState<number>()
    const [loading, setLoading] = useState<boolean>(true)
    const [animate, setAnimate] = useState<boolean>(false)

    useEffect(() => {
        if (!totals) {
            setLoading(true)
        }
        // hold values from props in state, so that we can detect changes and add animation class
        setLastDayColunt(totals?.LastDayCount[dataKey])
        setTotalColunt(totals?.TotalCount[dataKey])

        if (totals?.TotalCount && dataKey in totals?.TotalCount) {
            setLoading(!totals?.TotalCount[dataKey] && !totals?.LastDayCount[dataKey])
        }

        let timeout: NodeJS.Timeout
        if (totals?.LastDayCount[dataKey] && totalCount !== totals?.LastDayCount[dataKey]) {
            setAnimate(true)
            timeout = setTimeout(() => {
                setAnimate(false)
            }, 2000)
        }
        return function cleanup() {
            if (timeout) {
                clearTimeout(timeout)
            }
        }
    }, [totals?.TotalCount[dataKey], totals?.LastDayCount[dataKey], dataKey, totalCount])

    useEffect(() => {
        // for chains that do not have a key in the bigtable result, no messages have been seen yet.
        if (totals && "TotalCount" in totals && !(dataKey in totals?.TotalCount)) {
            // if we have TotalCount, but the dataKey is not in it, no transactions for this chain
            setLoading(false)
        } else if (!totals) {
            setLoading(true)
        }
    }, [totals?.TotalCount, dataKey])
    return (
        <Tooltip title={!!totalCount ?
            intl.formatMessage({ id: "explorer.clickToView" }) :
            loading ? "loading" : intl.formatMessage({ id: "explorer.comingSoon" })}>
            <Card
                style={{
                    width: 190,
                    paddingTop: 10,
                }}
                className="hover-z-index"
                cover={<Icon style={{ height: 140, ...iconStyle }} />}
                hoverable={!!totalCount}
                bordered={false}
                onClick={() => !!totalCount && navigate(`/${intl.locale}/explorer/?emitterChain=${dataKey}`)}
                loading={loading}
                bodyStyle={{
                    display: 'flex',
                    flexDirection: 'column',
                    justifyContent: 'center',
                    alignItems: 'center',

                }}
            >
                <Card.Meta title={title} style={{ margin: '12px 0' }} />
                {!!totalCount ? (
                    <>
                        <div style={{ display: 'flex', justifyContent: "space-between", alignItems: 'center', gap: 12 }}>
                            <div><Text type="secondary" style={{ fontSize: 14 }}>last&nbsp;24&nbsp;hours</Text></div>
                            <div><Text className={animate ? "highlight-new-val" : ""} style={{ fontSize: 26 }}>{lastDayCount}</Text></div>
                        </div>
                        <div style={{ display: 'flex', justifyContent: "center", alignItems: 'center', gap: 12 }}>
                            <div><Text type="secondary" style={{ fontSize: 14 }}>last&nbsp;{totalDays}&nbsp;days</Text></div>
                            <div><Text className={animate ? "highlight-new-val" : ""} style={{ fontSize: 26 }}>{totalCount}</Text></div>
                        </div>
                        {/* <Statistic title={<span>last 24 hours</span>} value={totals?.LastDayCount[dataKey]} style={{ display: 'flex', justifyContent: "space-between", alignItems: 'center', gap: 12 }} valueStyle={{ fontSize: 26 }} /> */}
                        {/* <Statistic title={<span>last {totalDays} days</span>} value={totals?.TotalCount[dataKey]} style={{ display: 'flex', justifyContent: "center", alignItems: 'center', gap: 12, }} valueStyle={{ fontSize: 26 }} /> */}
                    </>
                ) : <Text type="secondary" style={{ height: 86, fontSize: 14 }}>{intl.formatMessage({ id: "explorer.comingSoon" })}</Text>}
            </Card>

        </Tooltip>
    )
}

export default ChainOverviewCard
