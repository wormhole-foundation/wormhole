import {
  CircularProgress,
  makeStyles,
  MenuItem,
  Select,
  Typography,
} from "@material-ui/core";
import { useCallback } from "react";
import useRelayersAvailable, { Relayer } from "../hooks/useRelayersAvailable";

const useStyles = makeStyles((theme) => ({
  mainContainer: {
    textAlign: "center",
  },
}));

export default function RelaySelector({
  selectedValue,
  onChange,
}: {
  selectedValue: Relayer | null;
  onChange: (newValue: Relayer | null) => void;
}) {
  const classes = useStyles();
  const availableRelayers = useRelayersAvailable(true);

  const loader = (
    <div>
      <CircularProgress></CircularProgress>
      <Typography>Loading available relayers</Typography>
    </div>
  );

  const onChangeWrapper = useCallback(
    (event) => {
      console.log(event, "event in selector");
      event.target.value
        ? onChange(
            availableRelayers?.data?.relayers?.find(
              (x) => x.url === event.target.value
            ) || null
          )
        : onChange(null);
    },
    [onChange]
  );

  console.log("selectedValue in relay selector", selectedValue);

  const selector = (
    <Select
      onChange={onChangeWrapper}
      value={selectedValue ? selectedValue.url : null}
    >
      {availableRelayers.data?.relayers?.map((item) => (
        <MenuItem key={item.url} value={item.url}>
          {item.name}
        </MenuItem>
      ))}
    </Select>
  );

  const error = (
    <Typography variant="body2" color="textSecondary">
      No relayers are available at this time.
    </Typography>
  );

  return (
    <div className={classes.mainContainer}>
      {availableRelayers.data?.relayers?.length
        ? selector
        : availableRelayers.isFetching
        ? loader
        : error}
    </div>
  );
}
