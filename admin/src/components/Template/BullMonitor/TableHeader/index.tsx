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

const TableHeader = ({
  version,
  memoryUsage,
  fragmentationRatio,
  connectedClients,
  blockedClients,
  buttonText,
  onRefresh,
}: ITableHeaderProps & { onRefresh: () => void }) => {
  const handleRefresh = () => {
    onRefresh(); // 새로고침 이벤트를 부모 컴포넌트로 전달
  };
  return (
    <>
      <TableHeaderContainer>
        <TableHeaderBase>
          <BasicButton text={buttonText} width="auto" justifyContent="center" />
          <div onClick={handleRefresh}>
            <Icon
              component={CachedIcon}
              style={{ fontSize: "36px", cursor: "pointer" }}
              onMouseEnter={(
                e: React.MouseEvent<SVGSVGElement, MouseEvent>
              ) => {
                const target = e.currentTarget;
                target.style.color = "#858585";
              }}
              onMouseLeave={(
                e: React.MouseEvent<SVGSVGElement, MouseEvent>
              ) => {
                const target = e.currentTarget;
                target.style.color = "white";
              }}
            />
          </div>
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
