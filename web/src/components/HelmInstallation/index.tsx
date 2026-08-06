import { ReactElement } from "react";
import CodeBlock from "@theme/CodeBlock";

type HelmInstallationProps = {
  chartVersion?: string;
};

// HelmInstallationSnippet is the Helm incantation to install a specific
// or latest available version of the Barman Cloud Plugin.
export function HelmInstallationSnippet({
  chartVersion,
}: HelmInstallationProps): ReactElement<null> {
  const versionArg = chartVersion ? `\n  --version ${chartVersion} \\` : "";
  return (
    <CodeBlock language="sh">
      {`helm repo add cnpg https://cloudnative-pg.github.io/charts --force-update
helm upgrade --install barman-cloud \\
  --namespace cnpg-system \\
  --create-namespace \\${versionArg}
  cnpg/plugin-barman-cloud`}
    </CodeBlock>
  );
}
