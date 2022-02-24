import {
  Card,
  CardActionArea,
  Grid,
  GridSize,
  GridSpacing,
  Typography,
} from "@mui/material";
import { Box, ResponsiveStyleValue } from "@mui/system";
import { Link as RouterLink } from "gatsby";
import { OutboundLink } from "gatsby-plugin-google-gtag";
import React from "react";

interface CardData {
  key?: string;
  src: string;
  header: string;
  description: JSX.Element | string;
  href?: string;
  to?: string;
  imgStyle?: React.CSSProperties | undefined;
}

const GridWithCards = ({
  data,
  sm = 12,
  md = 4,
  spacing = 2,
  cardPaddingTop = 0,
  imgAlignMd = "right",
  imgOffsetRightMd = "-16px",
  imgOffsetTopXs = "-30px",
  imgOffsetTopMd = "-16px",
  imgOffsetTopMdHover,
  imgPaddingBottomXs = 0,
  imgPaddingBottomMd = 0,
  headerTextAlign = "left",
}: {
  data: CardData[];
  sm?: boolean | GridSize | undefined;
  md?: boolean | GridSize | undefined;
  spacing?: ResponsiveStyleValue<GridSpacing>;
  cardPaddingTop?: number;
  imgAlignMd?: string;
  imgOffsetRightMd?: string;
  imgOffsetTopXs?: string;
  imgOffsetTopMd?: string;
  imgOffsetTopMdHover?: string;
  imgPaddingBottomXs?: number;
  imgPaddingBottomMd?: number;
  headerTextAlign?: any;
}) => (
  <Grid
    container
    spacing={spacing}
    justifyContent="space-evenly"
    sx={{ "& > .MuiGrid-item": { pt: { xs: 8.25, md: 5.25 } } }}
  >
    {data.map(({ key, src, header, description, href, to, imgStyle }) => (
      <Grid key={key || header} item xs={12} sm={sm} md={md}>
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
            component={to ? RouterLink : href ? OutboundLink : undefined}
            to={to}
            href={href}
            target={href ? "_blank" : undefined}
            rel={href ? "noreferrer" : undefined}
            disabled={!(href || to)}
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
                textAlign: { xs: "center", md: imgAlignMd },
                position: "relative",
                right: { xs: null, md: imgOffsetRightMd },
                top: { xs: imgOffsetTopXs, md: imgOffsetTopMd },
                pb: { xs: imgPaddingBottomXs, md: imgPaddingBottomMd },
                zIndex: 1,
              }}
            >
              <img src={src} alt="" style={imgStyle} />
            </Box>
            <Typography variant="h4" textAlign={headerTextAlign}>
              {header}
            </Typography>
            <Typography component="div" sx={{ mt: 2, flexGrow: 1 }}>
              {description}
            </Typography>
          </CardActionArea>
        </Card>
      </Grid>
    ))}
  </Grid>
);

export default GridWithCards;
