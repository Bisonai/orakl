import { theme } from "@/theme/theme";
import { fontThemeToCss } from "@/utils";
import styled from "styled-components";

export const TabListBase = styled.ul`
  display: flex;
  justify-content: space-around;
  margin: 0px 40px;
  background: #b6c39f;
  padding: 40px;
`;

export const TabBase = styled.li`
  display: inline-block;
  box-sizing: border-box;
  min-width: 70px;
  padding: 10px 20px;
  transition: 0.2s;
  word-break: keep-all;
  white-space: nowrap;
  svg {
    transition: 0.2s;
  }
  &:hover {
    cursor: pointer;
  }
  &.selected {
    color: #212a3e;
    font-weight: 600;
    border-bottom: 4px solid #212a3e;
  }
`;

export const LabelWithIconBase = styled.span`
  display: flex;
  align-items: center;
  svg {
    margin-right: 8px;
  }
`;
