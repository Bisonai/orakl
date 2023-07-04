import styled from "styled-components";

export const LoginContainer = styled.div`
  z-index: 9998;
  position: fixed;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  background: #141619;
  width: 100%;
  height: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
`;

export const LoginInputBase = styled.input`
  width: 200px;
  height: 40px;
  padding: 8px;
  font-size: 16px;
  border: 1px solid #ccc;
  border-radius: 4px;
`;

export const LoginButtonBase = styled.button`
  width: 100px;
  height: 40px;
  margin-left: 8px;
  font-size: 16px;
  border: 1px solid #ccc;
  border-radius: 4px;
  cursor: pointer;
`;
