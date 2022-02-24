import { IconButton, Link, makeStyles, Typography } from "@material-ui/core";
import { Link as RouterLink, NavLink } from "react-router-dom";
import Discord from "../icons/Discord.svg";
import Github from "../icons/Github.svg";
import Medium from "../icons/Medium.svg";
import Portal from "../icons/portal_logo_w.svg";
import Telegram from "../icons/Telegram.svg";
import Twitter from "../icons/Twitter.svg";
import footerImg from "../images/Footer.png";

const useStyles = makeStyles((theme) => ({
  footer: {
    position: "relative",
  },
  backdrop: {
    position: "absolute",
    zIndex: -1,
    background: `url(${footerImg})`,
    backgroundRepeat: "no-repeat",
    backgroundPosition: "center center",
    backgroundSize: "cover",
    width: "100%",
    height: "100%",
    opacity: 0.25,
    [theme.breakpoints.up("md")]: {
      backgroundSize: "auto",
      opacity: 1,
      height: 444,
      backgroundPosition: "right bottom -326px",
    },
  },
  container: {
    maxWidth: 1100,
    margin: "0px auto",
    paddingTop: theme.spacing(11),
    paddingBottom: theme.spacing(6.5),
    [theme.breakpoints.up("md")]: {
      paddingBottom: theme.spacing(12),
    },
  },
  flex: {
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    marginLeft: theme.spacing(3.5),
    marginRight: theme.spacing(3.5),
    borderTop: "1px solid #585587",
    paddingTop: theme.spacing(7),
    [theme.breakpoints.up("md")]: {
      flexWrap: "wrap",
      flexDirection: "row",
      alignItems: "unset",
    },
  },
  logoWrapper: {
    paddingLeft: theme.spacing(0),
    paddingBottom: theme.spacing(2),
    borderTop: "1px solid #585587",
    paddingTop: theme.spacing(7),
    width: "100%",
    textAlign: "center",
    [theme.breakpoints.up("md")]: {
      paddingLeft: theme.spacing(2),
      paddingBottom: theme.spacing(2),
      borderTop: "none",
      paddingTop: theme.spacing(0),
      width: "auto",
      textAlign: "left",
    },
  },
  spacer: {
    flexGrow: 1,
  },
  linksWrapper: {
    paddingLeft: theme.spacing(0),
    order: -2,
    textAlign: "center",
    marginBottom: theme.spacing(7),
    [theme.breakpoints.up("md")]: {
      paddingLeft: theme.spacing(2),
      order: 0,
      textAlign: "left",
      mb: theme.spacing(0),
    },
  },
  linkStyle: {
    color: "white",
    display: "block",
    marginRight: theme.spacing(0),
    marginBottom: theme.spacing(1.5),
    fontSize: 14,
    textUnderlineOffset: "6px",
    [theme.breakpoints.up("md")]: {
      marginRight: theme.spacing(7.5),
    },
  },
  linkActiveStyle: { textDecoration: "underline" },
  socialWrapper: {
    padding: theme.spacing(0, 2),
    order: -2,
    textAlign: "center",
    borderTop: "1px solid #585587",
    paddingTop: theme.spacing(7),
    width: "100%",
    marginBottom: theme.spacing(7),
    [theme.breakpoints.up("md")]: {
      order: 0,
      textAlign: "left",
      borderTop: "none",
      paddingTop: theme.spacing(0),
      width: "auto",
      marginBottom: theme.spacing(0),
    },
  },
  socialHeader: {
    marginBottom: theme.spacing(3),
  },
  socialIcon: {
    padding: theme.spacing(1),
    "& img": {
      height: 24,
      width: 24,
    },
  },
  builtWithContainer: {
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    opacity: 0.5,
    marginTop: theme.spacing(1),
  },
  wormholeIcon: {
    height: 68,
    marginTop: -24,
  },
  copyWrapper: {
    flexBasis: "100%",
    paddingTop: theme.spacing(0),
    textAlign: "center",
    [theme.breakpoints.up("md")]: {
      paddingTop: theme.spacing(0),
      textAlign: "left",
    },
  },
}));

export default function Footer() {
  const classes = useStyles();
  return (
    <footer className={classes.footer}>
      <div className={classes.backdrop} />
      <div className={classes.container}>
        <div className={classes.flex}>
          <div className={classes.logoWrapper}>
            <RouterLink to={"/transfer"}>
              <img src={Portal} alt="Portal" className={classes.wormholeIcon} />
            </RouterLink>
          </div>
          <div className={classes.spacer} />
          <div className={classes.linksWrapper}>
            <div>
              <Link
                component={NavLink}
                to={"/transfer"}
                color="inherit"
                underline="hover"
                className={classes.linkStyle}
                activeClassName={classes.linkActiveStyle}
              >
                Bridge
              </Link>
              <Link
                href="https://docs.wormholenetwork.com/wormhole/faqs"
                target="_blank"
                rel="noopener noreferrer"
                color="inherit"
                underline="hover"
                className={classes.linkStyle}
              >
                FAQ
              </Link>
              <Link
                component={NavLink}
                to={"/stats"}
                color="inherit"
                underline="hover"
                className={classes.linkStyle}
                activeClassName={classes.linkActiveStyle}
              >
                Stats
              </Link>
              <Link
                href="https://wormholenetwork.com/"
                target="_blank"
                rel="noopener noreferrer"
                color="inherit"
                underline="hover"
                className={classes.linkStyle}
              >
                Wormhole
              </Link>
            </div>
          </div>
          <div className={classes.spacer} />
          <div className={classes.socialWrapper}>
            <Typography className={classes.socialHeader}>
              Let's be friends
            </Typography>
            <IconButton
              href="https://discord.gg/wormholecrypto"
              target="_blank"
              rel="noopener noreferrer"
              className={classes.socialIcon}
            >
              <img src={Discord} alt="Discord" />
            </IconButton>
            <IconButton
              href="https://github.com/certusone/wormhole"
              target="_blank"
              rel="noopener noreferrer"
              className={classes.socialIcon}
            >
              <img src={Github} alt="Github" />
            </IconButton>
            <IconButton
              href="http://wormholecrypto.medium.com"
              target="_blank"
              rel="noopener noreferrer"
              className={classes.socialIcon}
            >
              <img src={Medium} alt="Medium" />
            </IconButton>
            <IconButton
              href="https://t.me/wormholecrypto"
              target="_blank"
              rel="noopener noreferrer"
              className={classes.socialIcon}
            >
              <img src={Telegram} alt="Telegram" />
            </IconButton>
            <IconButton
              href="https://twitter.com/wormholecrypto"
              target="_blank"
              rel="noopener noreferrer"
              className={classes.socialIcon}
            >
              <img src={Twitter} alt="Twitter" />
            </IconButton>
          </div>
          <div className={classes.copyWrapper}>
            <Typography variant="body2">
              2022 &copy; Wormhole. All Rights Reserved.
            </Typography>
          </div>
        </div>
      </div>
    </footer>
  );
}
