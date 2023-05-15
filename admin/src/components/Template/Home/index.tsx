import NavigationDropdown from "@/components/Common/NavigationDropdown";
import React from "react";
import BullMonitor from "../BullMonitor";
import { styled } from "styled-components";

const MainContainer = styled.div`
  display: flex;
`;

const ContentContainer = styled.div`
  flex: 1;
`;
export default function HomeTemplate(): JSX.Element {
  return (
    <>
      <MainContainer>
        <NavigationDropdown />

        <ContentContainer>
          <BullMonitor />
        </ContentContainer>
      </MainContainer>
    </>
  );
}
