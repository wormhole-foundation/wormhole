import {
  Button,
  Checkbox,
  CircularProgress,
  FormControl,
  ListItemText,
  makeStyles,
  MenuItem,
  Paper,
  Select,
  TextField,
  Tooltip,
  Typography,
  withStyles,
} from "@material-ui/core";
import { ToggleButton, ToggleButtonGroup } from "@material-ui/lab";
import { useCallback, useMemo, useState } from "react";
import TVLAreaChart from "./Charts/TVLAreaChart";
import useCumulativeTVL from "../../hooks/useCumulativeTVL";
import { TIME_FRAMES } from "./Charts/TimeFrame";
import TVLLineChart from "./Charts/TVLLineChart";
import { ChainInfo, CHAINS_BY_ID } from "../../utils/consts";
import { ChainId } from "@certusone/wormhole-sdk";
import { COLORS } from "../../muiTheme";
import TVLBarChart from "./Charts/TVLBarChart";
import TVLTable from "./Charts/TVLTable";
import useTVL from "../../hooks/useTVL";
import { ArrowBack, InfoOutlined } from "@material-ui/icons";

const useStyles = makeStyles((theme) => ({
  description: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    marginBottom: "16px",
    [theme.breakpoints.down("xs")]: {
      flexDirection: "column",
    },
  },
  displayBy: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    flexWrap: "wrap",
    marginBottom: "16px",
    [theme.breakpoints.down("xs")]: {
      justifyContent: "center",
      columnGap: 8,
      rowGap: 8,
    },
  },
  mainPaper: {
    display: "flex",
    flexDirection: "column",
    backgroundColor: COLORS.whiteWithTransparency,
    padding: "2rem",
    marginBottom: theme.spacing(8),
    borderRadius: 8,
  },
  toggleButton: {
    textTransform: "none",
  },
  tooltip: {
    margin: 8,
  },
  alignCenter: {
    margin: "0 auto",
    display: "block",
  },
}));

const tooltipStyles = {
  tooltip: {
    minWidth: "max-content",
    borderRadius: "4px",
    backgroundColor: "#5EA1EC",
    color: "#0F0C48",
    fontSize: "14px",
  },
};

const StyledTooltip = withStyles(tooltipStyles)(Tooltip);

const DISPLAY_BY_VALUES = ["Time", "Chain"];

const TVLStats = () => {
  const classes = useStyles();

  const [displayBy, setDisplayBy] = useState(DISPLAY_BY_VALUES[0]);
  const [timeFrame, setTimeFrame] = useState("All time");

  const [selectedChains, setSelectedChains] = useState<ChainId[]>([]);

  const [selectedChainDetail, setSelectedChainDetail] =
    useState<ChainInfo | null>(null);

  const cumulativeTVL = useCumulativeTVL();
  const tvl = useTVL();

  const tvlAllTime = useMemo(() => {
    return tvl.data
      ? new Intl.NumberFormat("en-US", {
          style: "currency",
          currency: "USD",
          maximumFractionDigits: 0,
        }).format(
          tvl.data.AllTime[selectedChainDetail?.id || "*"]["*"].Notional || 0
        )
      : "";
  }, [selectedChainDetail, tvl]);

  const availableChains = useMemo(() => {
    const chainIds = cumulativeTVL.data
      ? Object.keys(
          Object.values(cumulativeTVL.data.DailyLocked)[0] || {}
        ).reduce<ChainId[]>((chainIds, key) => {
          if (key !== "*") {
            const chainId = parseInt(key) as ChainId;
            if (CHAINS_BY_ID[chainId]) {
              chainIds.push(chainId);
            }
          }
          return chainIds;
        }, [])
      : [];
    setSelectedChains(chainIds);
    return chainIds;
  }, [cumulativeTVL]);

  const handleDisplayByChange = useCallback((event, nextValue) => {
    if (nextValue) {
      setDisplayBy(nextValue);
    }
  }, []);

  const handleTimeFrameChange = useCallback(
    (event) => setTimeFrame(event.target.value),
    []
  );

  const handleSelectedChainsChange = useCallback(
    (event) => {
      const value = event.target.value;
      if (value[value.length - 1] === "all") {
        setSelectedChains((prevValue) =>
          prevValue.length === availableChains.length ? [] : availableChains
        );
      } else {
        setSelectedChains(value);
      }
    },
    [availableChains]
  );

  const handleChainDetailSelected = useCallback((chainInfo: ChainInfo) => {
    setSelectedChainDetail(chainInfo);
  }, []);

  const allChainsSelected = selectedChains.length === availableChains.length;
  const tvlText =
    "Total Value Locked" +
    (selectedChainDetail ? ` on ${selectedChainDetail?.name}` : "");
  const tooltipText = selectedChainDetail
    ? `Total Value Locked on ${selectedChainDetail?.name}`
    : "USD equivalent value of all assets locked in Portal";

  return (
    <>
      <div className={classes.description}>
        <Typography variant="h3">
          {tvlText}
          <StyledTooltip title={tooltipText} className={classes.tooltip}>
            <InfoOutlined />
          </StyledTooltip>
        </Typography>
        <Typography variant="h3">{tvlAllTime}</Typography>
      </div>
      <div className={classes.displayBy}>
        {!selectedChainDetail ? (
          <div>
            <Typography display="inline" style={{ marginRight: "8px" }}>
              Display by
            </Typography>
            <ToggleButtonGroup
              value={displayBy}
              exclusive
              onChange={handleDisplayByChange}
            >
              {DISPLAY_BY_VALUES.map((value) => (
                <ToggleButton
                  key={value}
                  value={value}
                  className={classes.toggleButton}
                >
                  {value}
                </ToggleButton>
              ))}
            </ToggleButtonGroup>
          </div>
        ) : null}
        {displayBy === "Time" && !selectedChainDetail ? (
          <div>
            <FormControl>
              <Select
                multiple
                variant="outlined"
                value={selectedChains}
                onChange={handleSelectedChainsChange}
                renderValue={(selected: any) =>
                  selected.length === availableChains.length
                    ? "All chains"
                    : selected.length > 1
                    ? `${selected.length} chains`
                    : //@ts-ignore
                      CHAINS_BY_ID[selected[0]]?.name
                }
                MenuProps={{ getContentAnchorEl: null }} // hack to prevent popup menu from moving
                style={{ minWidth: 128 }}
              >
                <MenuItem value="all">
                  <Checkbox
                    checked={availableChains.length > 0 && allChainsSelected}
                    indeterminate={
                      selectedChains.length > 0 &&
                      selectedChains.length < availableChains.length
                    }
                  />
                  <ListItemText primary="All chains" />
                </MenuItem>
                {availableChains.map((option) => (
                  <MenuItem key={option} value={option}>
                    <Checkbox checked={selectedChains.indexOf(option) > -1} />
                    <ListItemText primary={CHAINS_BY_ID[option]?.name} />
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <TextField
              select
              variant="outlined"
              value={timeFrame}
              onChange={handleTimeFrameChange}
              style={{ marginLeft: 8 }}
            >
              {Object.keys(TIME_FRAMES).map((timeFrame) => (
                <MenuItem key={timeFrame} value={timeFrame}>
                  {timeFrame}
                </MenuItem>
              ))}
            </TextField>
          </div>
        ) : selectedChainDetail ? (
          <Button
            startIcon={<ArrowBack />}
            onClick={() => {
              setSelectedChainDetail(null);
            }}
          >
            Back to all chains
          </Button>
        ) : null}
      </div>
      <Paper className={classes.mainPaper}>
        {displayBy === "Time" ? (
          cumulativeTVL.data ? (
            allChainsSelected ? (
              <TVLAreaChart
                cumulativeTVL={cumulativeTVL.data}
                timeFrame={TIME_FRAMES[timeFrame]}
              />
            ) : (
              <TVLLineChart
                cumulativeTVL={cumulativeTVL.data}
                timeFrame={TIME_FRAMES[timeFrame]}
                selectedChains={selectedChains}
              />
            )
          ) : (
            <CircularProgress className={classes.alignCenter} />
          )
        ) : tvl.data ? (
          selectedChainDetail ? (
            <TVLTable chainInfo={selectedChainDetail} tvl={tvl.data} />
          ) : (
            <TVLBarChart
              tvl={tvl.data}
              onChainSelected={handleChainDetailSelected}
            />
          )
        ) : (
          <CircularProgress className={classes.alignCenter} />
        )}
      </Paper>
    </>
  );
};

export default TVLStats;
