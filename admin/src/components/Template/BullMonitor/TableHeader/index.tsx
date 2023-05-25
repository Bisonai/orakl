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
import { IQueueInfoData } from "@/utils/types";
import BasicButton from "@/components/Common/BasicButton";
import Link from "next/link";
import RefreshIcon from "@/components/Common/refreshIcon";

const TableHeader = ({
  serviceData,
  onRefresh,
}: {
  serviceData: IQueueInfoData;
  onRefresh: () => void;
}) => {
  return (
    <>
      <TableHeaderContainer>
        <TableHeaderBase>
          <Link href={`/bullmonitor/${serviceData?.serviceName}`}>
            <BasicButton
              text={serviceData?.serviceName}
              width="auto"
              style={{ background: "#00ADB5" }}
              justifyContent="center"
            />
          </Link>
          <RefreshIcon onRefresh={onRefresh} />
        </TableHeaderBase>
        <TableHeaderContent>
          <TableColumnBase>
            <TitleBase>Version</TitleBase>
            <TextBase>{serviceData?.redisVersion}</TextBase>
          </TableColumnBase>
          <TableColumnBase>
            <TitleBase>Memory Usage</TitleBase>
            <TextBase>{serviceData?.usedMemoryHuman}%</TextBase>
          </TableColumnBase>
          <TableColumnBase>
            <TitleBase>Fragmentation Ratio</TitleBase>
            <TextBase>{serviceData?.fragmentationRatio}</TextBase>
          </TableColumnBase>
          <TableColumnBase>
            <TitleBase>Connected Clients</TitleBase>
            <TextBase>{serviceData?.connectedClients}</TextBase>
          </TableColumnBase>
          <TableColumnBase>
            <TitleBase>Blocked Clients</TitleBase>
            <TextBase>{serviceData?.blockedClients}</TextBase>
          </TableColumnBase>
        </TableHeaderContent>
      </TableHeaderContainer>
    </>
  );
};

export default TableHeader;
