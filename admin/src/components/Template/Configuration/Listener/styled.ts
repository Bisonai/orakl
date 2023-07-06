import styled from "styled-components";

export const Container = styled.div`
  width: 80%;
  margin: 0 auto;
`;

export const HeaderBase = styled.div`
  display: flex;
  justify-content: space-between;
  align-items: center;
`;

export const TitleBase = styled.h2`
  color: #fff;
  font-size: 48px;
  padding-top: 60px;
  font-weight: bold;
`;

export const TableBase = styled.div`
  width: 100%;
  padding: 20px;
  margin: 40px 0px;
  border-collapse: collapse;
  color: #fff;
  background: #323a47;
  border-radius: 12px;
  display: flex;
  flex-direction: column;
`;

export const TableRow = styled.div`
  display: flex;
  justify-content: start;
  align-items: center;
`;

export const TableLabel = styled.div`
  min-width: 200px;
  padding: 10px 0;
`;

export const TableData = styled.div`
  flex-grow: 1;
  padding: 10px 0;
  word-wrap: break-word;
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
`;
