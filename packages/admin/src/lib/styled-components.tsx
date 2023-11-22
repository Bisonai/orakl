import React from "react";
import { ServerStyleSheet, StyleSheetManager } from "styled-components";

export function useStyledComponentsRegistry() {
  const [styledComponentsStyleSheet] = React.useState(
    () => new ServerStyleSheet()
  );

  const styledComponentsFlushEffect = () => {
    const styles = styledComponentsStyleSheet.getStyleElement();
    // @ts-ignore
    styledComponentsStyleSheet.instance.clearTag();
    return <>{styles}</>;
  };

  function StyledComponentsRegistry({ children }: any) {
    if (typeof window !== "undefined") {
      return children;
    }
    return (
      <StyleSheetManager sheet={styledComponentsStyleSheet.instance}>
        {children as React.ReactElement}
      </StyleSheetManager>
    );
  }
  return [StyledComponentsRegistry, styledComponentsFlushEffect] as const;
}
