import { IObject } from "./types";
import { BREAKPOINTS } from "./constants";
import { CSSProperties } from "react";

export const noTypeCheck = (x: any) => x;

export const camelToHypen = (camel: string) =>
  camel.replace(/[A-Z]/g, (m) => "-" + m.toLowerCase());

export const reactCssToNormalCss = (css: CSSProperties) =>
  Object.entries(css)
    .map(([prop, value]) => `${camelToHypen(prop)}: ${value};`)
    .join("\n");

export const parseFontThemeToReactCss = (
  fontTheme: IObject<string | number>
) => {
  return Object.keys(fontTheme).reduce((result, key) => {
    const value = fontTheme[key];
    let newValue = value;
    if (key === "fontWeight") {
      if (typeof value === "string") {
        newValue = fontWeightMap[value];
      }
    } else if (key === "fontFamily") {
      newValue = (value as string).replace(
        "Elice DigitalBaeum",
        "__Elice_5c8c8b"
      );
    } else if (typeof value === "string" && !["animationName"].includes(key)) {
      newValue = value.toLowerCase();
    } else if (typeof value === "number") {
      newValue = `${value}px`;
    }
    return { ...result, [key]: newValue };
  }, {} as CSSProperties);
};

export const fontThemeToCss = (fontTheme: IObject<string | number>) =>
  reactCssToNormalCss(parseFontThemeToReactCss(fontTheme));

export const getResponsiveCss = (
  desktop: IObject<string | number>,
  tablet: IObject<string | number>,
  mobile: IObject<string | number>
) =>
  `
    @media (min-width: ${BREAKPOINTS.DESKTOP + 1}px) {
        ${fontThemeToCss(desktop)}
    }
    @media (min-width: ${BREAKPOINTS.TABLET + 1}px) and (max-width: ${
    BREAKPOINTS.DESKTOP
  }px) {
        ${fontThemeToCss(tablet)}
    }
    @media (max-width: ${BREAKPOINTS.TABLET}px) {
        ${fontThemeToCss(mobile)}
    }
    `;

export const fontWeightMap: IObject<number> = {
  Black: 900,
  Bold: 700,
  SemiBold: 600,
  Medium: 500,
  Regular: 400,
};

export const duplicateArray = <T>(arr: T[], times: number): T[] => {
  return [...Array(times)].reduce((result) => [...result, ...arr], []);
};

export function formatUnixTimestamp(timestamp: string | number) {
  const date = new Date((timestamp as number) * 1000);
  const formattedDate = date.toUTCString();
  return formattedDate;
}

export function getRandomColor(address = "") {
  let hash = 0;
  for (let i = 0; i < (address || "").length; i++) {
    hash = address.toLowerCase().charCodeAt(i) + ((hash << 5) - hash);
  }
  const hue = hash % 360;
  const saturation = 50;
  const lightness = 70;
  return `hsl(${hue}, ${saturation}%, ${lightness}%)`;
}

export const shortenString = (
  str: string,
  startLength: number,
  endLength: number
) => {
  const newStr = `${str}`;
  if (!newStr?.length) {
    return "";
  }
  if (newStr.length <= startLength + endLength) {
    return newStr;
  }
  return (
    newStr.substring(0, startLength) +
    "..." +
    newStr.substring(newStr.length - endLength)
  );
};

export const copyToClipboard = (text: string) => {
  navigator.clipboard.writeText(text);
};
export function getRandomProfileImage(address: string | null | undefined) {
  const index = address ? parseInt(`${address}`.slice(-2), 16) % 160 : 0;
  return `/image/profile_400/profile_${index + 1}.jpg`;
}

export const parseSignerErrorMessage = (message: string) => {
  if (message) {
    return "ERROR: " + message.split(" (action")[0].slice(0, 100) + "...";
  }
  return "Unexpected error";
};

export const isMobile = () => {
  return typeof window === "object" &&
    window.navigator.userAgent.match(
      /Android|iPhone|iPad|iPod|BlackBerry|IEMobile|Opera Mini/i
    )
    ? true
    : false;
};

export const hexValue = (value: number) => {
  return `0x${value.toString(16)}`;
};

export function validatePriceInput(input: string): boolean {
  if (!/^\d*\.?\d*$/.test(input)) {
    return false;
  }
  if (parseFloat(input) < 0) {
    return false;
  }
  return true;
}

export function cleanErrorMessage(
  errorMessage: string | undefined
): string | undefined {
  if (!errorMessage) {
    return undefined;
  }
  const cleanedMessage = errorMessage.split("(")[0].trim();
  return cleanedMessage;
}

export function getCleanedTransactionHash(transactionHash: string): string {
  const index = transactionHash.indexOf("/");
  const cleanedHash =
    index === -1 ? transactionHash : transactionHash.slice(0, index);
  return cleanedHash;
}
export function formatLongString(s: any, maxLineLength: number) {
  let result = "";
  for (let i = 0, line = 0; i < s.length; i += maxLineLength, line++) {
    if (line > 0) {
      result += "   ";
    }
    result += s.substring(i, i + maxLineLength) + "\n";
  }
  return result;
}
