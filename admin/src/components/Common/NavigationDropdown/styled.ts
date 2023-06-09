import { keyframes, styled } from "styled-components";

export const NavDropdownContainer = styled.div`
  display: flex;
  flex-direction: column;
  height: 100%;
  width: 200px;
  font-size: 16px;
  background: #222831;
  padding: 40px 0px 40px 20px;
`;

export const AccordionContainer = styled.div`
  display: flex;
  flex-direction: column;
  height: 100%;
  min-width: 200px;
`;

export const AccordionItem = styled.div`
  display: flex;
  flex-direction: column;
  margin-bottom: 8px;
  border-radius: 4px;
`;

export const AccordionHeader = styled.div`
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 16px;
  color: #eeeeee;
  border-radius: 4px;
  height: 50px;
  font-weight: 600;
  background-color: #00adb5;
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
