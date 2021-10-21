import {
  Button,
  InputAdornment,
  TextField,
  TextFieldProps,
} from "@material-ui/core";

export default function NumberTextField(
  props: TextFieldProps & { onMaxClick?: () => void }
) {
  return (
    <TextField
      type="number"
      {...props}
      InputProps={{
        endAdornment: props.onMaxClick ? (
          <InputAdornment position="end">
            <Button
              onClick={props.onMaxClick}
              disabled={props.disabled}
              variant="outlined"
            >
              Max
            </Button>
          </InputAdornment>
        ) : undefined,
        ...(props?.InputProps || {}),
      }}
    ></TextField>
  );
}
