"use client";

import { useServerInsertedHTML } from "next/navigation";

import { useStyledComponentsRegistry } from "./styled-components";
import React from "react";

export default function RootStyleRegistry({
  children,
}: {
  children: React.ReactNode;
}) {
  const [StyledComponentsRegistry, styledComponentsFlushEffect] =
    useStyledComponentsRegistry();

  useServerInsertedHTML(() => <>{styledComponentsFlushEffect()}</>);

  return <StyledComponentsRegistry>{children}</StyledComponentsRegistry>;
}
