import {
  Avatar,
  Card,
  CardContent,
  CardMedia,
  makeStyles,
  Tooltip,
  Typography,
} from "@material-ui/core";
import axios from "axios";
import { useCallback, useEffect, useLayoutEffect, useState } from "react";
import { NFTParsedTokenAccount } from "../../store/nftSlice";
import clsx from "clsx";
import {
  ChainId,
  CHAIN_ID_AVAX,
  CHAIN_ID_BSC,
  CHAIN_ID_ETH,
  CHAIN_ID_ETHEREUM_ROPSTEN,
  CHAIN_ID_POLYGON,
  CHAIN_ID_SOLANA,
  CHAIN_ID_OASIS,
  CHAIN_ID_FANTOM,
} from "@certusone/wormhole-sdk";
import SmartAddress from "../SmartAddress";
import avaxIcon from "../../icons/avax.svg";
import bscIcon from "../../icons/bsc.svg";
import ethIcon from "../../icons/eth.svg";
import fantomIcon from "../../icons/fantom.svg";
import solanaIcon from "../../icons/solana.svg";
import polygonIcon from "../../icons/polygon.svg";
import oasisIcon from "../../icons/oasis-network-rose-logo.svg";
import useCopyToClipboard from "../../hooks/useCopyToClipboard";
import { Skeleton } from "@material-ui/lab";
import Wormhole from "../../icons/wormhole-network.svg";

const safeIPFS = (uri: string) =>
  uri.startsWith("ipfs://ipfs/")
    ? uri.replace("ipfs://", "https://ipfs.io/")
    : uri.startsWith("ipfs://")
    ? uri.replace("ipfs://", "https://ipfs.io/ipfs/")
    : uri.startsWith("https://cloudflare-ipfs.com/ipfs/") // no CORS support?
    ? uri.replace("https://cloudflare-ipfs.com/ipfs/", "https://ipfs.io/ipfs/")
    : uri;

const LogoIcon = ({ chainId }: { chainId: ChainId }) =>
  chainId === CHAIN_ID_SOLANA ? (
    <Avatar
      style={{
        backgroundColor: "black",
        height: "1em",
        width: "1em",
        marginLeft: "4px",
        padding: "4px",
      }}
      src={solanaIcon}
      alt="Solana"
    />
  ) : chainId === CHAIN_ID_ETH || chainId === CHAIN_ID_ETHEREUM_ROPSTEN ? (
    <Avatar
      style={{
        backgroundColor: "white",
        height: "1em",
        width: "1em",
        marginLeft: "4px",
      }}
      src={ethIcon}
      alt="Ethereum"
    />
  ) : chainId === CHAIN_ID_BSC ? (
    <Avatar
      style={{
        backgroundColor: "rgb(20, 21, 26)",
        height: "1em",
        width: "1em",
        marginLeft: "4px",
        padding: "2px",
      }}
      src={bscIcon}
      alt="Binance Smart Chain"
    />
  ) : chainId === CHAIN_ID_POLYGON ? (
    <Avatar
      style={{
        backgroundColor: "black",
        height: "1em",
        width: "1em",
        marginLeft: "4px",
        padding: "3px",
      }}
      src={polygonIcon}
      alt="Polygon"
    />
  ) : chainId === CHAIN_ID_AVAX ? (
    <Avatar
      style={{
        backgroundColor: "black",
        height: "1em",
        width: "1em",
        marginLeft: "4px",
        padding: "3px",
      }}
      src={avaxIcon}
      alt="Avalanche"
    />
  ) : chainId === CHAIN_ID_OASIS ? (
    <Avatar
      style={{
        backgroundColor: "black",
        height: "1em",
        width: "1em",
        marginLeft: "4px",
        padding: "3px",
      }}
      src={oasisIcon}
      alt="Oasis"
    />
  ) : chainId === CHAIN_ID_FANTOM ? (
    <Avatar
      style={{
        backgroundColor: "black",
        height: "1em",
        width: "1em",
        marginLeft: "4px",
        padding: "3px",
      }}
      src={fantomIcon}
      alt="Fantom"
    />
  ) : null;

const useStyles = makeStyles((theme) => ({
  card: {
    borderRadius: 9,
    maxWidth: "100%",
    width: 400,
    margin: `${theme.spacing(1)}px auto`,
    padding: 8,
    position: "relative",
    zIndex: 1,
    transition: "background-position 1s, transform 0.25s",
    "&:hover": {
      backgroundPosition: "right center",
      transform: "scale(1.25)",
    },
    backgroundSize: "200% auto",
    backgroundColor: "#ffb347",
    background:
      "linear-gradient(to right, #ffb347 0%, #ffcc33  51%, #ffb347  100%)",
  },
  silverBorder: {
    backgroundColor: "#D9D8D6",
    backgroundSize: "200% auto",
    background:
      "linear-gradient(to bottom right, #757F9A 0%, #D7DDE8  51%, #757F9A  100%)",
    "&:hover": {
      backgroundPosition: "right center",
    },
  },
  cardInset: {},
  textContent: {
    background: "transparent",
    paddingTop: 4,
    paddingBottom: 2,
    display: "flex",
  },
  detailsContent: {
    background: "transparent",
    paddingTop: 4,
    paddingBottom: 2,
    "&:last-child": {
      //override rule
      paddingBottom: 2,
    },
  },
  title: {
    flex: 1,
  },
  description: {
    padding: theme.spacing(0.5, 0, 1),
  },
  tokenId: {
    fontSize: "8px",
  },
  mediaContent: {
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    justifyContent: "center",
    background: "transparent",
    margin: theme.spacing(0, 2),
    "& > img, & > video": {
      border: "1px solid #ffb347",
    },
  },
  silverMediaBorder: {
    "& > img, & > video": {
      borderColor: "#D7DDE8",
    },
  },
  // thanks https://cssgradient.io/ and https://htmlcolorcodes.com/color-picker/
  eth: {
    // colors from https://en.wikipedia.org/wiki/Ethereum#/media/File:Ethereum-icon-purple.svg
    backgroundColor: "rgb(69,74,117)",
    background:
      "linear-gradient(160deg, rgba(69,74,117,1) 0%, rgba(138,146,178,1) 33%, rgba(69,74,117,1) 66%, rgba(98,104,143,1) 100%)",
  },
  bsc: {
    // color from binance background rgb(20, 21, 26), 2 and 1 tint lighter
    backgroundColor: "#F0B90B",
    background:
      "linear-gradient(160deg, rgb(20, 21, 26) 0%, #4A4D57 33%, rgb(20, 21, 26) 66%, #2C2F3B 100%)",
  },
  polygon: {
    // color from polygon logo #8247E5 down to 30 lightness
    backgroundColor: "#0F0323",
    background:
      "linear-gradient(160deg, #0F0323 0%, #250957 33%, #0F0323 66%, #0F0323 100%)",
  },
  solana: {
    // colors from https://solana.com/branding/new/exchange/exchange-sq-black.svg
    backgroundColor: "rgb(153,69,255)",
    background:
      "linear-gradient(45deg, rgba(153,69,255,1) 0%, rgba(121,98,231,1) 20%, rgba(0,209,140,1) 100%)",
  },
  hidden: {
    display: "none",
  },
  skeleton: {
    height: "500px",
    width: "400px",
    maxWidth: "100%",
    borderRadius: 9,
    display: "grid",
    placeItems: "center",
    position: "absolute",
  },
  wormholeIcon: {
    height: 48,
    width: 48,
    filter: "contrast(0)",
    transition: "filter 0.5s",
    "&:hover": {
      filter: "contrast(1)",
    },
    verticalAlign: "middle",
    marginRight: theme.spacing(1),
    zIndex: 10,
  },
  wormholePositioner: {
    display: "grid",
    placeItems: "center",
    position: "relative",
    height: "500px",
    width: "400px",
    maxWidth: "100%",
    margin: `${theme.spacing(1)}px auto`,
  },
}));

const ViewerLoader = () => {
  const classes = useStyles();

  return (
    <div className={classes.wormholePositioner}>
      <Skeleton variant="rect" animation="wave" className={classes.skeleton} />
      <img src={Wormhole} alt="Wormhole" className={classes.wormholeIcon} />
    </div>
  );
};

export default function NFTViewer({
  value,
  chainId,
}: {
  value: NFTParsedTokenAccount;
  chainId: ChainId;
}) {
  const uri = safeIPFS(value.uri || "");
  const [metadata, setMetadata] = useState({
    uri,
    image: value.image,
    animation_url: value.animation_url,
    nftName: value.nftName,
    description: value.description,
    isLoading: !!uri,
  });
  const [isMediaLoading, setIsMediaLoading] = useState(false);
  const onLoad = useCallback(() => {
    setIsMediaLoading(false);
  }, []);
  const isLoading = isMediaLoading || metadata.isLoading;
  useEffect(() => {
    setMetadata((m) =>
      m.uri === uri
        ? m
        : {
            uri,
            image: value.image,
            animation_url: value.animation_url,
            nftName: value.nftName,
            description: value.description,
            isLoading: !!uri,
          }
    );
  }, [value, uri]);
  useEffect(() => {
    if (uri) {
      let cancelled = false;
      (async () => {
        try {
          const result = await axios.get(uri);
          if (!cancelled && result && result.data) {
            // support returns with nested data (e.g. {status: 10000, result: {data: {...}}})
            const data = result.data.result?.data || result.data;
            setMetadata({
              uri,
              image:
                data.image ||
                data.image_url ||
                data.big_image ||
                data.small_image,
              animation_url: data.animation_url,
              nftName: data.name,
              description: data.description,
              isLoading: false,
            });
          } else if (!cancelled) {
            setMetadata((m) => ({ ...m, isLoading: false }));
          }
        } catch (e) {
          if (!cancelled) {
            setMetadata((m) => ({ ...m, isLoading: false }));
          }
        }
      })();
      return () => {
        cancelled = true;
      };
    }
  }, [uri]);

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
  const hasImage = metadata.image;
  const copyTokenId = useCopyToClipboard(value.tokenId || "");
  const videoSrc = hasVideo && safeIPFS(metadata.animation_url || "");
  const imageSrc = hasImage && safeIPFS(metadata.image || "");
  const audioSrc = hasAudio && safeIPFS(metadata.animation_url || "");

  //set loading when the media src changes
  useLayoutEffect(() => {
    if (videoSrc || imageSrc || audioSrc) {
      setIsMediaLoading(true);
    } else {
      setIsMediaLoading(false);
    }
  }, [videoSrc, imageSrc, audioSrc]);

  const image = (
    <img
      src={imageSrc}
      alt={metadata.nftName || ""}
      style={{ maxWidth: "100%" }}
      onLoad={onLoad}
      onError={onLoad}
    />
  );
  const media = (
    <>
      {hasVideo ? (
        <video
          autoPlay
          controls
          loop
          style={{ maxWidth: "100%" }}
          onLoadedData={onLoad}
          onError={onLoad}
        >
          <source src={videoSrc || ""} />
          {image}
        </video>
      ) : hasImage ? (
        image
      ) : null}
      {hasAudio ? (
        <audio
          controls
          src={audioSrc || ""}
          onLoadedData={onLoad}
          onError={onLoad}
        />
      ) : null}
    </>
  );

  return (
    <>
      <div className={!isLoading ? classes.hidden : ""}>
        <ViewerLoader />
      </div>
      <Card
        className={clsx(classes.card, {
          [classes.silverBorder]:
            chainId === CHAIN_ID_SOLANA ||
            chainId === CHAIN_ID_POLYGON ||
            chainId === CHAIN_ID_AVAX,
          [classes.hidden]: isLoading,
        })}
        elevation={10}
      >
        <div
          className={clsx(classes.cardInset, {
            [classes.eth]:
              chainId === CHAIN_ID_ETH ||
              chainId === CHAIN_ID_ETHEREUM_ROPSTEN ||
              chainId === CHAIN_ID_AVAX || //TODO: give avax it's own bg
              chainId === CHAIN_ID_OASIS || //TODO: give oasis it's own bg
              chainId === CHAIN_ID_FANTOM, //TODO: give fantom it's own bg
            [classes.bsc]: chainId === CHAIN_ID_BSC,
            [classes.solana]: chainId === CHAIN_ID_SOLANA,
            [classes.polygon]: chainId === CHAIN_ID_POLYGON,
          })}
        >
          <CardContent className={classes.textContent}>
            {metadata.nftName ? (
              <Typography className={classes.title}>
                {metadata.nftName}
              </Typography>
            ) : (
              <div className={classes.title} />
            )}
            <SmartAddress
              chainId={chainId}
              parsedTokenAccount={value}
              noGutter
              noUnderline
            />
            <LogoIcon chainId={chainId} />
          </CardContent>
          <CardMedia
            className={clsx(classes.mediaContent, {
              [classes.silverMediaBorder]:
                chainId === CHAIN_ID_SOLANA ||
                chainId === CHAIN_ID_POLYGON ||
                chainId === CHAIN_ID_OASIS ||
                chainId === CHAIN_ID_AVAX,
            })}
          >
            {media}
          </CardMedia>
          <CardContent className={classes.detailsContent}>
            {metadata.description ? (
              <Typography variant="body2" className={classes.description}>
                {metadata.description}
              </Typography>
            ) : null}
            {value.tokenId ? (
              <Typography className={classes.tokenId} align="right">
                <Tooltip title="Copy" arrow>
                  <span onClick={copyTokenId}>
                    {value.tokenId.length > 18
                      ? `#${value.tokenId.substr(0, 16)}...`
                      : `#${value.tokenId}`}
                  </span>
                </Tooltip>
              </Typography>
            ) : null}
          </CardContent>
        </div>
      </Card>
    </>
  );
}
