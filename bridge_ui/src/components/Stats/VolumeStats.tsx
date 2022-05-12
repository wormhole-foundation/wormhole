import { ChainId } from "@certusone/wormhole-sdk";
import {
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
import { InfoOutlined } from "@material-ui/icons";
import { ToggleButton, ToggleButtonGroup } from "@material-ui/lab";
import { useCallback, useMemo, useState } from "react";
import useNotionalTransferred from "../../hooks/useNotionalTransferred";
import { COLORS } from "../../muiTheme";
import { CHAINS_BY_ID } from "../../utils/consts";
import { TIME_FRAMES } from "./Charts/TimeFrame";
import {
  createTransferChartData,
  createTransactionData,
  formatTransactionCount,
} from "./Charts/utils";
import VolumeAreaChart from "./Charts/VolumeAreaChart";
import VolumeStackedBarChart from "./Charts/VolumeStackedBarChart";
import VolumeLineChart from "./Charts/VolumeLineChart";
import TransactionsAreaChart from "./Charts/TransactionsAreaChart";
import TransactionsLineChart from "./Charts/TransactionsLineChart";
import useTransactionTotals from "../../hooks/useTransactionTotals";

const DISPLAY_BY_VALUES = ["Dollar", "Percent", "Transactions"];

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

const VolumeStats = () => {
  const classes = useStyles();

  const [displayBy, setDisplayBy] = useState(DISPLAY_BY_VALUES[0]);
  const [timeFrame, setTimeFrame] = useState("All time");

  const [selectedChains, setSelectedChains] = useState<ChainId[]>([]);

  const notionalTransferred = useNotionalTransferred();

  const [transferData, transferredAllTime] = useMemo(() => {
    const transferData = notionalTransferred.data
      ? createTransferChartData(
          notionalTransferred.data,
          TIME_FRAMES[timeFrame]
        )
      : [];
    const transferredAllTime = transferData.reduce((sum, value) => {
      return sum + value.totalTransferred;
    }, 0);
    const transferredAllTimeString = new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "USD",
      maximumFractionDigits: 0,
    }).format(transferredAllTime);
    return [transferData, transferredAllTimeString];
  }, [notionalTransferred, timeFrame]);

  const transactionTotals = useTransactionTotals();

  const [transactionData, transactionsAllTime] = useMemo(() => {
    const transactionData = transactionTotals.data
      ? createTransactionData(transactionTotals.data, TIME_FRAMES[timeFrame])
      : [];
    const transactionsAllTime = formatTransactionCount(
      transactionData.reduce((sum, value) => {
        return sum + value.totalTransactions;
      }, 0)
    );
    return [transactionData, transactionsAllTime];
  }, [transactionTotals, timeFrame]);

  const availableChains = useMemo(() => {
    const chainIds = notionalTransferred.data
      ? Object.keys(
          Object.values(notionalTransferred.data.Daily)[0] || {}
        ).reduce<ChainId[]>((chainIds, key) => {
          if (key !== "*") {
            const chainId = parseInt(key) as ChainId;
            if (CHAINS_BY_ID[chainId] !== undefined) {
              chainIds.push(chainId);
            }
          }
          return chainIds;
        }, [])
      : [];
    setSelectedChains(chainIds);
    return chainIds;
  }, [notionalTransferred]);

  const handleDisplayByChange = useCallback((event, nextValue) => {
    if (nextValue !== null) {
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

  const allChainsSelected = selectedChains.length === availableChains.length;

  return (
    <>
      <div className={classes.description}>
        <Typography variant="h3">
          {displayBy === "Transactions"
            ? "Transaction Count"
            : "Outbound Volume"}
          <StyledTooltip
            title={
              displayBy === "Transactions"
                ? "Total number of transactions the Token Bridge has processed"
                : "Amount of assets bridged through Portal in the outbound direction"
            }
            className={classes.tooltip}
          >
            <InfoOutlined />
          </StyledTooltip>
        </Typography>
        <Typography variant="h3">
          {displayBy === "Transactions"
            ? transactionsAllTime
            : transferredAllTime}
        </Typography>
      </div>
      <div className={classes.displayBy}>
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
      </div>
      <Paper className={classes.mainPaper}>
        {displayBy === "Dollar" ? (
          notionalTransferred.data ? (
            allChainsSelected ? (
              <VolumeAreaChart
                transferData={transferData}
                timeFrame={TIME_FRAMES[timeFrame]}
              />
            ) : (
              <VolumeLineChart
                transferData={transferData}
                timeFrame={TIME_FRAMES[timeFrame]}
                chains={selectedChains}
              />
            )
          ) : (
            <CircularProgress className={classes.alignCenter} />
          )
        ) : displayBy === "Percent" ? (
          <VolumeStackedBarChart
            transferData={transferData}
            timeFrame={TIME_FRAMES[timeFrame]}
            selectedChains={selectedChains}
          />
        ) : transactionTotals.data ? (
          allChainsSelected ? (
            <TransactionsAreaChart
              transactionData={transactionData}
              timeFrame={TIME_FRAMES[timeFrame]}
            />
          ) : (
            <TransactionsLineChart
              transactionData={transactionData}
              timeFrame={TIME_FRAMES[timeFrame]}
              chains={selectedChains}
            />
          )
        ) : (
          <CircularProgress className={classes.alignCenter} />
        )}
      </Paper>
    </>
  );
};

export default VolumeStats;
