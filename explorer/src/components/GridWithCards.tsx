import {
  Card,
  CardActionArea,
  Grid,
  GridSpacing,
  Typography,
} from "@mui/material";
import { Box, ResponsiveStyleValue } from "@mui/system";
import React from "react";

interface CardData {
  key?: string;
  src: string;
  header: string;
  description: string;
  href?: string;
}

const GridWithCards = ({
  data,
  spacing = 2,
  cardPaddingTop = 0,
  imgOffsetRightMd = "-16px",
  imgOffsetTopXs = "-30px",
  imgOffsetTopMd = "-16px",
  imgOffsetTopMdHover,
  imgPaddingBottomXs = 0,
  imgPaddingBottomMd = 0,
}: {
  data: CardData[];
  spacing?: ResponsiveStyleValue<GridSpacing>;
  cardPaddingTop?: number;
  imgOffsetRightMd?: string;
  imgOffsetTopXs?: string;
  imgOffsetTopMd?: string;
  imgOffsetTopMdHover?: string;
  imgPaddingBottomXs?: number;
  imgPaddingBottomMd?: number;
}) => (
  <Grid
    container
    spacing={spacing}
    sx={{ "& > .MuiGrid-item": { pt: { xs: 8.25, md: 5.25 } } }}
  >
    {data.map(({ key, src, header, description, href }) => (
      <Grid key={key || header} item xs={12} md={4}>
        <Card
          sx={{
            backgroundColor: "rgba(255,255,255,.07)",
            backgroundImage: "none",
            borderRadius: "28px",
            display: "flex",
            flexDirection: "column",
            height: "100%",
            overflow: "visible",
          }}
        >
          <CardActionArea
            href={href ? href : undefined}
            target="_blank"
            rel="noreferrer"
            disabled={!href}
            sx={{
              px: 4.25,
              pb: 3,
              pt: cardPaddingTop,
              borderRadius: "28px",
              height: "100%",
              "& > div": {
                transition: { md: "300ms top" },
              },
              "&:hover > div": {
                top: {
                  xs: imgOffsetTopXs,
                  md: imgOffsetTopMdHover || imgOffsetTopMd,
                },
              },
            }}
          >
            <Box
              sx={{
                textAlign: { xs: "center", md: "right" },
                position: "relative",
                right: { xs: null, md: imgOffsetRightMd },
                top: { xs: imgOffsetTopXs, md: imgOffsetTopMd },
                pb: { xs: imgPaddingBottomXs, md: imgPaddingBottomMd },
                zIndex: 1,
              }}
            >
              <img src={src} alt="" />
            </Box>
            <Typography variant="h4">{header}</Typography>
            <Typography sx={{ mt: 2, flexGrow: 1 }}>{description}</Typography>
          </CardActionArea>
        </Card>
      </Grid>
    ))}
  </Grid>
);

export default GridWithCards;
