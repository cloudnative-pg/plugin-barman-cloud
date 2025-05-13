import { useActiveVersion, useLatestVersion, useVersions } from '@docusaurus/plugin-content-docs/client';


export function useCurrentVersion(fallback: 'latest' | 'latestReleased' = 'latest') {
    const version = useActiveVersion('default');
    if (fallback === 'latestReleased') {
        return useLatestReleasedVersion();
    }
    if (fallback === 'latest') {
        return version?.name ?? useLatestVersion('default');
    }
}


export function useLatestReleasedVersion() {
    const versions = useVersions('default'); // returns all versions, including "current"

    // Filter out "current" to only consider versioned docs
    const versioned = versions.filter(v => v.name !== 'current');

    // Assuming the latest is the first in the list after sorting by semantic version
    const latestVersion = versioned.length > 0
        ? versioned.sort((a, b) => (b.name.localeCompare(a.name, undefined, {numeric: true, sensitivity: 'base'})))[0]
        : null;

    return latestVersion.name
}
