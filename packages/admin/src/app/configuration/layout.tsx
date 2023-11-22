"use client";

import DimmedPopupProvider from "@/components/Common/DimmedPopupProvider";

export default function ConfigurationLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <>
      <DimmedPopupProvider>{children}</DimmedPopupProvider>
    </>
  );
}
