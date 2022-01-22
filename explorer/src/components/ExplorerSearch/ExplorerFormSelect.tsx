import React from "react";
import { TextField, Typography, MenuItem } from "@mui/material";

export type explorerFormType = "txID" | "messageID";

interface ExplorerFormSelect {
  currentlyActive: explorerFormType;
  toggleFormType: () => void;
}

const ExplorerFormSelect: React.FC<ExplorerFormSelect> = ({
  currentlyActive,
  toggleFormType,
}) => {
  const onQueryType = (event: React.ChangeEvent<HTMLInputElement>) => {
    const { value } = event.target;
    if (value !== currentlyActive) {
      // toggle the other form type
      toggleFormType();
    }
  };
  const formatOption = (message: string) => (
    <Typography variant="body2">{message}</Typography>
  );
  return (
    <TextField
      select
      value={currentlyActive}
      onChange={onQueryType}
      sx={{
        mb: 2,
      }}
    >
      <MenuItem value="txID">{formatOption("Search Transaction")}</MenuItem>
      <MenuItem value="messageID">{formatOption("Search Message ID")}</MenuItem>
    </TextField>
  );
};

export default ExplorerFormSelect;
