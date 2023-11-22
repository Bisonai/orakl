import styled, { css } from "styled-components";

export const ScrollButton = styled.button`
  position: fixed;
  right: 50px;
  bottom: 50px;
  border-radius: 50px;
  width: 50px;
  height: 50px;
  cursor: pointer;
  border: 3px solid #02c7d1;
  box-shadow: 0px 0px 10px 3px rgba(0, 0, 0, 0.55);

  &:hover {
    background-color: #02c7d1;
    color: white;
  }

  &:active {
    transform: scale(0.95);
  }
`;
