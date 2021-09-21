import { useSnackbar } from "notistack";
import { useCallback } from "react";
import pushToClipboard from "../utils/pushToClipboard";

export default function useCopyToClipboard(content: string) {
  const { enqueueSnackbar } = useSnackbar();
  return useCallback(() => {
    pushToClipboard(content)?.then(() => {
      enqueueSnackbar("Copied", { variant: "success" });
    });
  }, [content, enqueueSnackbar]);
}
