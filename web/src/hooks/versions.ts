import {useActiveVersion, useLatestVersion} from '@docusaurus/plugin-content-docs/client';


export function useCurrentVersion(fallback: 'latest' | 'latestReleased' = 'latest'): string {
    switch (fallback) {
        case 'latestReleased':
            return useLatestReleasedVersion();
        case 'latest': {
            const version = useActiveVersion('default');
            if (version && version.name !== 'current') {
                return version.name;
            }
            return useLatestReleasedVersion();
        }
        default:
            // The following line ensures that if `fallback` is not 'latest' or 'latestReleased',
            // an error is thrown. This can be useful for catching unexpected states.
            throw new Error(`Unhandled fallback type: ${fallback}`);
    }
}

export function useLatestReleasedVersion(): string {
    return useLatestVersion('default').name;
}
