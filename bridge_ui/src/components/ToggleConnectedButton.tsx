import { Button, makeStyles, Tooltip } from "@material-ui/core";
import { LinkOff } from "@material-ui/icons";

const useStyles = makeStyles((theme) => ({
  button: {
    display: "flex",
    margin: `${theme.spacing(1)}px auto`,
    width: "100%",
    maxWidth: 400,
  },
  icon: {
    height: 24,
    width: 24,
  },
}));

const ToggleConnectedButton = ({
  connect,
  disconnect,
  connected,
  pk,
  walletIcon,
}: {
  connect(): any;
  disconnect(): any;
  connected: boolean;
  pk: string;
  walletIcon?: string;
}) => {
  const classes = useStyles();
  const is0x = pk.startsWith("0x");
  return connected ? (
    <Tooltip title={pk}>
      <Button
        color="primary"
        variant="contained"
        size="small"
        onClick={disconnect}
        className={classes.button}
        startIcon={
          walletIcon ? (
            <img className={classes.icon} src={walletIcon} alt="Wallet" />
          ) : (
            <LinkOff />
          )
        }
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
      Connect
    </Button>
  );
};

export default ToggleConnectedButton;
