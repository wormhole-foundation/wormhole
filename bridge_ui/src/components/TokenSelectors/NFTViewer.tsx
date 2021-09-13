import {
  Card,
  CardContent,
  CardMedia,
  makeStyles,
  Typography,
} from "@material-ui/core";
import axios from "axios";
import { useEffect, useState } from "react";
import { NFTParsedTokenAccount } from "../../store/nftSlice";

const safeIPFS = (uri: string) =>
  uri.startsWith("ipfs://ipfs/")
    ? uri.replace("ipfs://", "https://ipfs.io/")
    : uri.startsWith("ipfs://")
    ? uri.replace("ipfs://", "https://ipfs.io/ipfs/")
    : uri.startsWith("https://cloudflare-ipfs.com/ipfs/") // no CORS support?
    ? uri.replace("https://cloudflare-ipfs.com/ipfs/", "https://ipfs.io/ipfs/")
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

export default function NFTViewer({ value }: { value: NFTParsedTokenAccount }) {
  const uri = safeIPFS(value.uri || "");
  const [metadata, setMetadata] = useState({
    image: value.image,
    animation_url: value.animation_url,
    name: value.name,
  });
  useEffect(() => {
    setMetadata({
      image: value.image,
      animation_url: value.animation_url,
      name: value.name,
    });
  }, [value]);
  useEffect(() => {
    if (uri) {
      let cancelled = false;
      (async () => {
        const result = await axios.get(uri);
        if (!cancelled && result && result.data) {
          setMetadata({
            image: result.data.image,
            animation_url: result.data.animation_url,
            name: result.data.name,
          });
        }
      })();
      return () => {
        cancelled = true;
      };
    }
  }, [uri, metadata.image]);
  const classes = useStyles();
  const animLower = metadata.animation_url?.toLowerCase();
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
      src={safeIPFS(metadata.image || "")}
      alt={metadata.name || ""}
      style={{ maxWidth: "100%" }}
    />
  );
  return (
    <Card className={classes.card} elevation={10}>
      <CardContent className={classes.textContent}>
        <Typography>
          {(value.symbol ? value.symbol + " " : "") + value.mintKey}
        </Typography>
        {metadata.name || value.tokenId ? (
          <Typography>
            {metadata.name}
            {value.tokenId ? ` (${value.tokenId})` : null}
          </Typography>
        ) : null}
      </CardContent>
      <CardMedia className={classes.mediaContent}>
        {hasVideo ? (
          <video controls style={{ maxWidth: "100%" }}>
            <source src={safeIPFS(metadata.animation_url || "")} />
            {image}
          </video>
        ) : (
          image
        )}
        {hasAudio ? (
          <audio controls src={safeIPFS(metadata.animation_url || "")} />
        ) : null}
      </CardMedia>
    </Card>
  );
}
