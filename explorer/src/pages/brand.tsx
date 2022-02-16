import * as React from "react";
import { Box, Button, Grid, Typography } from "@mui/material";
import ArrowCircleDownIcon from '@mui/icons-material/ArrowCircleDown';
import { PageProps } from 'gatsby'
import { OutboundLink } from "gatsby-plugin-google-gtag";
import HeroText from "../components/HeroText";
import Layout from "../components/Layout";
import logos from "../images/brand/logos.svg";
import world from "../images/brand/world.svg";
import shape1 from "../images/index/shape1.svg";
import { SEO } from "../components/SEO";
import shapes from "../images/shape.png";
import shapes2 from "../images/shape2.png";
import icons1 from "../images/brand/icons1.png";
import icons2 from "../images/brand/icons2.svg";
import logo from "../images/brand/logo-name.svg";
import logo2 from "../images/brand/logo.svg";
import worm1 from "../images/brand/worm1.png";
import worm2 from "../images/brand/worm2.png";
import gradient1 from "../images/brand/gradient1.svg";
import gradient2 from "../images/brand/gradient2.svg";
import {
  logopackage,
  colors,
  icons,
  assets,
  logonamesvg,
  logonamepng,
  logopng,
  logosvg,
  wormpng1,
  wormpng2,
  gradients,
  gradient2svg,
  contact
} from "../utils/urls";

const BrandPage = ({ location }: PageProps) => {
  return (
    <Layout>
      <SEO
        title="BRAND"
        description="Please follow these guidelines when you’re sharing Wormhole with the world."
        pathname={location.pathname}
      />
      <Box sx={{ position: "relative", marginTop: 17 }}>
        <Box
            sx={{
              position: "absolute",
              zIndex: -2,
              bottom: '-220px',
              left: '20%',
              background: 'radial-gradient(closest-side at 50% 50%, #5189C8 0%, #5189C800 100%) ',
              transform: 'matrix(-0.19, 0.98, -0.98, -0.19, 0, 0)',
              width: 1609,
              height: 1264,
              pointerEvents: 'none',
              opacity: 0.7,
            }}
          />   
        <Box
          sx={{
            position: "absolute",
            zIndex: -1,
            transform: "translate(0px, -25%) scaleX(-1)",
            background: `url(${shape1})`,
            backgroundRepeat: "no-repeat",
            backgroundPosition: "top -500px center",
            backgroundSize: "2070px 1155px",
            width: "100%",
            height: 1155,
          }}
        />
        <HeroText
          heroSpans={["Brand"]}
          subtitleText="Integrate proudly with everything you need to show off Wormhole."
        />
      </Box>
      <Box sx={{ textAlign: "center", mt: 40, px: 2 }}>
        <Typography variant="h3">
          <Box component="span" sx={{ color: "#FFCE00" }}>
          Brand {" "}
          </Box>
          <Box component="span"> assets</Box>
        </Typography>
        <Typography sx={{ mt: 2, maxWidth: 860, mx: "auto" }}>Everything you need to show off Wormhole to the world.</Typography>
      </Box>
      <Box sx={{position: 'relative'}}>
        <Box
            sx={{
              position: "absolute",
              zIndex: -2,
              top: '-60%',
              background: 'radial-gradient(closest-side at 50% 50%, #5189C8 0%, #5189C800 100%)',
              transform: 'transform: matrix(-0.67, 0.74, -0.74, -0.67, 0, 0)',
              right: '15%',
              width: 1879,
              height: 1832,
              pointerEvents: 'none',
              opacity: 0.64,
            }}
          />
          <Box
            sx={{
              position: "absolute",
              zIndex: -1,
              background: `url(${shapes2})`,
              backgroundSize: 'contain',
              transform: 'scaleX(-1)',
              top: 150,
              right: '80vw',
              width: 1318,
              height: 1076,
              pointerEvents: 'none',
              display:{xs: 'none', md: 'block'},
            }}
          />
        <Box sx={{ m: "auto", maxWidth: 1164, px: 3.75, mt: {xs: 10, md:15.5} }}>
          <Box
            sx={{
              display: "flex",
              flexWrap: "wrap",
              alignItems: "center",
              justifyContent: "center",
            }}
          >
            <Box sx={{ flexBasis: { xs: "100%", md: "50%" }, flexGrow: 1 }}>
              <Box sx={{ px: { xs: 0, md: 4 } }}>
                <Box sx={{ maxWidth: 460, mx: "auto" }}>
                  <Typography variant="h3">
                    <Box component="span" sx={{ color: "#FFCE00" }}>
                    World of {" "}
                    </Box>
                    <Box component="span" sx={{ display: "inline-block" }}>
                      Wormhole
                    </Box>
                  </Typography>
                  <Typography sx={{ mt: 2 }}>
                    Combine the logo with a range of provided background colors and gradients to create the right feel.
                  </Typography>
                  <Button
                    component={OutboundLink}
                    href={logopackage}
                    sx={{ mt: 3 }}
                    variant="outlined"
                    color="inherit"
                    target="_blank"
                    startIcon={<ArrowCircleDownIcon />}
                  >
                LOGO
              </Button>
                </Box>
              </Box>
            </Box>
            <Box
              sx={{
                mt: { xs: 8, md: 0 },
                flexBasis: { xs: "100%", md: "50%" },
                textAlign: "center",
                flexGrow: 1,
                backgroundColor: "rgba(255,255,255,.06)",
                backdropFilter: "blur(3px)",
                borderRadius: "37px",
                px: { xs: 3, md: 5 },
                py: { xs: 3, md: 8 },
              }}
            >
              <img src={world} alt="" style={{ maxWidth: "100%" }} />
            </Box>
          </Box>
          <Box
            sx={{
              display: "flex",
              flexWrap: "wrap-reverse",
              alignItems: "center",
              justifyContent: "center",
              mt: { xs: 8, md: 0 },
            }}
          >
            <Box
              sx={{
                mt: { xs: 8, md: null },
                flexBasis: { xs: "100%", md: "50%" },
                textAlign: "center",
                flexGrow: 1,
                backgroundColor: "rgba(255,255,255,.06)",
                backdropFilter: "blur(3px)",
                borderRadius: "37px",
                pt: { xs: 3, md: 9.75 },
                pb: { xs: 3, md: 9 },
                px: { xs: 3, md: 8 },
              }}
            >
              <img src={logos} alt="" style={{ maxWidth: "100%" }} />
            </Box>
            <Box sx={{ flexBasis: { xs: "100%", md: "50%" }, flexGrow: 1 }}>
              <Box sx={{ px: { xs: 0, md: 4 } }}>
                <Box sx={{ maxWidth: 460, mx: "auto" }}>
                  <Typography variant="h3">
                    <Box component="span" sx={{ color: "#FFCE00" }}>
                      Give it {" "}
                    </Box>
                    <Box component="span" sx={{ display: "inline-block" }}>
                      space
                    </Box>
                  </Typography>
                  <Typography sx={{ mt: 2 }}>
                    Please give Wormhole the space it needs by 1x, which represents the small circle in the Wormhole brand logo mark.
                  </Typography>
                </Box>
              </Box>
            </Box>
          </Box>
        </Box>
      </Box>
      <Box sx={{position: 'relative'}}>
          <Box
              sx={{
                position: "absolute",
                zIndex: -2,
                top: '0',
                background: 'radial-gradient(closest-side at 50% 50%, #E72850 0%, #E7285000 100%)',
                transform: 'matrix(0.77, 0.64, -0.64, 0.77, 0, 0)',
                right: '75%',
                width: 1699,
                height: 1621,
                pointerEvents: 'none',
                opacity: 0.7,
              }}
            />   
             <Box
                sx={{
                  position: "absolute",
                  zIndex: -1,
                  background: `url(${shapes})`,
                  backgroundSize: 'contain',
                  transform: 'scaleX(-1)',
                  top: 250,
                  left: '75vw',
                  width: 1594,
                  height: 1322,
                  pointerEvents: 'none',
                  display:{xs: 'none', md: 'block'},
                }}
              />
              <Box
                sx={{
                  position: "absolute",
                  zIndex: -2,
                  top: -400,
                  background: 'radial-gradient(closest-side at 50% 50%, #5189C8 0%, #5189C800 100%)',
                  transform: 'matrix(-0.67, 0.74, -0.74, -0.67, 0, 0)',
                  left: '60%',
                  width: 1879,
                  height: 1832,
                  pointerEvents: 'none',
                  opacity: 0.7,
              }}
            />   
          <Box sx={{ textAlign: "center", mt: 12, px: 2 }}>
            <Typography variant="h3">
              <Box component="span" sx={{ color: "#FFCE00" }}>
              Color {" "}
              </Box>
              <Box component="span">palette</Box>
            </Typography>
            <Typography sx={{ mt: 2, maxWidth: 860, mx: "auto" }}>Mix and match from the color palette to fits your need.</Typography>
          </Box>
          <Box sx={{ m: "auto", maxWidth: 1006, px: 3.75 }}>
          <Grid container spacing={2}>
            <Grid item xs={12} md={4}>
              <Box
                  sx={{
                    mt: { xs: 8, md: null },
                    backgroundColor: "rgba(255,255,255,.06)",
                    backdropFilter: "blur(3px)",
                    borderRadius: "37px",
                    p:{xs:3, md: 5}
                  }}
                >
                  <Typography variant="caption"
                        sx={{
                          borderBottom: '1px solid #585587',
                          pb: 2,
                          mb: 5,
                          mt: 0
                        }}>
                        PRIMARY
                  </Typography>
                  <Grid container spacing={2} sx={{ textAlign: 'center'}}>
                    <Grid item xs={6}>
                      <Box sx={{
                        width: '100%',
                        aspectRatio: '1/1',
                        backgroundColor: '#E72850'
                      }}></Box>
                      <Typography variant="caption" >#E72850</Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{
                          width: '100%',
                          aspectRatio: '1/1',
                          backgroundColor: '#26276F'
                        }}></Box>
                      <Typography variant="caption">#26276F</Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{
                          width: '100%',
                          aspectRatio: '1/1',
                          backgroundColor: '#5189C8'
                        }}></Box>
                        <Typography variant="caption">#5189C8</Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{
                          width: '100%',
                          aspectRatio: '1/1',
                          backgroundColor: '#374B92'
                        }}></Box>
                      <Typography variant="caption">#374B92</Typography>
                    </Grid>
                  </Grid>
                </Box>
            </Grid>
            <Grid item xs={12} md={4}>
              <Box
                  sx={{
                    mt: { xs: 8, md: null },
                    backgroundColor: "rgba(255,255,255,.06)",
                    backdropFilter: "blur(3px)",
                    borderRadius: "37px",
                    p:{xs:3, md: 5}
                  }}
                >
                  <Typography variant="caption"
                        sx={{
                          borderBottom: '1px solid #585587',
                          pb: 2,
                          mb: 5,
                          mt: 0
                        }}>
                        GRADIENTS
                  </Typography>
                  <Grid container spacing={2} sx={{ textAlign: 'center'}}>
                    <Grid item xs={6}>
                      <Box sx={{
                        width: '100%',
                        aspectRatio: '1/1',
                        background: 'linear-gradient(180deg, #374B92 0%, #E72850 100%)'
                      }}></Box>
                      <Typography variant="caption">GRADIENT 1</Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{
                           width: '100%',
                           aspectRatio: '1/1',
                          background: 'linear-gradient(180deg, #E72850 0%, #5189C8 100%)'
                        }}></Box>
                      <Typography variant="caption">GRADIENT 2</Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{
                          width: '100%',
                          aspectRatio: '1/1',
                          background: 'linear-gradient(180deg, #17153F 0%, #E72850 100%)'
                        }}></Box>
                        <Typography variant="caption">GRADIENT 3</Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{
                          width: '100%',
                          aspectRatio: '1/1',
                          background: 'linear-gradient(180deg, #26276F 0%, #5189C8 100%)'
                        }}></Box>
                      <Typography variant="caption">GRADIENT 4</Typography>
                    </Grid>
                  </Grid>
                </Box>
            </Grid>
            <Grid item xs={12} md={4}>
              <Box
                  sx={{
                    mt: { xs: 8, md: null },
                    backgroundColor: "rgba(255,255,255,.06)",
                    backdropFilter: "blur(3px)",
                    borderRadius: "37px",
                    p:{xs:3, md: 5}
                  }}
                >
                  <Typography variant="caption"
                        sx={{
                          borderBottom: '1px solid #585587',
                          pb: 2,
                          mb: 5,
                          mt: 0
                        }}>
                        ACCENTS
                  </Typography>
                  <Grid container spacing={2} sx={{ textAlign: 'center'}}>
                    <Grid item xs={6}>
                      <Box sx={{
                        width: '100%',
                        aspectRatio: '1/1',
                        backgroundColor: '#FFCE00'
                      }}></Box>
                      <Typography variant="caption">#FFCE00</Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{
                          width: '100%',
                          aspectRatio: '1/1',
                          background: 'linear-gradient(180deg, #F44B1B 0%, #EEB430 100%)'
                        }}></Box>
                      <Typography variant="caption">GRADIENT 5</Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{
                          width: '100%',
                          aspectRatio: '1/1',
                          background: 'linear-gradient(180deg, #F44B1B 0%, #3D2670 100%)'
                        }}></Box>
                        <Typography variant="caption">GRADIENT 6</Typography>
                    </Grid>
                    <Grid item xs={6}>
                      <Box sx={{
                          width: '100%',
                          aspectRatio: '1/1',
                          background: 'linear-gradient(180deg, #EEB430 0%, #5189C8 100%)'
                        }}></Box>
                      <Typography variant="caption">GRADIENT 7</Typography>
                    </Grid>
                  </Grid>
                </Box>
            </Grid>

              

                
          </Grid>  
          <Box sx={{textAlign: 'center', mt: 5}}>
            <Button
                  component={OutboundLink}
                  href={colors}
                  sx={{ mt: 3 }}
                  variant="outlined"
                  color="inherit"
                  target="_blank"
                  startIcon={<ArrowCircleDownIcon />}
                >
                SVG
              </Button>
          </Box>
        </Box>  

      </Box>
      <Box sx={{position: 'relative'}}>
          <Box sx={{ textAlign: "center", mt: 12, px: 2 }}>
            <Typography variant="h3">
              <Box component="span" sx={{ color: "#FFCE00" }}>
              Iconography
              </Box>
            </Typography>
          </Box>
        <Box sx={{ m: "auto", maxWidth: 950, px: 3.75 }}>
          <Grid container spacing={2}>
            <Grid item xs={12} md={6}>
              <Box
                  sx={{
                    mt: { xs: 8, md: null },
                    backgroundColor: "rgba(255,255,255,.06)",
                    backdropFilter: "blur(3px)",
                    borderRadius: "37px",
                    p:{xs:3, md: 5}
                  }}
                >
                  <Typography variant="caption"
                        sx={{
                          borderBottom: '1px solid #585587',
                          pb: 2,
                          mb: 5,
                          mt: 0,
                        }}>
                        PRIMARY
                  </Typography>
                  <img src={icons1} alt="" style={{ maxWidth: "100%", display: 'block', margin: 'auto' }} />
                </Box>
            </Grid>
            <Grid item xs={12} md={6}>
              <Box
                  sx={{
                    mt: { xs: 8, md: null },
                    backgroundColor: "rgba(255,255,255,.06)",
                    backdropFilter: "blur(3px)",
                    borderRadius: "37px",
                    p:{xs:3, md: 5}
                  }}
                >
                  <Typography variant="caption"
                        sx={{
                          borderBottom: '1px solid #585587',
                          pb: 2,
                          mb: 5,
                          mt: 0
                        }}>
                        ALTERNATE
                  </Typography>
                  <img src={icons2} alt="" style={{ maxWidth: "100%", display: 'block', margin: 'auto' }} />
                </Box>
            </Grid>
                
          </Grid>  
          <Box sx={{textAlign: 'center', mt: 5}}>
            <Button
                  component={OutboundLink}
                  href={icons}
                  sx={{ mt: 3 }}
                  variant="outlined"
                  color="inherit"
                  target="_blank"
                  startIcon={<ArrowCircleDownIcon />}
                >
                SVG
              </Button>
          </Box>
        </Box>  

      </Box>
      <Box sx={{position: 'relative'}}>
        <Box
              sx={{
                position: "absolute",
                zIndex: -2,
                top: '0',
                background: 'radial-gradient(closest-side at 50% 50%, #E72850 0%, #E7285000 100%)',
                transform: 'matrix(0.77, 0.64, -0.64, 0.77, 0, 0)',
                left: '60%',
                width: 1699,
                height: 1621,
                pointerEvents: 'none',
                opacity: 0.5,
              }}
            />   
          <Box
              sx={{
                position: "absolute",
                zIndex: -1,
                background: `url(${shapes})`,
                backgroundSize: 'contain',
                top: -200,
                right: '80vw',
                width: 1594,
                height: 1322,
                pointerEvents: 'none',
                display:{xs: 'none', md: 'block'},
              }}
            />
          <Box sx={{ textAlign: "center", mt: 12, px: 2 }}>
            <Typography variant="h3">
              <Box component="span" sx={{ color: "#FFCE00" }}>
              Get the {" "}
              </Box>
              <Box component="span">assets</Box>
            </Typography>
            <Typography sx={{ mt: 2, maxWidth: 860, mx: "auto" }}>Mix and match from the color palette to fits your need.</Typography>
            <Button
                  component={OutboundLink}
                  href={assets}
                  sx={{ mt: 4 }}
                  variant="outlined"
                  color="inherit"
                  target="_blank"
                  startIcon={<ArrowCircleDownIcon />}
                >
                DOWNLOAD PACKAGE
              </Button>
          </Box>
         
          <Box sx={{maxWidth: 800, m:'60px auto 0',  borderTop: '1px solid #585587'}}>
            <Box sx={{
                      px: 3, 
                      py:2 , 
                      borderBottom: '1px solid #585587',
                      display: "flex",
                      flexWrap: "wrap",
                      alignItems: "center",
                      flexDirection: {xs: 'column-reverse', md:'row'},
                      minHeight: 124,
                      justifyContent: {xs: 'center', md:"space-between"},
               }}>
                  <Box 
                    sx={{
                      "a": {
                        m: {xs: '30px 10px 0 ', md:"0 16px 0 0"},
                      },
                    }}
                  >
                    <Button
                        component={OutboundLink}
                        href={logonamepng}
                        variant="outlined"
                        color="inherit"
                        target="_blank"
                        startIcon={<ArrowCircleDownIcon />}
                      >
                      PNG
                    </Button>
                    <Button
                        href={logonamesvg}
                        variant="outlined"
                        color="inherit"
                        target="_blank"
                        startIcon={<ArrowCircleDownIcon />}
                      >
                      SVG
                    </Button>
                  </Box>
                  <Box sx={{flexBasis:{xs: 'auto', md:250}, textAlign: 'center'}}>
                    <img src={logo} alt="" />
                  </Box>
            </Box>
            <Box sx={{
                      px: 3, 
                      py:2 , 
                      borderBottom: '1px solid #585587',
                      minHeight: 124,
                      display: "flex",
                      flexWrap: "wrap",
                      alignItems: "center",
                      flexDirection: {xs: 'column-reverse', md:'row'},
                      justifyContent: {xs: 'center', md:"space-between"},
               }}>
                  <Box 
                    sx={{
                      "a": {
                        m: {xs: '30px 10px 0 ', md:"0 16px 0 0"},
                      },
                    }}
                  >
                    <Button
                        component={OutboundLink}
                        href={logopng}
                        variant="outlined"
                        color="inherit"
                        target="_blank"
                        startIcon={<ArrowCircleDownIcon />}
                      >
                      PNG
                    </Button>
                    <Button
                        component={OutboundLink}
                        href={logosvg}
                        variant="outlined"
                        color="inherit"
                        target="_blank"
                        startIcon={<ArrowCircleDownIcon />}
                      >
                      SVG
                    </Button>
                  </Box>
                  
                  <Box sx={{flexBasis:{xs: 'auto', md:250}, textAlign: 'center'}}>
                    <img src={logo2} alt="" />
                  </Box>

            </Box>
            <Box sx={{
                      px: 3, 
                      py:2 , 
                      borderBottom: '1px solid #585587',
                      minHeight: 124,
                      display: "flex",
                      flexWrap: "wrap",
                      alignItems: "center",
                      flexDirection: {xs: 'column-reverse', md:'row'},
                      justifyContent: {xs: 'center', md:"space-between"},
               }}>
                  <Box 
                    sx={{
                      "a": {
                        m: {xs: '30px 10px 0 ', md:"0 16px 0 0"},
                      },
                    }}
                  >
                    <Button
                        component={OutboundLink}
                        href={wormpng1}
                        variant="outlined"
                        color="inherit"
                        target="_blank"
                        startIcon={<ArrowCircleDownIcon />}
                      >
                      PNG
                    </Button>
                    
                  </Box>
                  
                  <Box sx={{flexBasis:{xs: 'auto', md:250}, textAlign: 'center'}}>
                    <img src={worm1} alt="" />
                  </Box>

            </Box>
            <Box sx={{
                      px: 3, 
                      py:2 , 
                      minHeight: 124,
                      borderBottom: '1px solid #585587',
                      display: "flex",
                      flexWrap: "wrap",
                      alignItems: "center",
                      flexDirection: {xs: 'column-reverse', md:'row'},
                      justifyContent: {xs: 'center', md:"space-between"},
               }}>
                  <Box 
                    sx={{
                      "a": {
                        m: {xs: '30px 10px 0 ', md:"0 16px 0 0"},
                      },
                    }}
                  >
                    <Button
                        component={OutboundLink}
                        href={wormpng2}
                        variant="outlined"
                        color="inherit"
                        target="_blank"
                        startIcon={<ArrowCircleDownIcon />}
                      >
                      PNG
                    </Button>
                    
                  </Box>
                  <Box sx={{flexBasis:{xs: 'auto', md:250}, textAlign: 'center'}}>
                    <img src={worm2} alt="" />
                  </Box>
            </Box>
            <Box sx={{
                      px: 3, 
                      py:2 , 
                      borderBottom: '1px solid #585587',
                      minHeight: 124,
                      display: "flex",
                      flexWrap: "wrap",
                      alignItems: "center",
                      flexDirection: {xs: 'column-reverse', md:'row'},
                      justifyContent: {xs: 'center', md:"space-between"},
               }}>
                  <Box 
                    sx={{
                      "a": {
                        m: {xs: '30px 10px 0 ', md:"0 16px 0 0"},
                      },
                    }}
                  >
                    <Button
                        component={OutboundLink}
                        href={gradients}
                        variant="outlined"
                        color="inherit"
                        target="_blank"
                        startIcon={<ArrowCircleDownIcon />}
                      >
                      PNG
                    </Button>
                   
                  </Box>
                  <Box sx={{flexBasis:{xs: 'auto', md:250}, textAlign: 'center'}}>
                    <img src={gradient1} alt="" />
                  </Box>
            </Box>
            <Box sx={{
                      px: 3, 
                      py:2 , 
                      borderBottom: '1px solid #585587',
                      minHeight: 124,
                      display: "flex",
                      flexWrap: "wrap",
                      flexDirection: {xs: 'column-reverse', md:'row'},
                      alignItems: "center",
                      justifyContent: {xs: 'center', md:"space-between"},
               }}>
                  <Box 
                    sx={{
                      "a": {
                        m: {xs: '30px 10px 0 ', md:"0 16px 0 0"},
                      },
                    }}
                  >
                    <Button
                        href={gradient2svg}
                        variant="outlined"
                        color="inherit"
                        target="_blank"
                        startIcon={<ArrowCircleDownIcon />}
                      >
                      PNG
                    </Button>
                  </Box>
                  <Box sx={{flexBasis:{xs: 'auto', md:250}, textAlign: 'center'}}>
                    <img src={gradient2} alt="" />
                  </Box>
            </Box>
          </Box>

      </Box>

          {/* <Box sx={{ textAlign: "center", mt: 12, px: 2 }}>
            <Typography variant="h3">
                <Box component="span" sx={{ color: "#FFCE00" }}>
                Press  {" "}
                </Box>
              <Box component="span">inquiries</Box>
            </Typography>
            <Typography sx={{ mt: 2, maxWidth: 860, mx: "auto" }}>Reach out to our team to get the information you need.</Typography>
            <Button
                  href={contact}
                  sx={{ mt: 4 }}
                  variant="outlined"
                  color="inherit"
                >
                CONTACT US
              </Button>
          </Box> */}
    </Layout>
  );
};

export default BrandPage;
