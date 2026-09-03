import {ReactElement} from 'react';
import CodeBlock from '@theme/CodeBlock';
import Link from '@docusaurus/Link';
import {useLocation} from '@docusaurus/router';
import {useActiveVersion} from '@docusaurus/plugin-content-docs/client';

// DEV_MANIFEST_URL is the URL of the manifest.yaml on the main branch of the plugin repo.
const DEV_MANIFEST_URL =
    'https://raw.githubusercontent.com/cloudnative-pg/plugin-barman-cloud/refs/heads/main/manifest.yaml';

// InstallationSnippet is the kubectl incantation to install the Barman
// Cloud Plugin: the manifest matching the doc version being viewed, or
// (on the unreleased "current" docs) the latest manifest on main.
export function InstallationSnippet(): ReactElement<null> {
    const activeVersion = useActiveVersion('default');
    const url = activeVersion && activeVersion.name !== 'current'
        ? `https://github.com/cloudnative-pg/plugin-barman-cloud/releases/download/v${activeVersion.name}/manifest.yaml`
        : DEV_MANIFEST_URL;
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
        ? <code>v{activeVersion.name}</code>
        : <>the latest development snapshot from the <code>main</code> branch</>;
}

// DevSnapshotSection shows how to test the main-branch manifest. On
// the Dev docs, the kubectl install above already does that, so we
// hide this section there. (The Helm tab there installs the latest
// release, not a dev build.)
export function DevSnapshotSection(): ReactElement {
    const activeVersion = useActiveVersion('default');
    const {pathname} = useLocation();
    if (!activeVersion || activeVersion.name === 'current') {
        return (
            <p>The <Link to={`${pathname}?install-method=kubectl#installing-the-barman-cloud-plugin`}>kubectl
                install command above</Link> already applies the latest development
                snapshot from the <code>main</code> branch.</p>
        );
    }
    return (
        <>
            <p>You can also test the latest development snapshot of the plugin
                with the following command:</p>
            <CodeBlock language="sh">
                {`kubectl apply -f \\
        ${DEV_MANIFEST_URL}`}
            </CodeBlock>
        </>
    );
}
