import { Icon } from "@mui/material";
import {
  TableColumnBase,
  TableHeaderBase,
  TableHeaderContainer,
  TableHeaderContent,
  TextBase,
  TitleBase,
} from "./styled";
import CachedIcon from "@mui/icons-material/Cached";
import React from "react";
import { ITableHeaderProps } from "@/utils/types";
import BasicButton from "@/components/Common/BasicButton";
import RequestResponseTable from "../RequestResponseTable";
import MonitorTable from "../MonitorTable";
import Link from "next/link";
import RefreshIcon from "@/components/Common/refreshIcon";

const TableHeader = ({
  version,
  memoryUsage,
  fragmentationRatio,
  connectedClients,
  blockedClients,
  buttonText,
  onRefresh, // Add the onRefresh prop
}: ITableHeaderProps & { onRefresh: () => void }) => {
  const formatButtonText = (text: string) => {
    return text.toLowerCase();
  };

  return (
    <>
      <TableHeaderContainer>
        <TableHeaderBase>
          <Link href={`/bullmonitor/${formatButtonText(buttonText)}`}>
            <BasicButton
              text={buttonText}
              width="auto"
              justifyContent="center"
            />
          </Link>
          <RefreshIcon onRefresh={onRefresh} />
        </TableHeaderBase>
        <TableHeaderContent>
          <TableColumnBase>
            <TitleBase>Version</TitleBase>
            <TextBase>{version}</TextBase>
          </TableColumnBase>
          <TableColumnBase>
            <TitleBase>Memory Usage</TitleBase>
            <TextBase>{memoryUsage}</TextBase>
            {/* <div>(8.68MB of 17.2GB)</div> */}
          </TableColumnBase>
          <TableColumnBase>
            <TitleBase>Fragmentation Ratio</TitleBase>
            <TextBase>{fragmentationRatio}</TextBase>
          </TableColumnBase>
          <TableColumnBase>
            <TitleBase>Connected Clients</TitleBase>
            <TextBase>{connectedClients}</TextBase>
          </TableColumnBase>
          <TableColumnBase>
            <TitleBase>Blocked Clients</TitleBase>
            <TextBase>{blockedClients}</TextBase>
          </TableColumnBase>
        </TableHeaderContent>
      </TableHeaderContainer>
    </>
  );
};

export default TableHeader;
