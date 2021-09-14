import { makeStyles, Typography } from "@material-ui/core";
import { useSelector } from "react-redux";
import { selectAttestTargetChain } from "../../store/selectors";
import { CHAINS_BY_ID } from "../../utils/consts";

const useStyles = makeStyles((theme) => ({
  description: {
    textAlign: "center",
  },
}));

export default function TargetPreview() {
  const classes = useStyles();
  const targetChain = useSelector(selectAttestTargetChain);

  const explainerString = `to ${CHAINS_BY_ID[targetChain].name}`;

  return (
    <Typography
      component="div"
      variant="subtitle2"
      className={classes.description}
    >
      {explainerString}
    </Typography>
  );
}
