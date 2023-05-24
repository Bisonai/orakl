import styled from "styled-components";

export const ToastWrap = styled.div`
  border: 1px solid cornflowerblue;
  pointer-events: visible;
  animation: slideDownFadeIn 1s ease forwards;
  width: 333px;
  height: 76px;
  display: flex;
  border-radius: 8px;
  background-color: #222831;
  justify-content: space-between;
  align-items: center;
  flex-direction: row;
  padding: 0px 24px;
  transition: all 0.5s ease;
  @keyframes slideDownFadeIn {
    0% {
      transform: translateY(-100%);
      opacity: 0;
    }
    100% {
      transform: translateY(0);
      opacity: 1;
    }
  }

  @keyframes slideDownFadeOut {
    0% {
      transform: translateY(0);
      opacity: 1;
    }
    50% {
      opacity: 0.3;
    }
    100% {
      transform: translateY(200%);
      opacity: 0;
    }
  }
  &.fadeOut {
    animation: slideDownFadeOut 0.5s ease forwards;
  }
`;
export const CloseButtonWrap = styled.div`
  cursor: pointer;
  border-radius: 8px;
  width: 24px;
  height: 24px;
  display: flex;
  justify-content: center;
  align-items: center;
`;
export const TextWrap = styled.div`
  display: flex;
  flex-direction: column;
`;

export const TextTitleBase = styled.div`
  color: #eeeeee;
`;

export const TextBase = styled.div`
  line-height: 15px;
  font-size: 12px;
  padding-top: 4px;
  max-width: 250px;
  color: #eeeeee;
`;
