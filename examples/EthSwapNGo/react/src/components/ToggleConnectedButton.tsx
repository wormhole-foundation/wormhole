import { Button, makeStyles, Tooltip } from "@material-ui/core";

const useStyles = makeStyles((theme) => ({
  button: {
    display: "block",
    margin: `${theme.spacing(1)}px auto`,
    width: "100%",
    maxWidth: 400,
  },
}));

const ToggleConnectedButton = ({
  connect,
  disconnect,
  connected,
  pk,
}: {
  connect(): any;
  disconnect(): any;
  connected: boolean;
  pk: string;
}) => {
  const classes = useStyles();
  const is0x = pk.startsWith("0x");
  return connected ? (
    <Tooltip title={pk}>
      <Button
        color="secondary"
        variant="contained"
        size="small"
        onClick={disconnect}
        className={classes.button}
      >
        Disconnect {pk.substring(0, is0x ? 6 : 3)}...
        {pk.substr(pk.length - (is0x ? 4 : 3))}
      </Button>
    </Tooltip>
  ) : (
    <Button
      color="primary"
      variant="contained"
      size="small"
      onClick={connect}
      className={classes.button}
    >
      Connect Wallet
    </Button>
  );
};

export default ToggleConnectedButton;
