import { Grid, GridSpacing, Typography } from "@mui/material";
import { Box, ResponsiveStyleValue } from "@mui/system";
import React from "react";

interface CardData {
  key?: string;
  src: string;
  header: string;
  description: string;
}

const GridWithCards = ({
  data,
  spacing = 2,
  cardPaddingTop = 0,
  imgOffsetRightMd = "-16px",
  imgOffsetTopXs = "-30px",
  imgOffsetTopMd = "-16px",
  imgPaddingBottomXs = 0,
  imgPaddingBottomMd = 0,
}: {
  data: CardData[];
  spacing?: ResponsiveStyleValue<GridSpacing>;
  cardPaddingTop?: number;
  imgOffsetRightMd?: string;
  imgOffsetTopXs?: string;
  imgOffsetTopMd?: string;
  imgPaddingBottomXs?: number;
  imgPaddingBottomMd?: number;
}) => (
  <Grid
    container
    spacing={spacing}
    sx={{ "& > .MuiGrid-item": { pt: { xs: 8.25, md: 5.25 } } }}
  >
    {data.map(({ key, src, header, description }) => (
      <Grid key={key || header} item xs={12} md={4}>
        <Box
          sx={{
            backgroundColor: "rgba(255,255,255,.07)",
            px: 4.25,
            pb: 3,
            pt: cardPaddingTop,
            borderRadius: "28px",
            display: "flex",
            flexDirection: "column",
            height: "100%",
          }}
        >
          <Box
            sx={{
              textAlign: { xs: "center", md: "right" },
              position: "relative",
              right: { xs: null, md: imgOffsetRightMd },
              top: { xs: imgOffsetTopXs, md: imgOffsetTopMd },
              pb: { xs: imgPaddingBottomXs, md: imgPaddingBottomMd },
            }}
          >
            <img src={src} alt="" />
          </Box>
          <Typography variant="h4">{header}</Typography>
          <Typography sx={{ mt: 2, flexGrow: 1 }}>{description}</Typography>
        </Box>
      </Grid>
    ))}
  </Grid>
);

export default GridWithCards;
