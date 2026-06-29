import {ReactElement} from 'react';
import CodeBlock from '@theme/CodeBlock';
import {useCurrentVersion} from '@site/src/hooks/versions';
import {MultiLangCodeBlock} from '@site/src/components/MultiLangCodeBlock';

// InstallationSnippet is the kubectl incantation to install the lastest
// available version of the Barman Cloud Plugin.
export function InstallationSnippet(): ReactElement<null> {
    const latest = useCurrentVersion('latestReleased');

    const snippets = [
      {
        label: 'kubectl',
        language: 'sh',
        code: `kubectl apply -f \\
https://github.com/cloudnative-pg/plugin-barman-cloud/releases/download/v${latest}/manifest.yaml`,
      },
      {
        label: 'Helm',
        language: 'sh',
        code: `helm repo add cnpg https://cloudnative-pg.github.io/charts
helm repo update
helm upgrade \\
  --install \\
  --namespace cnpg-system \\
  plugin-barman-cloud cnpg/plugin-barman-cloud`,
      },
    ];

    return <MultiLangCodeBlock snippets={snippets} />;
}
