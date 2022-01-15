import { Box } from "@mui/material";
import React from "react";

const AvoidBreak = ({ spans }: { spans: string[] }) => (
  <>
    {spans.map((span, idx) => (
      <React.Fragment key={`${idx}|${span}`}>
        <Box component="span" sx={{ display: "inline-block" }}>
          {span}
        </Box>{" "}
      </React.Fragment>
    ))}
  </>
);

export default AvoidBreak;
