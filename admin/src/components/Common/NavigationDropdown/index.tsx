import React, { ReactNode, useEffect, useState } from "react";
import ExpandMoreIcon from "@material-ui/icons/ExpandMore";
import ExpandLessIcon from "@material-ui/icons/ExpandLess";
import {
  NavDropdownContainer,
  AccordionContainer,
  AccordionItem,
  AccordionHeader as StyledAccordionHeader,
  AccordionContent,
} from "./styled";
import Link from "next/link";
import { configRoutes, routes } from "@/utils/route";
import { IAccordionState } from "@/utils/types";
import BasicButton from "../BasicButton";

type AccordionHeaderProps = {
  isOpen: boolean;
  onClick: () => void;
  children: ReactNode;
  href: string;
};

type NavButtonProps = {
  href: string;
  text: string;
  disabled?: boolean;
  selected?: boolean;
  onClick?: () => void;
};

const AccordionHeader: React.FC<AccordionHeaderProps> = ({
  isOpen,
  onClick,
  children,
  href,
}) => {
  const handleIconClick = (e: React.MouseEvent<HTMLSpanElement>) => {
    e.stopPropagation();
    e.preventDefault();
    onClick();
  };

  return (
    <Link href={href} style={{ width: "90%" }}>
      <StyledAccordionHeader>
        {children}
        <span onClick={handleIconClick}>
          {isOpen ? (
            <ExpandLessIcon
              style={{
                background: "#eeeeee",
                borderRadius: "16px",
                marginTop: "3px",
                color: "#00adb5",
                boxShadow: "0px 3px 6px rgba(0,0,0,0.3)",
              }}
            />
          ) : (
            <ExpandMoreIcon
              style={{
                background: "#eeeeee",
                borderRadius: "16px",
                marginTop: "3px",
                color: "#00adb5",
                boxShadow: "0px 3px 5px rgba(0,0,0,0.3)",
              }}
            />
          )}
        </span>
      </StyledAccordionHeader>
    </Link>
  );
};

const NavButton: React.FC<NavButtonProps> = ({
  href,
  text,
  disabled,
  selected,
  onClick,
}) => (
  <Link href={href}>
    <BasicButton
      text={text}
      disabled={disabled}
      selected={selected}
      onClick={onClick}
    />
  </Link>
);

export default function NavigationDropdown(): JSX.Element {
  const [isAccordionOpen, setIsAccordionOpen] = useState<IAccordionState>({
    configuration: true,
    bull: true,
    account: true,
  });

  const [currentPath, setCurrentPath] = useState("");

  function handleAccordionToggle(index: keyof IAccordionState) {
    setIsAccordionOpen((isOpen) => ({ ...isOpen, [index]: !isOpen[index] }));
  }
  const handleNavButtonClick = (selectedPath: string) => {
    setCurrentPath(selectedPath);
  };

  return (
    <NavDropdownContainer>
      <AccordionContainer>
        <AccordionItem>
          <AccordionHeader
            href="/"
            isOpen={isAccordionOpen.configuration}
            onClick={() => handleAccordionToggle("configuration")}
          >
            Configuration
          </AccordionHeader>
          {isAccordionOpen.configuration && (
            <AccordionContent>
              {Object.entries(configRoutes).map(([key, href]) => (
                <NavButton
                  key={key}
                  href={href}
                  text={key}
                  disabled={key === "fetcher"}
                  selected={
                    configRoutes[key as keyof typeof configRoutes] ===
                    currentPath
                  }
                  onClick={() => handleNavButtonClick(href)}
                />
              ))}
            </AccordionContent>
          )}
        </AccordionItem>
        <AccordionItem>
          <AccordionHeader
            href="/bullmonitor"
            isOpen={isAccordionOpen.bull}
            onClick={() => handleAccordionToggle("bull")}
          >
            Bull Monitor
          </AccordionHeader>
          {isAccordionOpen.bull && (
            <AccordionContent>
              {Object.entries(routes).map(([key, href]) => {
                return (
                  <NavButton
                    key={key}
                    href={href}
                    text={key}
                    disabled={key === "fetcher" || key === "settings"}
                    selected={
                      routes[key as keyof typeof routes] === currentPath
                    }
                    onClick={() => handleNavButtonClick(href)}
                  />
                );
              })}
            </AccordionContent>
          )}
        </AccordionItem>
        <AccordionItem>
          <AccordionHeader
            href="/"
            isOpen={isAccordionOpen.account}
            onClick={() => handleAccordionToggle("account")}
          >
            Account Balance
          </AccordionHeader>
          {isAccordionOpen.account && <AccordionContent />}
        </AccordionItem>
      </AccordionContainer>
    </NavDropdownContainer>
  );
}
