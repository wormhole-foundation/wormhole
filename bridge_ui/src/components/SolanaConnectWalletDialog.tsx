import { useCallback, useMemo } from "react";
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
import { WalletName, WalletReadyState } from "@solana/wallet-adapter-base";
import { useWallet, Wallet } from "@solana/wallet-adapter-react";

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

const DetectedWalletListItem = ({
  wallet,
  select,
  onClose,
}: {
  wallet: Wallet;
  select: (walletName: WalletName) => void;
  onClose: () => void;
}) => {
  const handleWalletClick = useCallback(() => {
    select(wallet.adapter.name);
    onClose();
  }, [select, onClose, wallet]);

  return (
    <ListItem button onClick={handleWalletClick}>
      <WalletListItem wallet={wallet} text={wallet.adapter.name} />
    </ListItem>
  );
};

const WalletListItem = ({ wallet, text }: { wallet: Wallet; text: string }) => {
  const classes = useStyles();
  return (
    <>
      <ListItemIcon>
        <img
          src={wallet.adapter.icon}
          alt={wallet.adapter.name}
          className={classes.icon}
        />
      </ListItemIcon>
      <ListItemText>{text}</ListItemText>
    </>
  );
};

const SolanaConnectWalletDialog = ({
  isOpen,
  onClose,
}: {
  isOpen: boolean;
  onClose: () => void;
}) => {
  const classes = useStyles();
  const { wallets, select } = useWallet();

  const [detected, undetected] = useMemo(() => {
    const detected: Wallet[] = [];
    const undetected: Wallet[] = [];
    for (const wallet of wallets) {
      if (
        wallet.readyState === WalletReadyState.Installed ||
        wallet.readyState === WalletReadyState.Loadable
      ) {
        detected.push(wallet);
      } else if (wallet.readyState === WalletReadyState.NotDetected) {
        undetected.push(wallet);
      }
    }
    return [detected, undetected];
  }, [wallets]);

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
        {detected.map((wallet) => (
          <DetectedWalletListItem
            wallet={wallet}
            select={select}
            onClose={onClose}
            key={wallet.adapter.name}
          />
        ))}
        {undetected && <Divider variant="middle" />}
        {undetected.map((wallet) => (
          <ListItem
            button
            onClick={onClose}
            component="a"
            key={wallet.adapter.name}
            href={wallet.adapter.url}
            target="_blank"
            rel="noreferrer"
          >
            <WalletListItem
              wallet={wallet}
              text={"Install " + wallet.adapter.name}
            />
          </ListItem>
        ))}
      </List>
    </Dialog>
  );
};

export default SolanaConnectWalletDialog;
