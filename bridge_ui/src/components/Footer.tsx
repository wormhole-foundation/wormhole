import { makeStyles, Typography } from "@material-ui/core";

const useStyles = makeStyles((theme) => ({
  footer: {
    position: "relative",
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
  spacer: {
    flexGrow: 1,
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
  wormholeIcon: {
    height: 68,
    marginTop: -24,
  },
}));

export default function Footer() {
  const classes = useStyles();
  return (
    <footer className={classes.footer}>
      <div className={classes.container}>
        <div className={classes.flex}>
          <div className={classes.spacer} />
          <Typography variant="body2">
            This Interface is an open source software portal to Wormhole, a
            cross chain messaging protocol. THIS INTERFACE AND THE WORMHOLE
            PROTOCOL ARE PROVIDED "AS IS", AT YOUR OWN RISK, AND WITHOUT
            WARRANTIES OF ANY KIND. By using or accessing this Interface or
            Wormhole, you agree that no developer or entity involved in
            creating, deploying, maintaining, operating this Interface or
            Wormhole, or causing or supporting any of the foregoing, will be
            liable in any manner for any claims or damages whatsoever associated
            with your use, inability to use, or your interaction with other
            users of, this Interface or Wormhole, or this Interface or Wormhole
            themselves, including any direct, indirect, incidental, special,
            exemplary, punitive or consequential damages, or loss of profits,
            cryptocurrencies, tokens, or anything else of value. By using or
            accessing this Interface, you represent that you are not subject to
            sanctions or otherwise designated on any list of prohibited or
            restricted parties or excluded or denied persons, including but not
            limited to the lists maintained by the United States' Department of
            Treasury's Office of Foreign Assets Control, the United Nations
            Security Council, the European Union or its Member States, or any
            other government authority.
          </Typography>
        </div>
      </div>
    </footer>
  );
}
