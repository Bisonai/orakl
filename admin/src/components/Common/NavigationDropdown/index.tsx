import React, { ReactNode, useState } from "react";
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
    <Link href={href} style={{ width: "100%" }}>
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
              }}
            />
          ) : (
            <ExpandMoreIcon
              style={{
                background: "#eeeeee",
                borderRadius: "16px",
                marginTop: "3px",
                color: "#00adb5",
              }}
            />
          )}
        </span>
      </StyledAccordionHeader>
    </Link>
  );
};

const NavButton: React.FC<NavButtonProps> = ({ href, text }) => (
  <Link href={href}>
    <BasicButton text={text} />
  </Link>
);

export default function NavigationDropdown(): JSX.Element {
  const [isAccordionOpen, setIsAccordionOpen] = useState<IAccordionState>({
    configuration: true,
    bull: true,
    account: true,
  });

  function handleAccordionToggle(index: keyof IAccordionState) {
    setIsAccordionOpen((isOpen) => ({ ...isOpen, [index]: !isOpen[index] }));
  }

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
                <NavButton key={key} href={href} text={key} />
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
              {Object.entries(routes).map(([key, href]) => (
                <NavButton key={key} href={href} text={key} />
              ))}
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
