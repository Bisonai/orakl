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
import { configRoutes, routes } from "@/utils/route";
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
              <Link href={configRoutes.chain}>
                <Button text="Chain" />
              </Link>
              <Link href={configRoutes.service}>
                <Button text="Service" />
              </Link>
              <Link href={configRoutes.listener}>
                <Button text="Listener" />
              </Link>
              <Link href={configRoutes.vrfKeys}>
                <Button text="VRF Keys" />
              </Link>
              <Link href={configRoutes.adapter}>
                <Button text="Adapter" />
              </Link>
              <Link href={configRoutes.aggregator}>
                <Button text="Aggregator" />
              </Link>
              <Link href={configRoutes.reporter}>
                <Button text="Reporter" />
              </Link>
              <Link href={configRoutes.fetcher}>
                <Button text="Fetcher" />
              </Link>
              <Link href={configRoutes.delegator}>
                <Button text="Delegator" />
              </Link>
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
