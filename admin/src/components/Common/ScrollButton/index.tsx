import React, { useEffect, useState } from "react";
import styled from "styled-components";
import { ScrollButton } from "./styled";

function ScrollToTopButton() {
  const [isVisible, setIsVisible] = useState(false);
  const [scrollPercent, setScrollPercent] = useState(0);

  const checkScrollTopAndPercent = () => {
    const totalScroll = document.body.scrollHeight - window.innerHeight;
    const scrollPosition = window.pageYOffset;
    const scrollPercent = (scrollPosition / totalScroll) * 100;

    setScrollPercent(Math.round(scrollPercent));

    if (!isVisible && scrollPosition > 400) {
      setIsVisible(true);
    } else if (isVisible && scrollPosition <= 400) {
      setIsVisible(false);
    }
  };

  const scrollToTop = () => {
    window.scrollTo({
      top: 0,
      behavior: "smooth",
    });
  };

  useEffect(() => {
    window.addEventListener("scroll", checkScrollTopAndPercent);
    return () => window.removeEventListener("scroll", checkScrollTopAndPercent);
  });

  return (
    <ScrollButton onClick={scrollToTop} show={isVisible}>
      {scrollPercent}%
    </ScrollButton>
  );
}

export default ScrollToTopButton;
