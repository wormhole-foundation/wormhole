import React from "react";
import { Link as RouterLink } from "gatsby";
import { Recent } from "./ExplorerStats";
import ReactTimeAgo from "react-time-ago";
import {
  Link,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from "@mui/material";
import { contractNameFormatter } from "../../utils/explorer";
import { chainIDs } from "../../utils/consts";
import { formatQuorumDate } from "../../utils/time";
import { explorer } from "../../utils/urls";

interface RecentMessagesProps {
  recent: Recent;
  lastFetched?: number;
  title: string;
  hideTableTitles?: boolean;
}

const RecentMessages = (props: RecentMessagesProps) => {
  //   const columns: ColumnsType<BigTableMessage> = [
  //     {
  //       title: "",
  //       key: "icon",
  //       render: (item: BigTableMessage) =>
  //         networkIcons[chainIDs[item.EmitterChain]],
  //       responsive: ["sm"],
  //     },
  //     {
  //       title: "contract",
  //       key: "contract",
  //       render: (item: BigTableMessage) => {
  //         const name = contractNameFormatter(
  //           item.EmitterAddress,
  //           chainIDs[item.EmitterChain]
  //         );
  //         return <div>{name}</div>;
  //       },
  //       responsive: ["sm"],
  //     },
  //     {
  //       title: "message",
  //       key: "payload",
  //       render: (item: BigTableMessage) =>
  //         item.SignedVAABytes ? (
  //           <DecodePayload
  //             base64VAA={item.SignedVAABytes}
  //             emitterChainName={item.EmitterChain}
  //             emitterAddress={item.EmitterAddress}
  //             showType={true}
  //             showSummary={true}
  //             transferDetails={item.TransferDetails}
  //           />
  //         ) : null,
  //     },
  //     {
  //       title: "sequence",
  //       key: "sequence",
  //       render: (item: BigTableMessage) => {
  //         let sequence = item.Sequence.replace(/^0+/, "");
  //         if (!sequence) sequence = "0";

  //         return sequence;
  //       },
  //       responsive: ["md"],
  //     },
  //     {
  //       title: "attested",
  //       dataIndex: "QuorumTime",
  //       key: "time",
  //       render: (QuorumTime) => (
  //         <ReactTimeAgo
  //           date={
  //             QuorumTime ? Date.parse(formatQuorumDate(QuorumTime)) : new Date()
  //           }
  //           locale={intl.locale}
  //           timeStyle={!screens.md ? "twitter" : "round"}
  //         />
  //       ),
  //     },
  //     {
  //       title: "",
  //       key: "view",
  //       render: (item: BigTableMessage) => (
  //         <Link
  //           to={`/${intl.locale}/explorer/?emitterChain=${
  //             chainIDs[item.EmitterChain]
  //           }&emitterAddress=${item.EmitterAddress}&sequence=${item.Sequence}`}
  //         >
  //           View
  //         </Link>
  //       ),
  //     },
  //   ];

  // const formatKey = (key: string) => {
  //     if (props.hideTableTitles) {
  //         return null
  //     }
  //     if (key.includes(":")) {
  //         const parts = key.split(":")
  //         const link = `/${intl.locale}/explorer/?emitterChain=${parts[0]}&emitterAddress=${parts[1]}`
  //         return <Title level={4} style={titleStyles}>From {ChainID[Number(parts[0])]} contract: <Link to={link}>{contractNameFormatter(parts[1], Number(parts[0]))}</Link></Title>
  //     } else if (key === "*") {
  //         return <Title level={4} style={titleStyles}>From all chains and addresses</Title>
  //     } else {
  //         return <Title level={4} style={titleStyles}>From {ChainID[Number(key)]}</Title>
  //     }
  // }

  return (
    <>
      <Typography variant="h4">{props.title}</Typography>
      {Object.keys(props.recent).map((key) => (
        // <Table<BigTableMessage>
        //     key={key}
        //     rowKey={(item) => item.EmitterAddress + item.Sequence}
        //     style={{ marginBottom: 40 }}
        //     size={screens.lg ? "large" : "small"}
        //     columns={columns}
        //     dataSource={props.recent[key]}
        //     title={() => formatKey(key)}
        //     pagination={false}
        //     rowClassName="highlight-new-row"
        //     footer={() => {
        //         return props.lastFetched ? (
        //             <span>
        //                 <FormattedMessage id="explorer.lastUpdated" />:&nbsp;
        //                 <ReactTimeAgo date={new Date(props.lastFetched)} locale={intl.locale} timeStyle="twitter" />
        //             </span>

        //         ) : null
        //     }}
        // />
        <TableContainer key={key}>
          <Table size="small">
            <TableBody>
              {props.recent[key].map((item) => (
                <TableRow key={item.EmitterAddress + item.Sequence}>
                  <TableCell>
                    {contractNameFormatter(
                      item.EmitterAddress,
                      chainIDs[item.EmitterChain]
                    )}
                  </TableCell>
                  <TableCell sx={{ whiteSpace: "nowrap" }}>
                    {item.SignedVAABytes
                      ? null
                      : //  <DecodePayload
                        //    base64VAA={item.SignedVAABytes}
                        //   emitterChainName={item.EmitterChain}
                        //    emitterAddress={item.EmitterAddress}
                        //    showType={true}
                        //    showSummary={true}
                        //    transferDetails={item.TransferDetails}
                        //  />
                        null}
                  </TableCell>
                  <TableCell sx={{ whiteSpace: "nowrap" }}>
                    {item.Sequence.replace(/^0+/, "") || "0"}
                  </TableCell>
                  <TableCell sx={{ "& > time": { whiteSpace: "nowrap" } }}>
                    {
                      <ReactTimeAgo
                        date={
                          item.QuorumTime
                            ? Date.parse(formatQuorumDate(item.QuorumTime))
                            : new Date()
                        }
                        timeStyle={"round"}
                      />
                    }
                  </TableCell>
                  <TableCell>
                    <Link
                      component={RouterLink}
                      to={`${explorer}?emitterChain=${
                        chainIDs[item.EmitterChain]
                      }&emitterAddress=${item.EmitterAddress}&sequence=${
                        item.Sequence
                      }`}
                      color="inherit"
                    >
                      View
                    </Link>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      ))}
    </>
  );
};

export default RecentMessages;
