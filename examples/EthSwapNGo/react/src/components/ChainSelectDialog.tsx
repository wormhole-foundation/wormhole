import {
  Card,
  Dialog,
  DialogContent,
  ListItemIcon,
  ListItemText,
  makeStyles,
  MenuItem,
  Typography,
} from "@material-ui/core";
import { useCallback, useState } from "react";
import { ChainInfo, getDefaultNativeCurrencySymbol } from "../utils/consts";

const useStyles = makeStyles((theme) => ({
  selectedCard: {
    "&:hover": {
      cursor: "pointer",
      boxShadow: "inset 0 0 100px 100px rgba(255, 255, 255, 0.1)",
    },
    display: "flex",
    alignItems: "center",
    width: "max-content",
    padding: "1rem",
    background:
      "linear-gradient(90deg, rgba(69,74,117,.2) 0%, rgba(138,146,178,.2) 33%, rgba(69,74,117,.5) 66%, rgba(98,104,143,.5) 100%), linear-gradient(45deg, rgba(153,69,255,.1) 0%, rgba(121,98,231,.1) 20%, rgba(0,209,140,.1) 100%)",
  },
  style2: {
    background:
      "linear-gradient(270deg, rgba(69,74,117,.2) 0%, rgba(138,146,178,.2) 33%, rgba(69,74,117,.5) 66%, rgba(98,104,143,.5) 100%), linear-gradient(45deg, rgba(153,69,255,.1) 0%, rgba(121,98,231,.1) 20%, rgba(0,209,140,.1) 100%)",
  },
  selectedSymbol: {
    margin: "1rem",
  },
  listItemIcon: {
    minWidth: 40,
  },
  icon: {
    height: 50,
    maxWidth: 50,
  },
}));

const createChainMenuItem = (
  { id, name, logo }: ChainInfo,
  classes: any,
  handleSelect: any
) => {
  return (
    <MenuItem key={id} value={id} onClick={handleSelect}>
      <ListItemIcon className={classes.listItemIcon}>
        <img src={logo} alt={name} className={classes.icon} />
      </ListItemIcon>
      <ListItemText>{getDefaultNativeCurrencySymbol(id)}</ListItemText>
    </MenuItem>
  );
};

interface ChainSelectProps {
  chains: ChainInfo[];

  value: any;
  onChange: any;
  style2?: boolean;
}

export default function ChainSelect({
  chains,
  value,
  onChange,
  style2,
}: ChainSelectProps) {
  const classes = useStyles();
  const [open, setOpen] = useState(false);
  const info = chains.find((x) => x.id === value);

  // const handleClick = useCallback(() => {
  //   setOpen(true);
  // }, []);
  const handleClose = useCallback(() => {
    setOpen(false);
  }, []);
  const handleSelect = useCallback(
    (newValue) => {
      onChange(newValue);
      handleClose();
    },
    [handleClose, onChange]
  );
  return (
    <>
      <Card
        //TODO re-enable
        // onClick={handleClick}
        raised
        className={classes.selectedCard + (style2 ? " " + classes.style2 : "")}
      >
        <img
          src={info && info.logo}
          className={classes.icon}
          alt={"coin logo"}
        />
        <Typography variant="h6" className={classes.selectedSymbol}>
          {info && getDefaultNativeCurrencySymbol(info.id)}
        </Typography>
      </Card>
      <Dialog open={open} onClose={handleClose}>
        <DialogContent>
          {chains.map((chain) =>
            createChainMenuItem(chain, classes, () => handleSelect(chain))
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
