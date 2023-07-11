import styled, { keyframes } from "styled-components";

export const DetailTableHeaderBase = styled.div`
  display: flex;
  justify-content: flex-end;
  padding-top: 30px;
  margin-right: 50px;
`;

export const TablePagination = styled.div`
  color: white;
`;
export const DetailTableContainer = styled.div`
  background: #222831;
  margin: 40px;
  border-radius: 4px;
  border: 2px solid #02c7d1;
`;
export const DetailHeaderBase = styled.div`
  display: flex;
  flex-direction: row;
  padding: 20px;
  justify-content: space-between;
  border-bottom: 1px solid #02c7d1;
`;
export const ServiceNameBase = styled.div`
  color: white;
  font-weight: 600;
  color: white;
  font-size: 20px;
  padding: 10px 20px;
`;

export const JobIdBase = styled.div`
  color: white;
  font-weight: 400;
  color: white;
  font-size: 14px;
  padding: 0px 20px;
`;
export const DetailTableBase = styled.div`
  display: flex;
`;

export const DetailLeftBase = styled.div`
  width: 300px;
  display: flex;
  flex-direction: column;
  justify-content: space-around;
  border-right: 1px solid #02c7d1;
`;
export const TimeTableTextBase = styled.div`
  padding: 40px 20px;
  color: white;
  font-size: 14px;
  font-weight: 300;
`;

export const DetailRightBase = styled.div`
  width: 100%;
  padding: 20px;
  background: #222831;
`;

export const DetailTabBase = styled.div`
  display: flex;
  padding: 20px;
`;
export const CodeSnippetBase = styled.div`
  color: #00adb5;
  pre {
    font-size: 14px;
    line-height: 18px;
    font-size: 14px;
    max-width: 1000px;
    overflow: auto;
    font-family: inherit;
  }
`;

const blinkAnimation = keyframes`
  0% { opacity: 1; }
  50% { opacity: 0.5; }
  100% { opacity: 1; }
`;
export const IsLoadingBase = styled.div`
  width: 100%;
  height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 20px;
  animation: ${blinkAnimation} 2s linear infinite;
`;

export const ErrorMessageBase = styled.div`
  width: 100%;
  height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 20px;
`;

export const NoDataAvailableBase = styled.div`
  width: 100%;
  height: 300px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 20px;
`;
