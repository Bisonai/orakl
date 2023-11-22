import styled from "styled-components";

export const PopupContainer = styled.div`
  position: fixed;
  top: 0;
  left: 0;
  width: 100vw;
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: rgba(0, 0, 0, 0.5);
  z-index: 9999;
`;

export const PopupContent = styled.div<{ size?: string }>`
  width: ${(props) => {
    switch (props.size) {
      case "small":
        return "300px";
      case "medium":
        return "450px";
      case "large":
        return "550px";
      default:
        return "500px";
    }
  }};
  height: ${(props) => {
    switch (props.size) {
      case "small":
        return "250px";
      case "medium":
        return "220px";
      case "large":
        return "auto";
      default:
        return "350px";
    }
  }};
  border-radius: 16px;
  text-align: center;
  background: #b7c0bb;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
`;

export const PopupTitle = styled.h2<{ size?: string }>`
  padding: 24px 40px;
  letter-spacing: -0.02em;
  /* height: 50px; */
  /* margin-top: 20px; */
  display: flex;
  align-items: center;
  justify-content: center;
  color: #333333;
  font-size: ${(props) => {
    switch (props.size) {
      case "small":
        return "24px";
      case "medium":
        return "26px";
      case "large":
        return "32px";
      default:
        return "24px";
    }
  }};
`;

export const PopupButton = styled.button`
  display: flex;
  width: 100%;
  height: 60px;
  margin-top: 20px;
  justify-content: space-between;
  align-items: center;
  border: none;
  border-bottom-left-radius: 16px;
  border-bottom-right-radius: 16px;
  position: relative;
  .btn-cancel {
    border: none;
    width: 50%;
    height: 100%;
    cursor: pointer;
    font-size: 16px;
    background: #282828;
    color: #ffffff;
    border-bottom-left-radius: 12px;
  }

  .btn-confirm {
    border: none;
    cursor: pointer;
    width: 50%;
    height: 100%;
    font-size: 16px;
    background: #71ebff;
    border-bottom-right-radius: 12px;
    &::before {
      content: "";
      position: absolute;
      top: 0;
      bottom: 0;
      right: 50%;
      width: 1px;
    }
  }

  .btn {
    cursor: pointer;
    width: 100%;
    height: 100%;
    border: none;
    border-bottom-left-radius: 16px;
    border-bottom-right-radius: 16px;
  }
`;

export const PopupForm = styled.form`
  background: #b7c0bb;
  display: flex;
  flex-direction: column;
  align-items: center;
`;
export const FormInputBase = styled.input`
  width: 90%;
  height: 40px;

  padding: 10px 15px;
  margin-bottom: 10px;
  box-sizing: border-box;
  border: 1px solid #cccccc;
  border-radius: 12px;
  font-size: 16px;
  color: #ffffff;
  background-color: #333333;
  outline: none;
  &:focus {
    border-color: #0080ff;
    box-shadow: 0 0 5px rgb(114, 250, 147);
  }
  ::placeholder {
    color: #cccccc;
  }
`;
