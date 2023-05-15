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

export default function NavigationDropdown(): JSX.Element {
  const [isAccordionOpen, setIsAccordionOpen] = useState([true, true, true]);

  function handleAccordionToggle(index: number) {
    setIsAccordionOpen((isOpen) => ({ ...isOpen, [index]: !isOpen[index] }));
  }

  return (
    <NavDropdownContainer>
      <AccordionContainer>
        <AccordionItem>
          <AccordionHeader onClick={() => handleAccordionToggle(0)}>
            Configuration
            <span>{isAccordionOpen[0] ? "-" : "+"}</span>
          </AccordionHeader>
          {isAccordionOpen[0] && (
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
          <AccordionHeader onClick={() => handleAccordionToggle(1)}>
            Bull Monitor
            <span>{isAccordionOpen[1] ? "-" : "+"}</span>
          </AccordionHeader>
          {isAccordionOpen[1] && (
            <AccordionContent>
              <Link href={`/bullmonitor/vrf`}>
                <BasicButton text="VRF" />
              </Link>
              <Link href={`/bullmonitor/request-response`}>
                <BasicButton text="Request Response" />
              </Link>
              <Link href={`/bullmonitor/aggregator`}>
                <BasicButton text="Aggregator" />
              </Link>
              <BasicButton text="Fetcher" />
              <BasicButton text="Setting" />
            </AccordionContent>
          )}
        </AccordionItem>
        <AccordionItem>
          <AccordionHeader onClick={() => handleAccordionToggle(2)}>
            Account Balance
            <span>{isAccordionOpen[2] ? "-" : "+"}</span>
          </AccordionHeader>
          {isAccordionOpen[2] && <AccordionContent />}
        </AccordionItem>
      </AccordionContainer>
    </NavDropdownContainer>
  );
}
