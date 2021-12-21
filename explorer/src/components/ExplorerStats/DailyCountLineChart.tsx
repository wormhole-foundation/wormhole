import React, { useContext, useEffect, useState } from 'react';
import { Totals } from './ExplorerStats';
import { FormattedMessage, useIntl } from 'gatsby-plugin-intl'
import { Typography } from 'antd'
const { Title } = Typography
import ReactTimeAgo from 'react-time-ago'
import { ResponsiveLine, Serie } from '@nivo/line'


import { makeDate, makeGroupName, chainColors } from "./utils"
import { titleStyles } from '~/styles';
import { NetworkContext } from '../NetworkSelect';


interface DailyCountProps {
    dailyCount: Totals["DailyTotals"]
    lastFetched?: number
    title: string,
    emitterChain?: number,
    emitterAddress?: string
}

const DailyCountLineChart = (props: DailyCountProps) => {
    const intl = useIntl()
    const { activeNetwork } = useContext(NetworkContext)
    const [data, setData] = useState<Array<Serie>>([])
    const colors = [
        "hsl(9, 100%, 61%)",
        "hsl(30, 100%, 61%)",
        "hsl(54, 100%, 61%)",
        "hsl(82, 100%, 61%)",
        "hsl(114, 100%, 61%)",
        "hsl(176, 100%, 61%)",
        "hsl(224, 100%, 61%)",
        "hsl(270, 100%, 61%)",
        "hsl(320, 100%, 61%)",
        "hsl(360, 100%, 61%)",
    ]

    useEffect(() => {
        const datum = Object.keys(props.dailyCount).reduce<{ [groupKey: string]: Serie }>((accum, key) => {
            const vals = props.dailyCount[key]
            const subKeyColors: { [key: string]: string } = {}

            return Object.keys(vals).reduce<{ [groupKey: string]: Serie }>((subAccum, subKey) => {
                if (props.emitterAddress && subKey === "*") {
                    // if this chart is for a single emitterAddress, no need for "all messages" line.
                    return subAccum
                }
                const group = makeGroupName(subKey, activeNetwork, props.emitterChain)

                if (!(group in subAccum)) {
                    // first time this group has been seen
                    subAccum[group] = { id: group, data: [] }
                    if (subKey in chainColors) {
                        subAccum[group].color = chainColors[subKey]
                    } else {

                        if (!(subKey in subKeyColors)) {
                            let len = Object.keys(subKeyColors).length
                            subKeyColors[subKey] = colors[len]
                        }
                        subAccum[group].color = subKeyColors[subKey]
                    }
                }

                subAccum[group].data.push({
                    "y": vals[subKey],
                    "x": makeDate(key)
                })
                return subAccum
            }, accum)
        }, {})

        setData(Object.values(datum))
    }, [props.dailyCount, props.lastFetched, props.emitterChain, props.emitterAddress, activeNetwork])

    const dateLabel = [{
        id: "label",
        label: "Dates are UTC"
    }]


    return (
        <div style={{ flexGrow: 1, height: 500, width: '100%' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                <Title level={3} style={{ ...titleStyles, marginLeft: 20 }}>{props.title}</Title>
                {props.lastFetched ? (
                    <div style={{ marginRight: 40 }}>
                        <FormattedMessage id="explorer.lastUpdated" />:&nbsp;
                        <ReactTimeAgo date={new Date(props.lastFetched)} locale={intl.locale} timeStyle="twitter" />
                    </div>

                ) : null}
            </div>
            <ResponsiveLine
                theme={{ textColor: "rgba(255, 255, 255, 0.85)", fontSize: 12, legends: { text: { fontSize: 16 } } }}
                colors={({ color }) => color}
                data={data}
                curve={"monotoneX"}
                margin={{ top: 20, right: 40, bottom: 160, left: 60 }}
                xScale={{ type: 'point' }}
                yScale={{
                    type: 'symlog',
                    constant: 400,
                    max: 'auto',
                    min: 0,
                }}
                enableGridX={false}
                axisTop={null}
                axisRight={null}
                axisBottom={null}
                axisLeft={{
                    tickPadding: 5,
                    tickRotation: 0,
                }}
                pointSize={4}
                pointColor={{ theme: 'background' }}
                pointBorderWidth={2}
                pointBorderColor={{ from: 'serieColor' }}
                pointLabelYOffset={-12}
                useMesh={true}
                enableSlices={"x"}
                isInteractive={true}
                legends={[
                    {
                        anchor: 'bottom-right',
                        direction: 'column',
                        justify: false,
                        translateX: -20,
                        translateY: (30 + data.length * 20),
                        itemsSpacing: 10,
                        itemDirection: 'right-to-left',
                        itemWidth: 400,
                        itemHeight: 16,
                        itemOpacity: 0.85,
                        symbolSize: 12,
                        symbolShape: 'circle',
                        symbolBorderColor: 'rgba(0, 0, 0, .5)',
                        effects: [
                            {
                                on: 'hover',
                                style: {
                                    itemBackground: 'rgba(0, 0, 0, .03)',
                                    itemOpacity: 1
                                }
                            }
                        ]
                    },
                    {
                        anchor: 'bottom-left',
                        direction: 'column',
                        justify: false,
                        translateX: 0,
                        translateY: 40,
                        itemsSpacing: 10,
                        itemDirection: 'left-to-right',
                        itemWidth: 60,
                        itemHeight: 16,
                        itemOpacity: 0.85,
                        data: dateLabel,


                    },
                ]}
                sliceTooltip={({ slice }) => {
                    return (
                        <div
                            style={{
                                background: '#010114',
                                padding: '9px 12px',
                                border: '1px solid rgba(255, 255, 255, 0.85)',
                                color: "rgba(255, 255, 255, 0.85)",
                                fontSize: 14
                            }}
                        >
                            <Title level={4} style={{ color: 'rgba(255, 255, 255, 0.85)' }}>{slice.points[0].data.xFormatted}</Title>
                            {slice.points.map(point => (
                                <div
                                    key={point.id}
                                    style={{
                                        display: 'flex',
                                        padding: '3px 0',
                                    }}
                                >
                                    <div style={{ background: point.serieColor, height: 16, width: 16, }} />&nbsp;
                                    <span>{point.serieId}</span>&nbsp;-&nbsp;{point.data.yFormatted}
                                </div>
                            ))}
                        </div>
                    )
                }}
            />
        </div>
    )
}

export default DailyCountLineChart
