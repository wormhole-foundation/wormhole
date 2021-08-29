import { Button, CircularProgress, makeStyles } from "@material-ui/core";
import { ReactChild } from "react";

const useStyles = makeStyles((theme) => ({
  root: {
    position: "relative",
  },
  button: {
    marginTop: theme.spacing(2),
    textTransform: "none",
    width: "100%",
  },
  loader: {
    position: "absolute",
    bottom: 0,
    left: "50%",
    marginLeft: -12,
    marginBottom: 6,
  },
}));

export default function ButtonWithLoader({
  disabled,
  onClick,
  showLoader,
  children,
}: {
  disabled: boolean;
  onClick: () => void;
  showLoader: boolean;
  children: ReactChild;
}) {
  const classes = useStyles();
  return (
    <div className={classes.root}>
      <Button
        color="primary"
        variant="contained"
        className={classes.button}
        disabled={disabled}
        onClick={onClick}
      >
        {children}
      </Button>
      {showLoader ? (
        <CircularProgress
          size={24}
          color="inherit"
          className={classes.loader}
        />
      ) : null}
    </div>
  );
}
