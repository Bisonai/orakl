import styled from "styled-components";

export const ScrollButton = styled.button<{ show: boolean }>`
  position: fixed;
  right: 50px;
  bottom: 50px;
  border-radius: 50px;
  width: 50px;
  height: 50px;
  cursor: pointer;
  border: 3px solid #02c7d1;
  display: ${(props) => (props.show ? "block" : "none")};
`;
