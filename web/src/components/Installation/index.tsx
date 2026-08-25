import {ReactElement} from 'react';
import CodeBlock from '@theme/CodeBlock';
import {useActiveVersion} from '@docusaurus/plugin-content-docs/client';

// InstallationSnippet is the kubectl incantation to install the Barman
// Cloud Plugin: the manifest matching the doc version being viewed, or
// (on the unreleased "current" docs) the latest manifest on main.
export function InstallationSnippet(): ReactElement<null> {
    const activeVersion = useActiveVersion('default');
    const url = activeVersion && activeVersion.name !== 'current'
        ? `https://github.com/cloudnative-pg/plugin-barman-cloud/releases/download/v${activeVersion.name}/manifest.yaml`
        : 'https://raw.githubusercontent.com/cloudnative-pg/plugin-barman-cloud/refs/heads/main/manifest.yaml';
    return (
        <CodeBlock language="sh">
            {`kubectl apply -f \\
        ${url}`}
        </CodeBlock>
    );
}
