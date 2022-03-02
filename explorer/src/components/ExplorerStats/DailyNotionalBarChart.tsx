import React, { useEffect, useState } from 'react';
import { NotionalTransferredTo } from './ExplorerStats';
import { Typography } from '@mui/material';
import { ResponsiveBar, BarDatum } from '@nivo/bar'

import { makeDate, makeGroupName, chainColors, amountFormatter, usdFormatter, chainIdColors } from "../../utils/explorer"
import { useNetworkContext } from '../../contexts/NetworkContext';
import { chainEnums, } from '../../utils/consts';

function findMaxBoundOfIQR(array: number[]): number {

    if (array.length < 4) {
        array.sort((a, b) => a - b)
        return array[0]
    }

    let values, q1, q3, iqr, maxValue: number

    values = array.slice().sort((a, b) => a - b);//copy array fast and sort

    if ((values.length / 4) % 1 === 0) {//find quartiles
        q1 = 1 / 2 * (values[(values.length / 4)] + values[(values.length / 4) + 1]);
        q3 = 1 / 2 * (values[(values.length * (3 / 4))] + values[(values.length * (3 / 4)) + 1]);
    } else {
        q1 = values[Math.floor(values.length / 4 + 1)];
        q3 = values[Math.ceil(values.length * (3 / 4) + 1)];
    }

    iqr = q3 - q1;
    maxValue = q3 + iqr * 1.5;

    return maxValue
}

interface DailyCountProps {
    daily: NotionalTransferredTo["Daily"]
}

const DailyNotionalBarChart = (props: DailyCountProps) => {

    const { activeNetwork } = useNetworkContext()
    const [barData, setBarData] = useState<Array<BarDatum>>([])
    const [max, setMax] = useState<number>()

    useEffect(() => {

        // create a list of all data points in order to calculate min/max bounds of chart
        const all: number[] = []

        const data = Object.keys(props.daily).reduce<Array<BarDatum>>((accum, date) => {
            const chains = props.daily[date]

            return [...accum, Object.keys(chains).reduce<BarDatum>((subAccum, chain) => {

                const group = makeGroupName(chain, activeNetwork)
                // const group = chain
                all.push(chains[chain]["*"])
                return {
                    ...subAccum,
                    [group]: chains[chain]["*"],
                }

            }, { "date": date })]
        }, [])

        // create a max value for the y axis, in order to exclude outliers so the chart looks nice.
        let max = findMaxBoundOfIQR(all)
        setMax(max)

        setBarData(data)

    }, [props.daily, activeNetwork])

    const keys = chainEnums.slice(1)
    const today = new Date().toISOString().slice(0, 10)

    return (
        <div style={{ height: 400, minWidth: 360, flex: '1', marginBottom: 40 }}>
            <Typography variant="h4" style={{ marginLeft: 20 }}>value received (USD)</Typography>

            <ResponsiveBar
                theme={{ textColor: "rgba(255, 255, 255, 0.85)" }}
                colors={chainIdColors.slice(1)}
                data={barData}
                keys={keys}
                enableLabel={false}
                groupMode="grouped"
                indexBy="date"
                margin={{
                    top: 10,
                    right: 0,
                    bottom: 24,
                    left: 40,
                }}
                padding={0.3}
                valueScale={{ type: 'linear', max }}
                indexScale={{ type: 'band', round: true }}
                borderColor={{ from: 'color', modifiers: [['darker', 1.6]] }}
                axisTop={null}
                axisRight={null}
                axisLeft={{
                    format: (value) => amountFormatter(Number(value))
                }}
                axisBottom={{
                    format: (value) => {
                        if (value === today) {
                            return "today"
                        }
                        return makeDate(value)
                    }
                }}
                tooltip={(data) => {
                    let { id, value, indexValue, } = data
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
                            <Typography variant="subtitle1" style={{ color: 'rgba(255, 255, 255, 0.85)' }}>{id} - {makeDate(String(indexValue))}</Typography>
                            <div
                                style={{
                                    display: 'flex',
                                    padding: '3px 0',
                                    justifyContent: 'flex-end',
                                }}
                            >
                                {usdFormatter.format(Number(value))} received
                            </div>

                        </div>
                    )
                }}
            />
        </div>
    )
}

export default DailyNotionalBarChart
