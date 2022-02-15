import * as React from "react";
import { Box, MenuItem, TextField, Typography } from "@mui/material";
import { useNetworkContext } from "../contexts/NetworkContext";
import { networks } from "../utils/consts";

const NetworkSelect = () => {
  const { activeNetwork, setActiveNetwork } = useNetworkContext();
  const handleNetworkChange = React.useCallback((e) => {
    setActiveNetwork(e.target.value);
  }, []);
  return (
    <Box sx={{ display: "flex", alignItems: "center" }}>
      <Typography variant="body2" sx={{ pr: 2 }}>
        Network
      </Typography>
      <TextField
        select
        margin="none"
        value={activeNetwork.name}
        onChange={handleNetworkChange}
      >
        {networks.map((n) => (
          <MenuItem key={n} value={n}>{`${n[0].toUpperCase()}${n.substring(
            1
          )}`}</MenuItem>
        ))}
      </TextField>
    </Box>
  );
};
export default NetworkSelect;
