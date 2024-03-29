import { styled } from "styled-components";

export const TableContainer = styled.div`
  margin: 0px 40px;
  background-color: #393e46;
  color: white;
  text-align: center;
  padding: 40px;
`;

export const TableHeaderContainer = styled.div`
  display: flex;
  font-weight: 400;
  padding: 15px 10px;
  border-bottom: 1px solid #00adb5;
`;

export const TableDataContainer = styled.div`
  display: flex;
  font-weight: 500;
  padding: 10px;
  border-bottom: 1px solid #00adb5;
  color: #00adb5;
  &:hover {
    background-color: #222831;
  }
  a {
    width: 10%;
    min-width: 100px;
  }
`;

export const QueueNameBase = styled.div`
  flex-grow: 1;
  text-align: left;
  color: white;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 90%;
`;

export const HeaderItem = styled.div`
  width: 10%;
  min-width: 100px;
`;
