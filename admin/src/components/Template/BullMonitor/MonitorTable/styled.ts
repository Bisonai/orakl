import { styled } from "styled-components";

export const TableContainer = styled.div`
  margin: 0px 40px;
  background-color: #393e46;
  color: white;
  text-align: center;
  padding: 40px;
`;

export const TableHeaderContainer = styled.div`
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 10px;
  font-weight: 400;
  padding: 15px 10px;
  border-bottom: 1px solid #00adb5;
`;

export const TableDataContainer = styled.div`
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 5px;
  font-weight: 500;
  padding: 10px;
  border-bottom: 1px solid #00adb5;
  color: #00adb5;
  &:hover {
    background-color: #394867;
  }
`;

export const QueueNameBase = styled.div`
  min-width: 300px;
  text-align: left;
  color: white;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
`;
