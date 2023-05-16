import React, { useState } from "react";
import styled from "styled-components";
import Button from "../BasicButton";
import {
  NavDropdownContainer,
  AccordionContainer,
  AccordionItem,
  AccordionHeader,
  AccordionContent,
} from "./styled";
import Link from "next/link";
import BasicButton from "../BasicButton";
import { routes } from "@/utils/route";
import { IAccordionState } from "@/utils/types";

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
            onClick={() => handleAccordionToggle("configuration")}
          >
            Configuration
            <span>{isAccordionOpen.configuration ? "-" : "+"}</span>
          </AccordionHeader>
          {isAccordionOpen.configuration && (
            <AccordionContent>
              <Button text="Chain" />
              <Button text="Service" />
              <Button text="Listener" />
              <Button text="VRF Keys" />
              <Button text="Adapter" />
              <Button text="Aggregator" />
              <Button text="Reporter" />
              <Button text="Fetcher" />
              <Button text="Delegator" />
            </AccordionContent>
          )}
        </AccordionItem>
        <AccordionItem>
          <AccordionHeader onClick={() => handleAccordionToggle("bull")}>
            Bull Monitor
            <span>{isAccordionOpen.bull ? "-" : "+"}</span>
          </AccordionHeader>
          {isAccordionOpen.bull && (
            <AccordionContent>
              <Link href={routes.vrf}>
                <BasicButton text="VRF" />
              </Link>
              <Link href={routes["request-response"]}>
                <BasicButton text="Request Response" />
              </Link>
              <Link href={`${routes.aggregator}`}>
                <BasicButton text="Aggregator" />
              </Link>
              <Link href={routes.fetcher}>
                <BasicButton text="Fetcher" />
              </Link>
              <Link href={routes.settings}>
                <BasicButton text="Setting" />
              </Link>
            </AccordionContent>
          )}
        </AccordionItem>
        <AccordionItem>
          <AccordionHeader onClick={() => handleAccordionToggle("account")}>
            Account Balance
            <span>{isAccordionOpen.account ? "-" : "+"}</span>
          </AccordionHeader>
          {isAccordionOpen.account && <AccordionContent />}
        </AccordionItem>
      </AccordionContainer>
    </NavDropdownContainer>
  );
}
