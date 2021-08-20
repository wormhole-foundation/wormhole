import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import { useHandleAttest } from "../../hooks/useHandleAttest";

const useStyles = makeStyles((theme) => ({
  transferButton: {
    marginTop: theme.spacing(2),
    textTransform: "none",
    width: "100%",
  },
}));

function Send() {
  const classes = useStyles();
  const { handleClick, disabled, showLoader } = useHandleAttest();
  return (
    <>
      <div style={{ position: "relative" }}>
        <Button
          color="primary"
          variant="contained"
          className={classes.transferButton}
          onClick={handleClick}
          disabled={disabled}
        >
          Attest
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
    </>
  );
}

export default Send;
