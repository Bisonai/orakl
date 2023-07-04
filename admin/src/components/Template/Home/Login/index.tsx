import { setCookie } from "@/lib/cookies";
import authenticatedAxios from "@/lib/authenticatedAxios";
import { useState } from "react";
import { LoginButtonBase, LoginContainer, LoginInputBase } from "./styled";
import axios, { AxiosError } from "axios";
import { ToastType } from "@/utils/types";
import { useToastContext } from "@/hook/useToastContext";

interface LoginPageProps {
  onLogin: () => void;
}

export const LoginPage: React.FC<LoginPageProps> = ({ onLogin }) => {
  const [password, setPassword] = useState("");
  const { addToast } = useToastContext();
  const handleLogin = async () => {
    try {
      const response = await authenticatedAxios.post(
        "http://localhost:8888/auth/login",
        {
          password: password,
        }
      );

      if (response.status === 200 && response.data.access_token) {
        setCookie("token", response.data.access_token);
        onLogin();
      } else {
      }
    } catch (error) {
      const axiosError = error as AxiosError;
      if (axiosError.isAxiosError) {
        if (axiosError.response) {
          console.log("Error data:", axiosError.response.data);
        } else {
          console.log("Error:", axiosError.message);
        }
      } else {
        console.log("Unknown error:", error);
      }
    }
  };

  return (
    <LoginContainer>
      <LoginInputBase
        type="password"
        value={password}
        onChange={(e) => setPassword(e.target.value)}
      />
      <LoginButtonBase onClick={handleLogin}>Login</LoginButtonBase>
    </LoginContainer>
  );
};
