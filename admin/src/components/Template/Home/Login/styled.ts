import styled, { keyframes, css } from "styled-components";

export const LoginContainer = styled.div`
  color: #eeeeee;
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
  flex-direction: column;
  justify-content: center;
  align-items: center;
`;

const gradientAnimation = keyframes`
  0% {
    background-position: 0% 50%;
  }
  50% {
    background-position: 100% 50%;
  }
  100% {
    background-position: 0% 50%;
  }
`;

export const LoginTitleBase = styled.div`
  font-size: 36px;
  margin-bottom: 24px;
  background: linear-gradient(to right, #ff6b6b, #b66dff);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  animation-name: ${gradientAnimation};
  animation-duration: 3s;
  animation-timing-function: linear;
  animation-iteration-count: infinite;
  background-size: 200% auto;
`;

const shakeAnimation = keyframes`
  0% { transform: translate(1px, 1px) rotate(0deg); }
  10% { transform: translate(-1px, -2px) rotate(-1deg); }
  20% { transform: translate(-3px, 0px) rotate(1deg); }
  30% { transform: translate(3px, 2px) rotate(0deg); }
  40% { transform: translate(1px, -1px) rotate(1deg); }
  50% { transform: translate(-1px, 2px) rotate(-1deg); }
  60% { transform: translate(-3px, 1px) rotate(0deg); }
  70% { transform: translate(3px, 1px) rotate(-1deg); }
  80% { transform: translate(-1px, -1px) rotate(1deg); }
  90% { transform: translate(1px, 2px) rotate(0deg); }
  100% { transform: translate(1px, -2px) rotate(-1deg); }
`;

export const LoginInputBase = styled.input<{ error?: string }>`
  width: 200px;
  height: 40px;
  padding: 8px;
  font-size: 16px;
  border: ${(props) => (props.error ? "2px solid salmon" : "1px solid #000")};
  border-radius: 4px;
  margin-right: 10px;
  animation: ${(props) =>
    props.error
      ? css`
          ${shakeAnimation} 0.5s linear
        `
      : "none"};
`;

export const LoginButtonBase = styled.button`
  width: 100px;
  height: 40px;
  margin-left: 8px;
  font-size: 16px;
  border: none;
  border-radius: 20px;
  cursor: pointer;
  background: linear-gradient(90deg, #ff6b6b, #b66dff);
  color: #fff;
  transition: all 0.5s ease;

  &:hover {
    transform: scale(1.05);
    background: linear-gradient(90deg, #b66dff, #ff6b6b);
  }

  &:active {
    transform: scale(0.95);
  }
`;
