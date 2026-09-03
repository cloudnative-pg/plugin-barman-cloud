import {ReactElement} from 'react';
import CodeBlock from '@theme/CodeBlock';
import {useActiveVersion, useLatestVersion} from '@docusaurus/plugin-content-docs/client';

// HelmInstallationSnippet is the Helm command to install the plugin.
//
// - Latest release: no override. The chart already defaults to
//   itself.
// - Older release: pin image.tag and sidecarImage.tag to that
//   version. We check this again on every build using
//   useLatestVersion, so an old page updates itself once a newer
//   version ships.
// - Dev docs: also no override, but this does NOT install a dev
//   build. Setting the tag alone would not be enough, since the
//   chart's own templates (CRDs, RBAC...) can be older than what
//   main needs. Use the kubectl method to test a dev build.
export function HelmInstallationSnippet(): ReactElement<null> {
    const activeVersion = useActiveVersion('default');
    const latestVersion = useLatestVersion('default');
    const isOlderRelease = activeVersion
        && activeVersion.name !== 'current'
        && activeVersion.name !== latestVersion.name;
    const setArgs = isOlderRelease
        ? ` \\
  --set image.tag=v${activeVersion.name} \\
  --set sidecarImage.tag=v${activeVersion.name}`
        : '';
    return (
        <CodeBlock language="sh">
            {`helm repo add cnpg https://cloudnative-pg.github.io/charts --force-update
helm upgrade --install plugin-barman-cloud \\
  --namespace cnpg-system${setArgs} \\
  cnpg/plugin-barman-cloud`}
        </CodeBlock>
    );
}
