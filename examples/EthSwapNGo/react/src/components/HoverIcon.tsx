import { IconButton } from "@material-ui/core";
import KeyboardArrowDownIcon from "@material-ui/icons/KeyboardArrowDown";
import SwapVertIcon from "@material-ui/icons/SwapVert";
import { useCallback, useState } from "react";

export default function HoverIcon({ onClick }: { onClick: () => void }) {
  const [showSwap, setShowSwap] = useState(false);

  const hovered = useCallback(() => {
    setShowSwap(true);
  }, []);

  const unHovered = useCallback(() => {
    setShowSwap(false);
  }, []);
  return (
    <IconButton onClick={onClick} onMouseOver={hovered} onMouseOut={unHovered}>
      {showSwap ? (
        <SwapVertIcon fontSize={"large"} />
      ) : (
        <KeyboardArrowDownIcon fontSize={"large"} />
      )}
    </IconButton>
  );
}
