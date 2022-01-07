import { IconButton } from "@material-ui/core";
import { ArrowForward, SwapHoriz } from "@material-ui/icons";
import { useState } from "react";

export default function ChainSelectArrow({
  onClick,
  disabled,
}: {
  onClick: () => void;
  disabled: boolean;
}) {
  const [showSwap, setShowSwap] = useState(false);

  return (
    <IconButton
      onClick={onClick}
      onMouseEnter={() => {
        setShowSwap(true);
      }}
      onMouseLeave={() => {
        setShowSwap(false);
      }}
      disabled={disabled}
    >
      {showSwap ? <SwapHoriz /> : <ArrowForward />}
    </IconButton>
  );
}
