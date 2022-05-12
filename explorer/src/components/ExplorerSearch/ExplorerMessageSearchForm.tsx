import React, { useEffect, useState } from "react";
import { PageProps, navigate } from "gatsby";

import ExplorerQuery from "./ExplorerQuery";
import { chainEnums, ChainID, chainIDs } from "../../utils/consts";
import { useNetworkContext } from "../../contexts/NetworkContext";
import { truncateAddress } from "../../utils/explorer";

import {
  Autocomplete,
  Button,
  FormControl,
  TextField,
  Typography,
  MenuItem,
  Box,
} from "@mui/material";
import ExplorerFormSelect, { explorerFormType } from "./ExplorerFormSelect";

// form props
interface ExplorerMessageSearchValues {
  emitterChain: number;
  emitterAddress: string;
  sequence: string;
}

const emitterChains = [
  { label: ChainID[1], value: chainIDs["solana"] },
  { label: ChainID[2], value: chainIDs["ethereum"] },
  { label: ChainID[3], value: chainIDs["terra"] },
  { label: ChainID[4], value: chainIDs["bsc"] },
  { label: ChainID[5], value: chainIDs["polygon"] },
  { label: ChainID[6], value: chainIDs["avalanche"] },
  { label: ChainID[7], value: chainIDs["oasis"] },
  { label: ChainID[9], value: chainIDs["aurora"] },
  { label: ChainID[10], value: chainIDs["fantom"] },
];

interface ExplorerMessageSearchProps {
  location: PageProps["location"];
  toggleFormType: () => void;
  formName: explorerFormType;
}

const ExplorerMessageSearchForm: React.FC<ExplorerMessageSearchProps> = ({
  location,
  toggleFormType,
  formName,
}) => {
  const [emitterChain, setEmitterChain] =
    useState<ExplorerMessageSearchValues["emitterChain"]>();
  const [emitterAddress, setEmitterAddress] =
    useState<ExplorerMessageSearchValues["emitterAddress"]>();
  const [sequence, setSequence] =
    useState<ExplorerMessageSearchValues["sequence"]>();

  const { activeNetwork } = useNetworkContext();

  useEffect(() => {
    if (location.search) {
      // take searchparams from the URL and set the values in the form
      const searchParams = new URLSearchParams(location.search);

      const chain = searchParams.get("emitterChain");
      const address = searchParams.get("emitterAddress");
      const seqQuery = searchParams.get("sequence");

      setEmitterChain(Number(chain));
      setEmitterAddress(address || undefined);
      setSequence(seqQuery || undefined);
    } else {
      // clear state
      setEmitterChain(undefined);
      setEmitterAddress(undefined);
      setSequence(undefined);
    }
  }, [location.search]);

  const handleSubmit = (event: React.ChangeEvent<HTMLFormElement>) => {
    event.preventDefault();
    // pushing to the history stack will cause the component to get new props, and useEffect will run.
    if (emitterChain && emitterAddress && sequence) {
      navigate(
        `?emitterChain=${emitterChain}&emitterAddress=${emitterAddress}&sequence=${sequence}`
      );
    }
  };

  const onChain = (event: React.ChangeEvent<HTMLInputElement>) => {
    const { value } = event.target;
    setEmitterChain(Number(value));
  };

  const onAddress = (value: string) => {
    // trim whitespace
    setEmitterAddress(value.replace(/\s/g, ""));
  };

  const onSequence = (event: React.ChangeEvent<HTMLInputElement>) => {
    const { value } = event.target;
    // remove everything except numbers
    setSequence(value.replace(/\D/g, ""));
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
          select
          value={emitterChain || ""}
          onChange={onChain}
          placeholder="Chain"
          label="Chain"
          fullWidth
          size="small"
          sx={{ my: 1 }}
        >
          {emitterChains.map(({ label, value }) => (
            <MenuItem key={label} value={value}>
              {label}
            </MenuItem>
          ))}
        </TextField>

        <Autocomplete
          // TODO set value when loading the page with emitterAddress
          // value={emitterAddress || ""}
          freeSolo
          fullWidth
          size="small"
          sx={{ my: 1 }}
          onChange={(event, newVal: any) => onAddress(newVal.value)}
          placeholder="Contract"
          renderInput={(params: any) => (
            <TextField {...params} label="Emitter Contract" />
          )}
          getOptionLabel={(option) => option.label}
          // Get the chainID from the emitterChain form item, then use chainEnums to transform it to the
          // lowercase chain name, in order to use it to lookup the emitterAdresses of the active network.
          // Filter out keys that are not human readable names, by checking for a space in the key.
          options={Object.entries(
            activeNetwork.chains[
              chainEnums[emitterChain || 1]?.toLowerCase()
            ] || {}
          )
            .filter(([key]) => key.includes(" "))
            .map(([key, val]) => ({
              label: `${key} (${truncateAddress(val)})`,
              value: val,
            }))}
        />

        <TextField
          type="number"
          value={sequence ? Number(sequence) : ""}
          onChange={onSequence}
          label="Sequence"
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

      {emitterChain && emitterAddress && sequence ? (
        <ExplorerQuery
          emitterChain={emitterChain}
          emitterAddress={emitterAddress}
          sequence={sequence}
        />
      ) : null}
    </>
  );
};

export default ExplorerMessageSearchForm;
