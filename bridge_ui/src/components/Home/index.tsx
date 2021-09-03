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
