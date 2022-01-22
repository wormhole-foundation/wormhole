import React, { useEffect, useState } from "react";
import { PageProps, navigate } from "gatsby";
import { Button, TextField, Box } from "@mui/material";

import ExplorerQuery from "./ExplorerQuery";
import ExplorerFormSelect, { explorerFormType } from "./ExplorerFormSelect";

interface ExplorerTxSearchProps {
  location: PageProps["location"];
  toggleFormType: () => void;
  formName: explorerFormType;
}
const ExplorerTxSearchForm: React.FC<ExplorerTxSearchProps> = ({
  location,
  toggleFormType,
  formName,
}) => {
  const [txId, setTxId] = useState<string>();

  useEffect(() => {
    if (location.search) {
      // take searchparams from the URL and set the values in the form
      const searchParams = new URLSearchParams(location.search);
      const txQuery = searchParams.get("txId");

      // if the search params are different form values, update the form.
      if (txQuery) {
        setTxId(txQuery);
      }
    } else {
      // clear state
      setTxId(undefined);
    }
  }, [location.search]);

  const handleSubmit = (event: any) => {
    event.preventDefault();
    // pushing to the history stack will cause the component to get new props, and useEffect will run.
    if (txId) {
      navigate(`/explorer?txId=${txId}`);
    }
  };

  const onTxId = (event: React.ChangeEvent<HTMLInputElement>) => {
    const tx = event.target.value;
    setTxId(tx.replace(/\s/g, ""));
  };

  return (
    <>
      <Box
        component="form"
        noValidate
        autoComplete="off"
        onSubmit={handleSubmit}
      >
        <ExplorerFormSelect
          currentlyActive={formName}
          toggleFormType={toggleFormType}
        />

        <TextField
          value={txId || ""}
          onChange={onTxId}
          label="Transaction"
          fullWidth
          size="small"
          sx={{ my: 1 }}
        />

        <Button
          type="submit"
          variant="contained"
          sx={{
            display: "block",
            mt: 1,
            ml: "auto",
          }}
        >
          Search
        </Button>
      </Box>
      {txId ? <ExplorerQuery txId={txId} /> : null}
    </>
  );
};

export default ExplorerTxSearchForm;
