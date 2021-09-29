import { Button, Paper, Typography } from "@material-ui/core";
import { useLogger } from "../contexts/Logger";

function LogWatcher() {
  const { logs, clear } = useLogger();

  return (
    <Paper style={{ padding: "1rem", maxHeight: "600px", overflow: "auto" }}>
      <Typography variant="h5">Logs</Typography>
      {logs.map((x) => (
        <Typography>{x}</Typography>
      ))}
      <Button
        onClick={clear}
        variant="contained"
        color="primary"
        style={{ marginTop: "2rem" }}
      >
        Clear All Logs
      </Button>
    </Paper>
  );
}

export default LogWatcher;
