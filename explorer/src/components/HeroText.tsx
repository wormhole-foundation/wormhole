import { Box, Typography } from "@mui/material";
import React from "react";
import AvoidBreak from "./AvoidBreak";

const HeroText = ({
  heroSpans,
  subtitleText,
  maxWidth = 1155 + 16 * 2,
}: {
  heroSpans: string[];
  subtitleText: string | string[];
  maxWidth?: number;
}) => (
  <Box sx={{ m: "auto", maxWidth, textAlign: "center", px: 2 }}>
    <Typography variant="h1">
      <AvoidBreak spans={heroSpans} />
    </Typography>
    <Typography sx={{ marginTop: 2, fontWeight: 400 }}>
      {Array.isArray(subtitleText) ? (
        <AvoidBreak spans={subtitleText} />
      ) : (
        subtitleText
      )}
    </Typography>
  </Box>
);

export default HeroText;
