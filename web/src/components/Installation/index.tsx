import { react } from 'react';
import CodeBlock from '@theme/CodeBlock';
import { useCurrentVersion } from '@site/src/hooks/versions';

// InstallationSnippet is the kubectl incantation to install the lastest
// available version of the Barman Cloud Plugin.
export function InstallationSnippet() {
    const latest = useCurrentVersion('latestReleased');
    return(
      <CodeBlock language="sh" >
              {`kubectl apply -f \\
        https://github.com/cloudnative-pg/plugin-barman-cloud/releases/download/v${latest}/manifest.yaml`}
      </CodeBlock>
    );
}

