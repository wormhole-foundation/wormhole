import React, { useState, useEffect } from 'react'

import { Box, Card, Typography, } from '@mui/material'


import { navigate } from 'gatsby'
import { NotionalTransferred, NotionalTransferredToCumulative, Totals } from './ExplorerStats'
import { amountFormatter } from '../../utils/explorer'
import { chainIDStrings } from '../../utils/consts'

interface ChainOverviewCardProps {
    icon: string
    title: string
    dataKey: keyof typeof chainIDStrings
    totals?: Totals
    iconStyle?: { [key: string]: string | number }
    notionalTransferred?: NotionalTransferred
    notionalTransferredToCumulative?: NotionalTransferredToCumulative
    imgOffsetRightMd?: string
    imgOffsetTopXs?: string
    imgOffsetTopMd?: string
    imgPaddingBottomXs?: number
    imgPaddingBottomMd?: number
}

const ChainOverviewCard: React.FC<ChainOverviewCardProps> = ({
    icon,
    iconStyle,
    title,
    dataKey,
    totals,
    notionalTransferred,
    notionalTransferredToCumulative,
    imgOffsetRightMd = "-16px",
    imgOffsetTopXs = "-30px",
    imgOffsetTopMd = "-16px",
    imgPaddingBottomXs = 0,
    imgPaddingBottomMd = 0,

}) => {
    const [totalCount, setTotalColunt] = useState<number>()
    const [animate, setAnimate] = useState<boolean>(false)

    useEffect(() => {
        // hold values from props in state, so that we can detect changes and add animation class
        setTotalColunt(totals?.TotalCount[dataKey])

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

    const centerStyles: any = { display: 'flex', justifyContent: "flex-start", alignItems: 'center', flexDirection: "column" }
    return (
        <Card
            variant="outlined"
            onClick={() => navigate(`/explorer/?emitterChain=${dataKey}`)}
            sx={{
                backgroundColor: "rgba(255,255,255,.07)",
                backgroundImage: "none",
                borderRadius: "28px",
                display: "flex",
                flexDirection: "column",
                height: "100%",
                overflow: "visible",
                cursor: "pointer",
            }}
        >
            <Box
                sx={{
                    textAlign: { xs: "center", md: "right" },
                    position: "relative",
                    right: { xs: null, md: imgOffsetRightMd },
                    top: { xs: imgOffsetTopXs, md: imgOffsetTopMd },
                    pb: { xs: imgPaddingBottomXs, md: imgPaddingBottomMd },
                    zIndex: 1,
                }}
            >
                <img src={icon} alt="" style={{ height: 140, ...iconStyle }} />
            </Box>
            <div style={centerStyles}>
                <Typography variant="h4">{title}</Typography>
            </div>
            <>
                <div style={{ ...centerStyles, gap: 8 }}>

                    {notionalTransferredToCumulative && notionalTransferredToCumulative.AllTime &&
                        <div style={centerStyles}>
                            <div>
                                <Typography variant="h5" className={animate ? "highlight-new-val" : ""}>
                                    ${amountFormatter(notionalTransferredToCumulative.AllTime[dataKey]["*"])}
                                </Typography>
                            </div>
                            <div style={{ marginTop: -10 }}><Typography variant="subtitle1">received</Typography></div>
                        </div>
                    }
                    {notionalTransferred &&
                        notionalTransferred.WithinPeriod &&
                        dataKey in notionalTransferred.WithinPeriod &&
                        "*" in notionalTransferred.WithinPeriod[dataKey] &&
                        "*" in notionalTransferred.WithinPeriod[dataKey]["*"] &&
                        notionalTransferred.WithinPeriod[dataKey]["*"]["*"] > 0 ?
                        <div style={centerStyles}>

                            <div>
                                <Typography variant="h5" className={animate ? "highlight-new-val" : ""}>
                                    {notionalTransferred.WithinPeriod[dataKey]["*"]["*"] ?
                                        "$" + amountFormatter(notionalTransferred.WithinPeriod[dataKey]["*"]["*"]) : "..."
                                    }
                                </Typography>
                            </div>
                            <div style={{ marginTop: -10 }}>
                                <Typography variant="subtitle1">sent</Typography>
                            </div>
                        </div> :
                        <div style={centerStyles}>
                            <div style={{ marginTop: -10 }}>
                                <Typography variant="body1">amount sent<br />coming soon</Typography>
                            </div>
                        </div>
                    }
                    {!!totalCount &&
                        <div style={centerStyles}>
                            <div>
                                <Typography variant="h5" className={animate ? "highlight-new-val" : ""}>
                                    {amountFormatter(totalCount)}
                                </Typography>
                            </div>
                            <div style={{ marginTop: -10 }}>
                                <Typography variant="subtitle1"> messages </Typography></div>
                        </div>
                    }
                </div>
            </>

            {totalCount === 0 && <Typography variant="h6">coming soon</Typography>}
        </Card>
    )
}

export default ChainOverviewCard
