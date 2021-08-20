import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import { useHandleRedeem } from "../../hooks/useHandleRedeem";

const useStyles = makeStyles((theme) => ({
  transferButton: {
    marginTop: theme.spacing(2),
    textTransform: "none",
    width: "100%",
  },
}));

function Redeem() {
  const classes = useStyles();
  const { handleClick, disabled, showLoader } = useHandleRedeem();
  return (
    <div style={{ position: "relative" }}>
      <Button
        color="primary"
        variant="contained"
        className={classes.transferButton}
        disabled={disabled}
        onClick={handleClick}
      >
        Redeem
      </Button>
      {showLoader ? (
        <CircularProgress
          size={24}
          color="inherit"
          style={{
            position: "absolute",
            bottom: 0,
            left: "50%",
            marginLeft: -12,
            marginBottom: 6,
          }}
        />
      ) : null}
    </div>
  );
}

export default Redeem;
