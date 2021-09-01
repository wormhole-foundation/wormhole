import { Button, makeStyles } from "@material-ui/core";
import { ReactChild } from "react";

const useStyles = makeStyles((theme) => ({
  offsetButton: { display: "block", marginLeft: "auto", marginTop: 8 },
}));

export default function OffsetButton({
  onClick,
  disabled,
  children,
}: {
  onClick: () => void;
  disabled?: boolean;
  children: ReactChild;
}) {
  const classes = useStyles();
  return (
    <Button
      onClick={onClick}
      disabled={disabled}
      variant="outlined"
      className={classes.offsetButton}
    >
      {children}
    </Button>
  );
}
