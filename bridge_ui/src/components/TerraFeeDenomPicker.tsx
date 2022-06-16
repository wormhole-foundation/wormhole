import {
  MenuItem,
  makeStyles,
  TextField,
  Typography,
  ListItemIcon,
} from "@material-ui/core";
import { useConnectedWallet } from "@terra-money/wallet-provider";
import { useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { setTerraFeeDenom } from "../store/feeSlice";
import { selectTerraFeeDenom } from "../store/selectors";
import useTerraNativeBalances from "../hooks/useTerraNativeBalances";
import { formatNativeDenom, getNativeTerraIcon } from "../utils/terra";
import { TerraChainId } from "@certusone/wormhole-sdk";

const useStyles = makeStyles((theme) => ({
  feePickerContainer: {
    display: "flex",
    flexDirection: "column",
    margin: `${theme.spacing(1)}px auto`,
    maxWidth: 200,
    width: "100%",
  },
  select: {
    "& .MuiSelect-root": {
      display: "flex",
      alignItems: "center",
    },
  },
  listItemIcon: {
    minWidth: 40,
  },
  icon: {
    height: 24,
    maxWidth: 24,
  },
}));

type TerraFeeDenomPickerProps = {
  disabled: boolean;
  chainId: TerraChainId;
};

export default function TerraFeeDenomPicker(props: TerraFeeDenomPickerProps) {
  const terraFeeDenom = useSelector(selectTerraFeeDenom);
  const wallet = useConnectedWallet();
  const { balances } = useTerraNativeBalances(
    props.chainId,
    wallet?.walletAddress
  );
  const dispatch = useDispatch();
  const classes = useStyles();

  const feeDenomItems = useMemo(() => {
    const items = [];
    if (balances) {
      for (const [denom, amount] of Object.entries(balances)) {
        if (amount === "0") continue;
        const symbol = formatNativeDenom(denom, props.chainId);
        if (symbol) {
          items.push({
            denom,
            symbol,
            icon: getNativeTerraIcon(symbol),
          });
        }
      }
    }
    // prevent an out-of-range value from being selected
    if (!items.find((item) => item.denom === terraFeeDenom)) {
      const symbol = formatNativeDenom(terraFeeDenom, props.chainId);
      items.push({
        denom: terraFeeDenom,
        symbol,
        icon: getNativeTerraIcon(symbol),
      });
    }
    return items;
  }, [balances, terraFeeDenom, props.chainId]);

  return (
    <div className={classes.feePickerContainer}>
      <Typography variant="caption">Fee Denomination</Typography>
      <TextField
        variant="outlined"
        size="small"
        select
        fullWidth
        value={terraFeeDenom}
        onChange={(event) => dispatch(setTerraFeeDenom(event.target.value))}
        disabled={props.disabled}
        className={classes.select}
      >
        {feeDenomItems.map((item) => {
          return (
            <MenuItem key={item.denom} value={item.denom}>
              <ListItemIcon>
                <img
                  src={item.icon}
                  alt={item.symbol}
                  className={classes.icon}
                />
              </ListItemIcon>
              {item.symbol}
            </MenuItem>
          );
        })}
      </TextField>
    </div>
  );
}
