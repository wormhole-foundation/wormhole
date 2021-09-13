import {
  createStyles,
  IconButton,
  makeStyles,
  Tooltip,
} from "@material-ui/core";
import RefreshIcon from "@material-ui/icons/Refresh";

const useStyles = makeStyles(() =>
  createStyles({
    inlineContentWrapper: {
      display: "inline-block",
      flexGrow: 1,
    },
    flexWrapper: {
      "& > *": {
        margin: ".5rem",
      },
      display: "flex",
      alignItems: "center",
    },
  })
);

export default function RefreshButtonWrapper({
  children,
  callback,
}: {
  children: JSX.Element;
  callback: () => any;
}) {
  const classes = useStyles();

  const refreshWrapper = (
    <div className={classes.flexWrapper}>
      <div className={classes.inlineContentWrapper}>{children}</div>
      <Tooltip title="Reload Tokens">
        <IconButton onClick={callback}>
          <RefreshIcon />
        </IconButton>
      </Tooltip>
    </div>
  );

  return refreshWrapper;
}
