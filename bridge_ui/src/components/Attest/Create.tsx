import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import { useHandleCreateWrapped } from "../../hooks/useHandleCreateWrapped";

const useStyles = makeStyles((theme) => ({
  transferButton: {
    marginTop: theme.spacing(2),
    textTransform: "none",
    width: "100%",
  },
}));

function Create() {
  const classes = useStyles();
  const { handleClick, disabled, showLoader } = useHandleCreateWrapped();
  return (
    <div style={{ position: "relative" }}>
      <Button
        color="primary"
        variant="contained"
        className={classes.transferButton}
        disabled={disabled}
        onClick={handleClick}
      >
        Create
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

export default Create;
