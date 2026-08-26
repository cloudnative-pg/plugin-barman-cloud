import {ReactElement} from 'react';
import CodeBlock from '@theme/CodeBlock';
import {useActiveVersion} from '@docusaurus/plugin-content-docs/client';
import Heading from '@theme/Heading';

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

// ManifestVersion names the manifest the snippet below installs: the
// release tag for a versioned docs page, or the main branch on Dev docs.
export function ManifestVersion(): ReactElement<null> {
    const activeVersion = useActiveVersion('default');
    return activeVersion && activeVersion.name !== 'current'
        ? <>version <code>v{activeVersion.name}</code></>
        : <>the latest development snapshot from the <code>main</code> branch</>;
}


// DevSnapshotSection offers the main-branch manifest as an alternative;
// on the Dev docs the main install already is that manifest, so hide it.
export function DevSnapshotSection(): ReactElement<null> | null {
    const activeVersion = useActiveVersion('default');
    const url = 'https://raw.githubusercontent.com/cloudnative-pg/plugin-barman-cloud/refs/heads/main/manifest.yaml';
    if (!activeVersion || activeVersion.name === 'current') return null;
    return (
        <>
            <Heading as="h2" id="testing-the-latest-development-snapshot">
                Testing the latest development snapshot
            </Heading>
            <CodeBlock language="sh">
                {`kubectl apply -f \\
        ${url}`}
            </CodeBlock>
        </>
    );
}
