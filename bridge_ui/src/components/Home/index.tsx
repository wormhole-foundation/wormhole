import {
  Button,
  Card,
  Container,
  Link,
  makeStyles,
  Typography,
} from "@material-ui/core";
import { Link as RouterLink } from "react-router-dom";
import overview from "../../images/overview2.svg";
import { COLORS } from "../../muiTheme";

const useStyles = makeStyles((theme) => ({
  centeredContainer: {
    textAlign: "center",
    width: "100%",
  },
  header: {
    marginTop: theme.spacing(12),
    marginBottom: theme.spacing(15),
    [theme.breakpoints.down("sm")]: {
      marginBottom: theme.spacing(6),
    },
  },
  linearGradient: {
    background: `linear-gradient(to left, ${COLORS.blue}, ${COLORS.green});`,
    WebkitBackgroundClip: "text",
    backgroundClip: "text",
    WebkitTextFillColor: "transparent",
    MozBackgroundClip: "text",
    MozTextFillColor: "transparent",
    filter: `drop-shadow( 0px 0px 8px ${COLORS.nearBlack}) drop-shadow( 0px 0px 14px ${COLORS.nearBlack}) drop-shadow( 0px 0px 24px ${COLORS.nearBlack})`,
  },
  description: {
    marginBottom: theme.spacing(2),
  },
  button: {
    marginBottom: theme.spacing(4),
  },
  overview: {
    marginTop: theme.spacing(6),
    [theme.breakpoints.down("sm")]: {
      marginTop: theme.spacing(2),
    },
    maxWidth: "100%",
  },
  mainCard: {
    padding: theme.spacing(1),
    borderRadius: "5px",
    backgroundColor: COLORS.nearBlackWithMinorTransparency,
  },
  spacer: {
    height: theme.spacing(5),
  },
}));

function Home() {
  const classes = useStyles();
  return (
    <div>
      <Container maxWidth="md">
        <div className={classes.centeredContainer}>
          <Typography variant="h2" component="h1" className={classes.header}>
            <span className={classes.linearGradient}>The Portal is Open</span>
          </Typography>
        </div>
      </Container>
      <Container maxWidth="sm">
        <Card className={classes.mainCard}>
          <Typography variant="h4" className={classes.description}>
            Wormhole v2 is here!
          </Typography>
          <Typography variant="h6" className={classes.description}>
            The Wormhole Token Bridge allows you to seamlessly transfer
            tokenized assets across Solana and Ethereum.
          </Typography>
          <Button
            component={RouterLink}
            to="/transfer"
            variant="contained"
            color="secondary"
            size="large"
            className={classes.button}
          >
            Transfer Tokens
          </Button>
          <div className={classes.spacer} />
          <Typography variant="subtitle1" className={classes.description}>
            If you transferred assets using the previous version of Wormhole,
            most assets can be migrated to v2 on the{" "}
            <Link
              component={RouterLink}
              to="/transfer"
              color="secondary"
              noWrap
            >
              transfer page
            </Link>
            .
          </Typography>
          <Typography variant="subtitle1" className={classes.description}>
            For assets that don't support the migration, the v1 UI can be found
            at{" "}
            <Link href="https://v1.wormholebridge.com" color="secondary">
              v1.wormholebridge.com
            </Link>
          </Typography>
          <Typography variant="subtitle1" className={classes.description}>
            To learn more about the Wormhole Protocol that powers it, visit{" "}
            <Link href="https://wormholenetwork.com/en/" color="secondary">
              wormholenetwork.com
            </Link>
          </Typography>
        </Card>
      </Container>
      <Container maxWidth="md">
        <div className={classes.centeredContainer}>
          <img src={overview} alt="overview" className={classes.overview} />
        </div>
      </Container>
    </div>
  );
}

export default Home;
