import { ChainId } from "@certusone/wormhole-sdk";
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
import { useCallback } from "react";
import {
  Connection,
  ConnectType,
  useEthereumProvider,
} from "../contexts/EthereumProviderContext";
import { getEvmChainId } from "../utils/consts";
import { EVM_RPC_MAP } from "../utils/metaMaskChainParameters";

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
  connection,
  connect,
  onClose,
}: {
  connection: Connection;
  connect: (connectType: ConnectType) => void;
  onClose: () => void;
}) => {
  const classes = useStyles();

  const handleClick = useCallback(() => {
    connect(connection.connectType);
    onClose();
  }, [connect, connection, onClose]);

  return (
    <ListItem button onClick={handleClick}>
      <ListItemIcon>
        <img
          src={connection.icon}
          alt={connection.name}
          className={classes.icon}
        />
      </ListItemIcon>
      <ListItemText>{connection.name}</ListItemText>
    </ListItem>
  );
};

const EvmConnectWalletDialog = ({
  isOpen,
  onClose,
  chainId,
}: {
  isOpen: boolean;
  onClose: () => void;
  chainId: ChainId;
}) => {
  const { availableConnections, connect } = useEthereumProvider();
  const classes = useStyles();

  const availableWallets = availableConnections
    .filter((connection) => {
      if (connection.connectType === ConnectType.METAMASK) {
        return true;
      } else if (connection.connectType === ConnectType.WALLETCONNECT) {
        const evmChainId = getEvmChainId(chainId);
        // WalletConnect requires a rpc provider
        return (
          evmChainId !== undefined && EVM_RPC_MAP[evmChainId] !== undefined
        );
      } else {
        return false;
      }
    })
    .map((connection) => (
      <WalletOptions
        connection={connection}
        connect={connect}
        onClose={onClose}
        key={connection.name}
      />
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
      <List>{availableWallets}</List>
    </Dialog>
  );
};

export default EvmConnectWalletDialog;
