import { theme } from "@/theme/theme";
import { fontThemeToCss } from "@/utils";
import styled from "styled-components";

export const TabListBase = styled.ul`
  display: flex;
  margin: 0px 40px;
  background: #222831;
  padding: 10px 40px;
`;

export const TabBase = styled.li`
  display: inline-block;
  box-sizing: border-box;
  min-width: 70px;
  padding: 10px;
  transition: 0.2s;
  word-break: keep-all;
  white-space: nowrap;
  color: #eeeeee;
  svg {
    transition: 0.2s;
  }
  &:hover {
    cursor: pointer;
  }
  &.selected {
    color: #00adb5;
    font-weight: 600;
    border-bottom: 4px solid #00adb5;
  }
`;

export const LabelWithIconBase = styled.span`
  display: flex;
  align-items: center;
  svg {
    margin-right: 8px;
  }
`;
export const TabContainer = styled.div`
  display: flex;
`;

export const NumberOfDataBase = styled.div<{ hasData: boolean }>`
  width: 40px;
  font-size: 14px;
  height: 20px;
  border-radius: 30px;
  font-weight: ${({ hasData }) => (hasData ? "600" : "400")};
  color: ${({ hasData }) => (hasData ? "#252525" : "#eeeeee")};
  background: ${({ hasData }) => (hasData ? "rgb(192, 255, 65)" : "#252525")};
  text-align: center;
`;
