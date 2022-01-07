import {
  Card,
  Chip,
  Container,
  Link,
  makeStyles,
  Typography,
} from "@material-ui/core";
import { Link as RouterLink } from "react-router-dom";
import { COLORS } from "../../muiTheme";
import { BETA_CHAINS, CHAINS, COMING_SOON_CHAINS } from "../../utils/consts";
import HeaderText from "../HeaderText";

const useStyles = makeStyles((theme) => ({
  header: {
    marginTop: theme.spacing(12),
    marginBottom: theme.spacing(8),
    [theme.breakpoints.down("sm")]: {
      marginBottom: theme.spacing(6),
    },
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
    padding: theme.spacing(8),
    backgroundColor: COLORS.nearBlackWithMinorTransparency,
  },
  spacer: {
    height: theme.spacing(5),
  },
  chainList: {
    display: "flex",
    flexWrap: "wrap",
    justifyContent: "center",
    margin: theme.spacing(-1, -1, 8),
    [theme.breakpoints.down("sm")]: {
      margin: theme.spacing(-1, -1, 6),
    },
  },
  chainCard: {
    backgroundColor: COLORS.nearBlackWithMinorTransparency,
    borderRadius: 8,
    display: "flex",
    flexDirection: "column",
    margin: theme.spacing(1),
    minHeight: "100%",
    padding: theme.spacing(2),
    width: 149, // makes it square
    maxWidth: 149,
    [theme.breakpoints.down("sm")]: {
      padding: theme.spacing(1.5),
      width: 141, // keeps it square
      maxWidth: 141,
    },
  },
  chainLogoWrapper: {
    position: "relative",
    textAlign: "center",
  },
  chainLogo: {
    height: 64,
    maxWidth: 64,
  },
  chainName: {
    marginTop: theme.spacing(1),
    flex: "1",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    textAlign: "center",
    minHeight: 40, // 2 lines
  },
  chip: {
    backgroundColor: COLORS.blueWithTransparency,
    position: "absolute",
    top: "50%",
    right: "50%",
    transform: "translate(50%, -50%)",
  },
}));

function Home() {
  const classes = useStyles();
  return (
    <div>
      <Container maxWidth="md">
        <div className={classes.header}>
          <HeaderText>The Portal is Open</HeaderText>
        </div>
      </Container>
      <Container maxWidth="md">
        <div className={classes.chainList}>
          {CHAINS.filter(({ id }) => !BETA_CHAINS.includes(id)).map((chain) => (
            <div key={chain.id} className={classes.chainCard}>
              <div className={classes.chainLogoWrapper}>
                <img
                  src={chain.logo}
                  alt={chain.name}
                  className={classes.chainLogo}
                />
              </div>
              <Typography
                variant="body2"
                component="div"
                className={classes.chainName}
              >
                <div>{chain.name}</div>
              </Typography>
            </div>
          ))}
          {COMING_SOON_CHAINS.map((item) => (
            <div className={classes.chainCard}>
              <div className={classes.chainLogoWrapper}>
                <img
                  src={item.logo}
                  alt={item.name}
                  className={classes.chainLogo}
                />
                <Chip
                  label="Coming soon"
                  size="small"
                  className={classes.chip}
                />
              </div>
              <Typography
                variant="body2"
                component="div"
                className={classes.chainName}
              >
                <div>{item.name}</div>
              </Typography>
            </div>
          ))}
        </div>
      </Container>
      <Container maxWidth="md">
        <Card className={classes.mainCard}>
          <Typography variant="h4" className={classes.description}>
            Wormhole v2 is here!
          </Typography>
          <Typography variant="h6" className={classes.description}>
            The Wormhole Token Bridge allows you to seamlessly transfer
            tokenized assets across Solana, Ethereum, BSC, Terra, Polygon,
            Avalanche, and Oasis.
          </Typography>
          <div className={classes.spacer} />
          <Typography variant="subtitle1" className={classes.description}>
            If you transferred assets using the previous version of Wormhole,
            most assets can be migrated to v2 on the{" "}
            <Link component={RouterLink} to="/transfer" noWrap>
              transfer page
            </Link>
            .
          </Typography>
          <Typography variant="subtitle1" className={classes.description}>
            For assets that don't support the migration, the v1 UI can be found
            at{" "}
            <Link href="https://v1.wormholebridge.com">
              v1.wormholebridge.com
            </Link>
          </Typography>
          <Typography variant="subtitle1" className={classes.description}>
            To learn more about the Wormhole Protocol that powers it, visit{" "}
            <Link href="https://wormholenetwork.com/en/">
              wormholenetwork.com
            </Link>
          </Typography>
        </Card>
      </Container>
    </div>
  );
}

export default Home;
