import { keyframes, styled } from "styled-components";

export const NavDropdownContainer = styled.div`
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 200px;
  font-size: 16px;
  background: #9ba4b5;
  padding: 40px 0px 40px 20px;
`;

export const AccordionContainer = styled.div`
  display: flex;
  flex-direction: column;
  height: 100%;
`;

export const AccordionItem = styled.div`
  display: flex;
  flex-direction: column;
  margin-bottom: 8px;
`;

export const AccordionHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  color: white;
  height: 50px;
  background-color: #212a3e;
  cursor: pointer;
`;

const slideDown = keyframes`
  from {
    opacity: 0;
    transform: translateY(-10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
`;

export const AccordionContent = styled.div`
  display: flex;
  flex-direction: column;

  overflow: hidden;
  animation: slideDown 0.3s ease;
  a {
    width: 100%;
  }
  Button {
    margin: 2px 0px;
    font-size: 12px;
    font-weight: 400;
  }
`;

// export const Icon = styled.span`
//   font-size: 1.2rem;
//   margin-right: 8px;
// `;
