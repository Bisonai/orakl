"use client";

import { ThemeProvider } from "@mui/material";
import "../theme/globals.css";
import { Inter } from "next/font/google";
import { theme } from "@/theme/theme";
import QueryClientProviders from "./provider";
import RootStyleRegistry from "@/lib/RootStyleRegistry";
import Header from "@/components/Common/Header";
import NavigationDropdown from "@/components/Common/NavigationDropdown";
import { MainContainer, Container } from "./styled";
import ToastContextProvider from "@/components/Common/ToastProvider";
import TabContextProvider from "@/components/Common/TabContextProvider";

const inter = Inter({ subsets: ["latin"] });

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head></head>
      <RootStyleRegistry>
        <QueryClientProviders>
          <ThemeProvider theme={theme}>
            <body className={inter.className}>
              <ToastContextProvider>
                <TabContextProvider initTab={"active"}>
                  <Header />
                  <MainContainer>
                    <NavigationDropdown />
                    <Container> {children}</Container>
                  </MainContainer>
                </TabContextProvider>
              </ToastContextProvider>
            </body>
          </ThemeProvider>
        </QueryClientProviders>
      </RootStyleRegistry>
    </html>
  );
}
