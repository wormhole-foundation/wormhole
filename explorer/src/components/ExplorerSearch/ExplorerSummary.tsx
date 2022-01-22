import React from "react";
import { Box, Button, Link, Typography } from "@mui/material";

import { BigTableMessage } from "./ExplorerQuery";
import { DecodePayload } from "../DecodePayload";
import ReactTimeAgo from "react-time-ago";
import { Link as RouterLink } from "gatsby";
import {
  contractNameFormatter,
  getNativeAddress,
  nativeExplorerContractUri,
  nativeExplorerTxUri,
  truncateAddress,
  usdFormatter,
} from "../../utils/explorer";
import { OutboundLink } from "gatsby-plugin-google-gtag";
import { ChainID, chainIDs } from "../../utils/consts";
import { hexToNativeString } from "@certusone/wormhole-sdk";
import { explorer } from "../../utils/urls";

interface SummaryProps {
  emitterChain?: number;
  emitterAddress?: string;
  sequence?: string;
  txId?: string;
  message: BigTableMessage;
  polling?: boolean;
  lastFetched?: number;
  refetch: () => void;
}
const textStyles = { fontSize: 16, margin: "6px 0" };

const ExplorerSummary = (props: SummaryProps) => {
  const { SignedVAA, ...message } = props.message;

  const {
    EmitterChain,
    EmitterAddress,
    InitiatingTxID,
    TokenTransferPayload,
    TransferDetails,
  } = message;
  // get chainId from chain name
  let chainId = chainIDs[EmitterChain];

  let transactionId: string | undefined;
  if (InitiatingTxID) {
    if (
      chainId === chainIDs["ethereum"] ||
      chainId === chainIDs["bsc"] ||
      chainId === chainIDs["polygon"]
    ) {
      transactionId = InitiatingTxID;
    } else {
      if (chainId === chainIDs["solana"]) {
        const txId = InitiatingTxID.slice(2); // remove the leading "0x"
        transactionId = hexToNativeString(txId, chainId);
      } else if (chainId === chainIDs["terra"]) {
        transactionId = InitiatingTxID.slice(2); // remove the leading "0x"
      }
    }
  }

  return (
    <>
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          gap: 8,
          alignItems: "baseline",
          marginTop: 40,
        }}
      >
        <Typography variant="h4">Message Summary</Typography>
        {props.polling ? (
          <>
            <div style={{ flexGrow: 1 }}></div>
            <Typography variant="caption">listening</Typography>
          </>
        ) : (
          <div>
            <Button onClick={props.refetch}>Refresh</Button>
            <Button component={RouterLink} to={explorer} sx={{ ml: 1 }}>
              Clear
            </Button>
          </div>
        )}
      </div>
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          margin: "20px 0 24px 20px",
        }}
      >
        <ul>
          {TokenTransferPayload &&
          TokenTransferPayload.TargetAddress &&
          TransferDetails &&
          nativeExplorerContractUri(
            Number(TokenTransferPayload.TargetChain),
            TokenTransferPayload.TargetAddress
          ) ? (
            <>
              <li>
                <span style={textStyles}>
                  This is a token transfer of{" "}
                  {Math.round(Number(TransferDetails.Amount) * 100) / 100}
                  {` `}
                  {!["UST", "LUNA"].includes(TransferDetails.OriginSymbol) ? (
                    <Link
                      component={OutboundLink}
                      href={nativeExplorerContractUri(
                        Number(TokenTransferPayload.OriginChain),
                        TokenTransferPayload.OriginAddress
                      )}
                      target="_blank"
                      rel="noopener noreferrer"
                      style={{ ...textStyles, whiteSpace: "nowrap" }}
                    >
                      {TransferDetails.OriginSymbol}
                    </Link>
                  ) : (
                    TransferDetails.OriginSymbol
                  )}
                  {` `}from {ChainID[chainId]}, to{" "}
                  {ChainID[Number(TokenTransferPayload.TargetChain)]}, destined
                  for address{" "}
                </span>
                <Link
                  component={OutboundLink}
                  href={nativeExplorerContractUri(
                    Number(TokenTransferPayload.TargetChain),
                    TokenTransferPayload.TargetAddress
                  )}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{ ...textStyles, whiteSpace: "nowrap" }}
                >
                  {truncateAddress(
                    getNativeAddress(
                      Number(TokenTransferPayload.TargetChain),
                      TokenTransferPayload.TargetAddress
                    )
                  )}
                </Link>
                <span style={textStyles}>.</span>
              </li>
              {TransferDetails.NotionalUSDStr && (
                <>
                  <li>
                    <span style={textStyles}>
                      When these tokens were sent to Wormhole, the{" "}
                      {Math.round(Number(TransferDetails.Amount) * 100) / 100}{" "}
                      {TransferDetails.OriginSymbol} was worth about{" "}
                      {usdFormatter.format(
                        Number(TransferDetails.NotionalUSDStr)
                      )}
                      .
                    </span>
                  </li>
                  <li>
                    <span style={textStyles}>
                      At the time of the transfer, 1{" "}
                      {TransferDetails.OriginName} was worth about{" "}
                      {usdFormatter.format(
                        Number(TransferDetails.TokenPriceUSDStr)
                      )}
                      .{" "}
                    </span>
                  </li>
                </>
              )}
            </>
          ) : null}
          {EmitterChain &&
          EmitterAddress &&
          nativeExplorerContractUri(chainId, EmitterAddress) ? (
            <li>
              <span style={textStyles}>
                This message was emitted by the {ChainID[chainId]}{" "}
              </span>
              <Link
                component={OutboundLink}
                href={nativeExplorerContractUri(chainId, EmitterAddress)}
                target="_blank"
                rel="noopener noreferrer"
                style={{ ...textStyles, whiteSpace: "nowrap" }}
              >
                {contractNameFormatter(EmitterAddress, chainId)}
              </Link>
              <span style={textStyles}> contract</span>
              {transactionId && (
                <>
                  <span style={textStyles}>
                    {" "}
                    after the Wormhole Guardians observed transaction{" "}
                  </span>
                  <Link
                    component={OutboundLink}
                    href={nativeExplorerTxUri(chainId, transactionId)}
                    target="_blank"
                    rel="noopener noreferrer"
                    style={{ ...textStyles, whiteSpace: "nowrap" }}
                  >
                    {truncateAddress(transactionId)}
                  </Link>
                </>
              )}{" "}
              <span style={textStyles}>.</span>
            </li>
          ) : null}
        </ul>
      </div>
      <Typography variant="h4">Raw message data:</Typography>
      <Box component="div" sx={{ overflow: "auto", mb: 2.5 }}>
        <pre style={{ fontSize: 14 }}>
          {JSON.stringify(message, undefined, 2)}
        </pre>
      </Box>
      <DecodePayload
        base64VAA={props.message.SignedVAABytes}
        emitterChainName={props.message.EmitterChain}
        emitterAddress={props.message.EmitterAddress}
        showPayload={true}
        transferDetails={props.message.TransferDetails}
      />
      <Box component="div" sx={{ overflow: "auto", mb: 2.5 }}>
        <Typography variant="h4">Signed VAA</Typography>
        <pre style={{ fontSize: 12 }}>
          {JSON.stringify(SignedVAA, undefined, 2)}
        </pre>
      </Box>
      <div style={{ display: "flex", justifyContent: "flex-end" }}>
        {props.lastFetched ? (
          <span>
            last updated:&nbsp;
            <ReactTimeAgo
              date={new Date(props.lastFetched)}
              timeStyle="round"
            />
          </span>
        ) : null}
      </div>
    </>
  );
};

export default ExplorerSummary;
