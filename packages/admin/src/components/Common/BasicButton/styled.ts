import { styled } from "styled-components";

export const ButtonContainer = styled.div<{ textAlign?: string }>`
  text-align: ${({ textAlign }) => textAlign};
`;
