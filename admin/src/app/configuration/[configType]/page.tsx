"use client";

import ConfigurationDetailTemplate from "@/components/Template/Configuration/detail";

export default function ConfigurationDetail({
  params,
}: {
  params: { configType: string };
}) {
  return (
    <div>
      <ConfigurationDetailTemplate configType={params.configType} />
    </div>
  );
}
