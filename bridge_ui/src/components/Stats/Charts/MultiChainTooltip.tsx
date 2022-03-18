import { ChainId } from "@certusone/wormhole-sdk";
import { makeStyles, Grid, Typography } from "@material-ui/core";
import {
  getChainShortName,
  CHAINS_BY_ID,
  COLOR_BY_CHAIN_ID,
} from "../../../utils/consts";
import { formatDate } from "./utils";

const useStyles = makeStyles(() => ({
  container: {
    padding: "16px",
    minWidth: "214px",
    background: "rgba(255, 255, 255, 0.95)",
    borderRadius: "4px",
  },
  titleText: {
    color: "#21227E",
    fontSize: "24px",
    fontWeight: 500,
  },
  row: {
    display: "flex",
    alignItems: "center",
    marginBottom: "8px",
  },
  ruler: {
    height: "3px",
    backgroundColor: "#374B92",
  },
  valueText: {
    color: "#404040",
    fontSize: "18px",
    fontWeight: 500,
  },
  icon: {
    width: "24px",
    height: "24px",
  },
}));

const MultiChainTooltip = ({ active, payload, title, valueFormatter }: any) => {
  const classes = useStyles();
  if (active && payload && payload.length) {
    if (payload.length === 1) {
      const chainId = +payload[0].dataKey.split(".")[1] as ChainId;
      const chainShortName = getChainShortName(chainId);
      const data = payload.find((data: any) => data.name === chainShortName);
      if (data) {
        return (
          <div className={classes.container}>
            <Grid container alignItems="center">
              <img
                className={classes.icon}
                src={CHAINS_BY_ID[chainId]?.logo}
                alt={chainShortName}
              />
              <Typography
                display="inline"
                className={classes.titleText}
                style={{ marginLeft: "8px" }}
              >
                {chainShortName}
              </Typography>
            </Grid>
            <hr
              className={classes.ruler}
              style={{ backgroundColor: COLOR_BY_CHAIN_ID[chainId] }}
            ></hr>
            <Typography className={classes.valueText}>
              {valueFormatter(data.value)}
            </Typography>
            <Typography className={classes.valueText}>
              {formatDate(data.payload.date)}
            </Typography>
          </div>
        );
      }
    } else {
      return (
        <div className={classes.container}>
          <Typography noWrap className={classes.titleText}>
            {title}
          </Typography>
          <Typography className={classes.valueText}>
            {formatDate(payload[0].payload.date)}
          </Typography>
          <hr className={classes.ruler}></hr>
          {payload.map((data: any) => {
            return (
              <div key={data.name} className={classes.row}>
                <div
                  style={{
                    width: "24px",
                    height: "24px",
                    backgroundColor: data.stroke,
                  }}
                />
                <Typography
                  display="inline"
                  className={classes.valueText}
                  style={{ marginLeft: "8px", marginRight: "8px" }}
                >
                  {data.name}
                </Typography>
                <Typography
                  display="inline"
                  className={classes.valueText}
                  style={{ marginLeft: "auto" }}
                >
                  {valueFormatter(data.value)}
                </Typography>
              </div>
            );
          })}
        </div>
      );
    }
  }
  return null;
};

export default MultiChainTooltip;
