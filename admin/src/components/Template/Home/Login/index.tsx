import { setCookie } from "@/lib/cookies";
import authenticatedAxios from "@/lib/authenticatedAxios";
import { useEffect, useRef, useState } from "react";
import {
  LoginButtonBase,
  LoginContainer,
  LoginInputBase,
  LoginTitleBase,
} from "./styled";
import { AxiosError } from "axios";

interface LoginPageProps {
  onLogin: () => void;
}

export const LoginPage: React.FC<LoginPageProps> = ({ onLogin }) => {
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    inputRef.current?.focus();
  }, [password]);

  const handleLogin = async () => {
    try {
      const response = await authenticatedAxios.post(
        `${process.env.NEXT_PUBLIC_API_QUEUES_URL}/auth/login`,
        {
          password: password,
        }
      );

      if (response.status === 200 && response.data.access_token) {
        setCookie("token", response.data.access_token);
        onLogin();
        setError("");
      } else {
        setError("Invalid password. Please try again.");
        setPassword("");
      }
    } catch (error) {
      const axiosError = error as AxiosError;
      if (axiosError.isAxiosError) {
        if (axiosError.response) {
          console.log("Error data:", axiosError.response.data);
          setError("Invalid password. Please try again.");
          setPassword("");
        } else {
          console.log("Error:", axiosError.message);
          setError("An error occurred. Please try again.");
          setPassword("");
        }
      } else {
        console.log("Unknown error:", error);
        setError("An unknown error occurred. Please try again.");
        setPassword("");
      }
    }
  };

  return (
    <LoginContainer>
      <LoginTitleBase>
        {error
          ? "You entered the wrong password Please enter it again"
          : "Please enter the password"}
      </LoginTitleBase>
      <div>
        <LoginInputBase
          type="password"
          ref={inputRef}
          value={password}
          error={error}
          onChange={(e) => setPassword(e.target.value)}
        />
        <LoginButtonBase onClick={handleLogin}>Login</LoginButtonBase>
      </div>
    </LoginContainer>
  );
};
