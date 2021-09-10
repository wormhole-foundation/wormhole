import {
  Card,
  CardContent,
  CardMedia,
  makeStyles,
  Typography,
} from "@material-ui/core";
import { NFTParsedTokenAccount } from "../../store/nftSlice";

const safeIPFS = (uri: string) =>
  uri.startsWith("ipfs://ipfs/")
    ? uri.replace("ipfs://", "https://cloudflare-ipfs.com/")
    : uri.startsWith("ipfs://")
    ? uri.replace("ipfs://", "https://cloudflare-ipfs.com/ipfs/")
    : uri;

const useStyles = makeStyles((theme) => ({
  card: {
    background: "transparent",
    border: `1px solid ${theme.palette.divider}`,
    maxWidth: 480,
    width: 480,
    margin: `${theme.spacing(1)}px auto`,
  },
  textContent: {
    background: theme.palette.background.paper,
  },
  mediaContent: {
    background: "transparent",
  },
}));

export default function NFTViewer({
  value,
  symbol,
}: {
  value: NFTParsedTokenAccount;
  symbol?: string;
}) {
  const classes = useStyles();
  const animLower = value.animation_url?.toLowerCase();
  // const has3DModel = animLower?.endsWith('gltf') || animLower?.endsWith('glb')
  const hasVideo =
    !animLower?.startsWith("ipfs://") && // cloudflare ipfs doesn't support streaming video
    (animLower?.endsWith("webm") ||
      animLower?.endsWith("mp4") ||
      animLower?.endsWith("mov") ||
      animLower?.endsWith("m4v") ||
      animLower?.endsWith("ogv") ||
      animLower?.endsWith("ogg"));
  const hasAudio =
    animLower?.endsWith("mp3") ||
    animLower?.endsWith("flac") ||
    animLower?.endsWith("wav") ||
    animLower?.endsWith("oga");
  const image = (
    <img
      src={safeIPFS(value.image || "")}
      alt={value.name || ""}
      style={{ maxWidth: "100%" }}
    />
  );
  return (
    <Card className={classes.card} elevation={10}>
      <CardContent className={classes.textContent}>
        <Typography>{(symbol ? symbol + " " : "") + value.mintKey}</Typography>
        <Typography>
          {value.name} ({value.tokenId})
        </Typography>
      </CardContent>
      <CardMedia className={classes.mediaContent}>
        {hasVideo ? (
          <video controls style={{ maxWidth: "100%" }}>
            <source src={safeIPFS(value.animation_url || "")} />
            {image}
          </video>
        ) : (
          image
        )}
        {hasAudio ? (
          <audio controls src={safeIPFS(value.animation_url || "")} />
        ) : null}
      </CardMedia>
    </Card>
  );
}
