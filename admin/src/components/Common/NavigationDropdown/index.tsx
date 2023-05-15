import React, { useState } from "react";
import styled from "styled-components";
import Button from "../BasicButton";
import {
  NavDropdownContainer,
  AccordionContainer,
  AccordionItem,
  AccordionHeader,
  AccordionContent,
  Icon,
} from "./styled";

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
            <Icon>{isAccordionOpen[0] ? "-" : "+"}</Icon>
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
            <Icon>{isAccordionOpen[1] ? "-" : "+"}</Icon>
          </AccordionHeader>
          {isAccordionOpen[1] && (
            <AccordionContent>
              <Button text="VRF" />
              <Button text="Request Response" />
              <Button text="Aggregator" />
              <Button text="Fetcher" />
              <Button text="Setting" />
            </AccordionContent>
          )}
        </AccordionItem>
        <AccordionItem>
          <AccordionHeader onClick={() => handleAccordionToggle(2)}>
            Account Balance
            <Icon>{isAccordionOpen[2] ? "-" : "+"}</Icon>
          </AccordionHeader>
          {isAccordionOpen[2] && <AccordionContent />}
        </AccordionItem>
      </AccordionContainer>
    </NavDropdownContainer>
  );
}
