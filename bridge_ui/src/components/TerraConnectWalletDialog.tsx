import {
  Dialog,
  DialogTitle,
  IconButton,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  makeStyles,
} from "@material-ui/core";
import CloseIcon from "@material-ui/icons/Close";
import { ConnectType, useWallet } from "@terra-money/wallet-provider";

const useStyles = makeStyles((theme) => ({
  flexTitle: {
    display: "flex",
    alignItems: "center",
    "& > div": {
      flexGrow: 1,
      marginRight: theme.spacing(4),
    },
    "& > button": {
      marginRight: theme.spacing(-1),
    },
  },
  icon: {
    height: 24,
    width: 24,
  },
}));

const TerraConnectWalletDialog = ({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) => {
  const { availableConnections, availableInstallations, connect } = useWallet();
  const classes = useStyles();

  return (
    <Dialog open={isOpen} onClose={onClose}>
      <DialogTitle>
        <div className={classes.flexTitle}>
          <div>Select your wallet</div>
          <IconButton onClick={onClose}>
            <CloseIcon />
          </IconButton>
        </div>
      </DialogTitle>
      <List>
        {availableConnections
          .filter(({ type }) => type !== ConnectType.READONLY)
          .map(({ type, name, icon, identifier = "" }) => (
            <ListItem
              button
              key={"connection-" + type + identifier}
              onClick={() => {
                connect(type, identifier);
                onClose();
              }}
            >
              <ListItemIcon>
                <img src={icon} alt={name} className={classes.icon} />
              </ListItemIcon>
              <ListItemText>{name}</ListItemText>
            </ListItem>
          ))}
        {availableInstallations
          .filter(({ type }) => type !== ConnectType.READONLY)
          .map(({ type, name, icon, url, identifier = "" }) => (
            <ListItem
              button
              component="a"
              key={"install-" + type + identifier}
              href={url}
              target="_blank"
              rel="noreferrer"
            >
              <ListItemIcon>
                <img src={icon} alt={name} className={classes.icon} />
              </ListItemIcon>
              <ListItemText>{"Install " + name}</ListItemText>
            </ListItem>
          ))}
      </List>
    </Dialog>
  );
};

export default TerraConnectWalletDialog;
