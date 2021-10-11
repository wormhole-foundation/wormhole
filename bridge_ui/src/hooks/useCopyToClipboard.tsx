import { Alert } from "@material-ui/lab";
import { useSnackbar } from "notistack";
import { useCallback } from "react";
import pushToClipboard from "../utils/pushToClipboard";

export default function useCopyToClipboard(content: string) {
  const { enqueueSnackbar } = useSnackbar();
  return useCallback(() => {
    pushToClipboard(content)?.then(() => {
      enqueueSnackbar(null, {
        content: <Alert severity="success">Copied.</Alert>,
      });
    });
  }, [content, enqueueSnackbar]);
}
