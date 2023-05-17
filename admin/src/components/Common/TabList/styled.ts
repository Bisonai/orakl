import { theme } from "@/theme/theme";
import { fontThemeToCss } from "@/utils";
import styled from "styled-components";

export const TabListBase = styled.ul`
  display: flex;
  justify-content: space-around;
  margin: 0px 40px;
  background: #d9e5ef;
`;

export const TabBase = styled.li`
  display: inline-block;
  box-sizing: border-box;

  min-width: 100px;
  padding: 20px;

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
    color: white;
    border-bottom: 5px solid black;
  }
`;

export const LabelWithIconBase = styled.span`
  display: flex;
  align-items: center;
  svg {
    margin-right: 8px;
  }
`;
