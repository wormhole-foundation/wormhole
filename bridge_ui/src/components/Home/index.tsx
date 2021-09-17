import {
  Button,
  Container,
  Link,
  makeStyles,
  Typography,
} from "@material-ui/core";
import { Link as RouterLink } from "react-router-dom";
import overview from "../../images/overview.svg";

const useStyles = makeStyles((theme) => ({
  rootContainer: {
    backgroundColor: "rgba(0,0,0,0.2)",
    margin: theme.spacing(4, 0),
    padding: theme.spacing(4, 4),
    textAlign: "center",
  },
  header: {
    marginBottom: theme.spacing(12),
    [theme.breakpoints.down("sm")]: {
      marginBottom: theme.spacing(6),
    },
  },
  description: {
    fontWeight: 400,
    marginBottom: theme.spacing(4),
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
}));

function Home() {
  const classes = useStyles();
  return (
    <Container maxWidth="lg">
      <div className={classes.rootContainer}>
        <Typography variant="h3" className={classes.header}>
          The portal is open.
        </Typography>
        <Typography variant="h5" gutterBottom>
          Wormhole v2 is here!
        </Typography>
        <Typography variant="subtitle1" gutterBottom>
          If you transferred assets using the previous version of Wormhole, most
          assets can be migrated to v2 on the{" "}
          <Link component={RouterLink} to="/transfer" color="secondary">
            transfer page
          </Link>
          .
        </Typography>
        <Typography variant="subtitle1" className={classes.header}>
          For assets that don't support the migration, the v1 UI can be found at{" "}
          <Link href="https://v1.wormholebridge.com" color="secondary">
            v1.wormholebridge.com
          </Link>
        </Typography>
        <Typography variant="h6" className={classes.description}>
          The Wormhole Token Bridge allows you to seamlessly transfer tokenized
          assets across Solana and Ethereum.
        </Typography>
        <Button
          component={RouterLink}
          to="/transfer"
          variant="contained"
          color="primary"
          size="large"
          className={classes.button}
        >
          Transfer Tokens
        </Button>
        <Typography variant="h6" className={classes.description}>
          To learn more about the Wormhole Protocol that powers it, visit{" "}
          <Link href="https://wormholenetwork.com/en/" color="secondary">
            wormholenetwork.com
          </Link>
        </Typography>
        <img src={overview} alt="overview" className={classes.overview} />
      </div>
    </Container>
  );
}

export default Home;
