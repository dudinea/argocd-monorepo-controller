// Git browse and commit URLs, for links that behave exactly like Argo CD's own.
//
// This is a port of Argo CD's UI helpers, with the TypeScript annotations removed
// and nothing else changed. Keeping it a faithful copy is the point: the host
// quirks it encodes - Bitbucket Server's browse URLs differing from its clone
// URLs, GitLab's /-/ prefix, Bitbucket's commits vs commit - are easy to get
// subtly wrong, and any divergence would make our links disagree with the ones
// in the built in status panel items.
//
// Sources, all Apache-2.0, the same licence as this repository:
//   repoUrl, revisionUrl and their helpers
//       ui/src/app/shared/components/urls.ts
//   isSHA
//       ui/src/app/shared/components/revision.tsx
//   isValidURL
//       ui/src/app/shared/utils.ts
//
// Note that these only produce URLs for github*, gitlab.com, bitbucket.org and
// Bitbucket Server, and return null for everything else, self hosted GitLab and
// Gitea included. That is Argo CD's own behaviour and is deliberately preserved.

import gitUrlParse from 'git-url-parse';

// Accepts abbreviated and sha256: prefixed revisions, and decides whether a URL
// points at a commit or at a tree. This is NOT interchangeable with the stricter
// full SHA test the extension uses to decide whether a revision may be
// abbreviated for display - that one must only match a complete 40 character SHA.
export const isSHA = (revision) => {
    if (revision.startsWith('sha256:')) {
        const hashOnly = revision.replace('sha256:', '');
        return hashOnly.match(/^[a-f0-9]{8,69}$/) !== null;
    }
    // https://stackoverflow.com/questions/468370/a-regex-to-match-a-sha1
    return revision.match(/^[a-f0-9]{5,40}$/) !== null;
};

export function isValidURL(url) {
    try {
        const parsedUrl = new URL(url);
        return parsedUrl.protocol !== 'javascript:' && parsedUrl.protocol !== 'data:' && parsedUrl.protocol !== 'vbscript:';
    } catch {
        try {
            // Try parsing as a relative URL.
            const parsedUrl = new URL(url, window.location.origin);
            return parsedUrl.protocol !== 'javascript:' && parsedUrl.protocol !== 'data:' && parsedUrl.protocol !== 'vbscript:';
        } catch {
            return false;
        }
    }
}

// Returns true for self-hosted Bitbucket Server instances.
// git-url-parse sets source='bitbucket-server' for HTTPS clone URLs (which contain /scm/
// in their path) and for SCP-style SSH clone URLs that also include /scm/. For SSH clone
// URLs without the /scm/ prefix (e.g. ssh://git@HOST:7999/PROJECT/repo.git) it does not
// set that source, so we additionally check the resource hostname.
function isBitbucketServer(parsed) {
    return parsed.source === 'bitbucket-server' || (parsed.resource.startsWith('bitbucket') && parsed.source !== 'bitbucket.org');
}

function supportedSource(parsed) {
    return parsed.resource.startsWith('github') || parsed.source === 'bitbucket.org' || isBitbucketServer(parsed) || parsed.source === 'gitlab.com';
}

// Bitbucket Server browse URLs differ from clone URLs:
//   clone:  https://HOST/scm/~user/repo.git  or  https://HOST/scm/PROJECTKEY/repo.git
//   browse: https://HOST/users/user/repos/repo  or  https://HOST/projects/PROJECTKEY/repos/repo
function bitbucketServerBrowseUrl(parsed) {
    const port = parsed.port ? `:${parsed.port}` : '';
    const host = `${protocol(parsed.protocol)}://${parsed.resource}${port}`;
    const owner = parsed.owner;
    // Personal repos use the ~ prefix in the clone URL owner
    if (owner.startsWith('~')) {
        return `${host}/users/${owner.slice(1)}/repos/${parsed.name}`;
    }
    return `${host}/projects/${owner}/repos/${parsed.name}`;
}

function protocol(proto) {
    return proto === 'ssh' ? 'https' : proto;
}

export function repoUrl(url) {
    try {
        const parsed = gitUrlParse(url);

        if (!supportedSource(parsed)) {
            return null;
        }

        // Self-hosted Bitbucket Server has a different browse URL structure
        // from its clone URLs, so reconstruct it using the known pattern.
        if (isBitbucketServer(parsed)) {
            const browseUrl = bitbucketServerBrowseUrl(parsed);
            if (isValidURL(browseUrl)) {
                return browseUrl;
            }
            return null;
        }

        const parsedUrl = `${protocol(parsed.protocol)}://${parsed.resource}/${parsed.owner}/${parsed.name}`;
        if (!isValidURL(parsedUrl)) {
            return null;
        }
        return parsedUrl;
    } catch {
        return null;
    }
}

export function revisionUrl(url, revision, forPath) {
    let parsed;
    try {
        parsed = gitUrlParse(url);
    } catch {
        return null;
    }

    if (!supportedSource(parsed)) {
        return null;
    }

    // Bitbucket Server uses /commits/SHA for bare commit links, and /browse[/PATH]?at=REF
    // for branch and path links. The Revision component (revision.tsx) inserts the path
    // segment BEFORE the ?at= query param, so we return only the base browse URL here.
    // When revision is empty or HEAD, omit ?at= so Bitbucket Server uses its default branch.
    if (isBitbucketServer(parsed)) {
        const base = bitbucketServerBrowseUrl(parsed);
        if (isSHA(revision) && !forPath) {
            return `${base}/commits/${revision}`;
        }
        if (!revision || revision === 'HEAD') {
            return `${base}/browse`;
        }
        return `${base}/browse?at=${encodeURIComponent(revision)}`;
    }

    let urlSubPath = isSHA(revision) ? 'commit' : 'tree';

    if (parsed.source === 'bitbucket.org') {
        // The reason for the condition of 'forPath' is that when we build nested path, we need to use 'src'
        urlSubPath = isSHA(revision) && !forPath ? 'commits' : 'src';
    }

    // Gitlab changed the way urls to commit look like
    // Ref: https://docs.gitlab.com/ee/update/deprecations.html#legacy-urls-replaced-or-removed
    if (parsed.source === 'gitlab.com') {
        urlSubPath = '-/' + urlSubPath;
    }

    return `${protocol(parsed.protocol)}://${parsed.resource}/${parsed.owner}/${parsed.name}/${urlSubPath}/${revision || 'HEAD'}`;
}
