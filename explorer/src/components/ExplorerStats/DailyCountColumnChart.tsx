import React, { useEffect, useState } from 'react';
import { Totals } from './ExplorerStats';
import { useIntl, FormattedMessage } from 'gatsby-plugin-intl'
import { ResponsiveBar, BarDatum } from '@nivo/bar'
import { makeDate, makeGroupName } from "./utils"


interface DailyCountProps {
    dailyCount: Totals["DailyTotals"]
}

const DailyCountColumnChart = (props: DailyCountProps) => {
    const intl = useIntl()
    const [data, setData] = useState<Array<BarDatum>>([])

    useEffect(() => {
        const datum = Object.keys(props.dailyCount).reduce<Array<BarDatum>>((accum, key) => {
            const val = props.dailyCount[key]
            return [...accum, Object.keys(val).reduce<BarDatum>((subAccum, subKey) => {

                const group = makeGroupName(subKey)
                return {
                    ...subAccum,
                    [group]: val[subKey],
                }

                // "SolanaColor": "hsl(259, 70%, 50%)", "EthereumColor": "hsl(43, 70%, 50%)", "BSCColor": "hsl(164, 70%, 50%)", "allColor": "hsl(345, 70%, 50%)"
            }, { "date": makeDate(key) })]
        }, [])
        // console.log('bar datum: ', datum)

        // TODO - create a dynamic list of keys
        setData(datum)
    }, [props.dailyCount])



    return (
        <div style={{ flexGrow: 1, width: '100%', height: 400, color: 'rgba(0, 0, 0, 0.85)' }}>
            <h2>daily totals</h2>

            <ResponsiveBar
                theme={{ textColor: "rgba(255, 255, 255, 0.85)" }}
                data={data}
                keys={["All Messages", "Solana", "Ethereum", "BSC"]}
                groupMode="grouped"
                indexBy="date"
                margin={{
                    top: 50,
                    right: 130,
                    bottom: 50,
                    left: 60
                }}
                padding={0.3}
                valueScale={{ type: 'linear' }}
                indexScale={{ type: 'band', round: true }}
                colors={{ scheme: 'category10' }}
                borderColor={{ from: 'color', modifiers: [['darker', 1.6]] }}
                axisTop={null}
                axisRight={null}
                axisBottom={{
                    tickSize: 5,
                    tickPadding: 5,
                    tickRotation: 0,
                    legend: 'date',
                    legendPosition: 'middle',
                    legendOffset: 32
                }}
                axisLeft={{
                    tickSize: 5,
                    tickPadding: 5,
                    tickRotation: 0,
                    legend: 'messages',
                    legendPosition: 'middle',
                    legendOffset: -40
                }}
                labelSkipWidth={12}
                labelSkipHeight={12}
                labelTextColor={{ from: 'color', modifiers: [['darker', 1.6]] }}
                legends={[
                    {
                        dataFrom: 'keys',
                        anchor: 'bottom-right',
                        direction: 'column',
                        justify: false,
                        translateX: 120,
                        translateY: 0,
                        itemsSpacing: 2,
                        itemWidth: 100,
                        itemHeight: 20,
                        itemDirection: 'left-to-right',
                        itemOpacity: 0.85,
                        symbolSize: 20,
                        effects: [
                            {
                                on: 'hover',
                                style: {
                                    itemOpacity: 1
                                }
                            }
                        ]
                    }
                ]}
            // tooltip={(props) => {
            //     console.log(props)
            //     // formattedValue: "21"
            //     // height: 114
            //     // hidden: false
            //     // id: "Ethereum"
            //     // index: 29
            //     // indexValue: "09/21"
            //     // label: "Ethereum - 09/21"
            //     return <div>tooltip</div>
            // }}
            />
        </div>
    )
}

export default DailyCountColumnChart
