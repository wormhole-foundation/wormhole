import {
  Button,
  CircularProgress,
  makeStyles,
  Typography,
} from "@material-ui/core";
import { ReactChild } from "react";

const useStyles = makeStyles((theme) => ({
  root: {
    position: "relative",
  },
  button: {
    marginTop: theme.spacing(2),
    width: "100%",
  },
  loader: {
    position: "absolute",
    bottom: 0,
    left: "50%",
    marginLeft: -12,
    marginBottom: 6,
  },
  error: {
    marginTop: theme.spacing(1),
    textAlign: "center",
  },
}));

export default function ButtonWithLoader({
  disabled,
  onClick,
  showLoader,
  error,
  children,
}: {
  disabled?: boolean;
  onClick: () => void;
  showLoader?: boolean;
  error?: string;
  children: ReactChild;
}) {
  const classes = useStyles();
  return (
    <>
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
      {error ? (
        <Typography variant="body2" color="error" className={classes.error}>
          {error}
        </Typography>
      ) : null}
    </>
  );
}
