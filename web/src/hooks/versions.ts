import {useActiveVersion, useLatestVersion, useVersions} from '@docusaurus/plugin-content-docs/client';


export function useCurrentVersion(fallback: 'latest' | 'latestReleased' = 'latest'): string {
    switch (fallback) {
        case 'latestReleased':
            return useLatestReleasedVersion();
        case 'latest': {
            const version = useActiveVersion('default');
            return version?.name ?? useLatestVersion('default')?.name;
        }
        default:
            // The following line ensures that if `fallback` is not 'latest' or 'latestReleased',
            // an error is thrown. This can be useful for catching unexpected states.
            throw new Error(`Unhandled fallback type: ${fallback}`);
    }
}

export function useLatestReleasedVersion(): string {
    const allVersions = useVersions('default');

    // Filter out "current" to only consider versioned docs
    const versionedDocs = allVersions.filter(version => version.name !== 'current');

    // Handle the case where no versioned documents are found
    if (versionedDocs.length === 0) {
        return "unknown_version";
    }

    const sortedVersions = versionedDocs.sort((a, b) => {
        return b.name.localeCompare(a.name, undefined, { numeric: true, sensitivity: 'base' });
    });

    // The latest version is the first in the sorted list since versionedDocs was not empty,
    return sortedVersions[0].name;
}
