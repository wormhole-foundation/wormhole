import { makeStyles, Typography } from "@material-ui/core";
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
  ruler: {
    height: "3px",
    backgroundImage: "linear-gradient(90deg, #F44B1B 0%, #EEB430 100%)",
  },
  valueText: {
    color: "#404040",
    fontSize: "18px",
    fontWeight: 500,
  },
}));

const CustomTooltip = ({ active, payload, title, valueFormatter }: any) => {
  const classes = useStyles();
  if (active && payload && payload.length) {
    return (
      <div className={classes.container}>
        <Typography className={classes.titleText}>{title}</Typography>
        <hr className={classes.ruler}></hr>
        <Typography className={classes.valueText}>
          {valueFormatter(payload[0].value)}
        </Typography>
        <Typography className={classes.valueText}>
          {formatDate(payload[0].payload.date)}
        </Typography>
      </div>
    );
  }
  return null;
};

export default CustomTooltip;
