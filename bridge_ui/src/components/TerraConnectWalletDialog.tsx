import {
  Dialog,
  DialogTitle,
  Divider,
  IconButton,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  makeStyles,
} from "@material-ui/core";
import CloseIcon from "@material-ui/icons/Close";
import { ConnectType, useWallet } from "@terra-money/wallet-provider";
import { useCallback } from "react";

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

const WalletOptions = ({
  type,
  identifier,
  connect,
  onClose,
  icon,
  name,
}: {
  type: ConnectType;
  identifier: string;
  connect: (
    type: ConnectType | undefined,
    identifier: string | undefined
  ) => void;
  onClose: () => void;
  icon: string;
  name: string;
}) => {
  const classes = useStyles();

  const handleClick = useCallback(() => {
    connect(type, identifier);
    onClose();
  }, [connect, onClose, type, identifier]);
  return (
    <ListItem button onClick={handleClick}>
      <ListItemIcon>
        <img src={icon} alt={name} className={classes.icon} />
      </ListItemIcon>
      <ListItemText>{name}</ListItemText>
    </ListItem>
  );
};

const TerraConnectWalletDialog = ({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) => {
  const { availableConnections, availableInstallations, connect } = useWallet();
  const classes = useStyles();

  const filteredConnections = availableConnections
    .filter(({ type }) => type !== ConnectType.READONLY)
    .map(({ type, name, icon, identifier = "" }) => (
      <WalletOptions
        type={type}
        identifier={identifier}
        connect={connect}
        onClose={onClose}
        icon={icon}
        name={name}
        key={"connection-" + type + identifier}
      />
    ));

  const filteredInstallations = availableInstallations
    .filter(({ type }) => type !== ConnectType.READONLY)
    .map(({ type, name, icon, url, identifier = "" }) => (
      <ListItem
        button
        component="a"
        onClick={onClose}
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
    ));
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
        {filteredConnections}
        {filteredInstallations && <Divider variant="middle" />}
        {filteredInstallations}
      </List>
    </Dialog>
  );
};

export default TerraConnectWalletDialog;
