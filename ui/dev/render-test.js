// Tests for the rendering logic of ui/extension-monorepo-controller.js.
//
// The extension is a plain script with no module system, so it is loaded here as
// a function of its single "window" argument and given a stub window exposing a
// stub React and a stub extensionsAPI. No dependencies, run it with node:
//
//     node ui/dev/render-test.js      (or: make test-ui-local)
//
// Multi source applications are the main reason this exists: they are awkward to
// set up in a cluster, and the annotation to source index alignment is the part
// most likely to break.

'use strict';

const assert = require('assert');
const fs = require('fs');
const path = require('path');

// The built bundle, not the source: the tests then cover the real artifact,
// including the bundled git-url-parse. Build it with "make build-ui-local".
const EXTENSION_PATH = path.join(__dirname, '..', 'dist', 'extension-monorepo-controller.js');

const CHANGE_REVISION_ANN = 'mrp-controller.argoproj.io/change-revision';
const CHANGE_REVISIONS_ANN = 'mrp-controller.argoproj.io/change-revisions';

// Real values taken from applications tracked by the controller.
const SHA_1 = '2563efeae5ed486459df5cf72bb6363a228165b9';
const SHA_2 = '075c2df2dcd6d04c27a7c97fc76fbfe7ef037cfb';
const SHA_3 = 'fb119d0c59f9f36f113f9cd1f7c42e424a47b6f5';

const EM_DASH = '—';

// The bounding box the stub reports for a hovered tooltip target. Overridable so
// that the viewport dependent placement can be exercised.
const DEFAULT_RECT = {left: 100, top: 200, bottom: 216, right: 140, width: 40, height: 16};
let fakeRect = DEFAULT_RECT;
// Stands in for the .sliding-panel element Argo CD renders the flyout inside.
const panelClasses = new Set();
const FAKE_PANEL = {
    classList: {
        add: (name) => panelClasses.add(name),
        remove: (name) => panelClasses.delete(name),
        contains: (name) => panelClasses.has(name)
    }
};
const FAKE_NODE = {
    getBoundingClientRect: () => fakeRect,
    closest: (selector) => (selector === '.sliding-panel' ? FAKE_PANEL : null)
};

// Timers the code deferred, so a test can decide when they fire.
const timers = [];
const runTimers = () => timers.splice(0).forEach((fn) => fn());
const withRect = (rect, fn) => {
    fakeRect = Object.assign({}, DEFAULT_RECT, rect);
    try {
        return fn();
    } finally {
        fakeRect = DEFAULT_RECT;
    }
};

// Hook state, indexed by call order. Kept across a re-render so that a state
// update made by an event handler is visible on the next render, which is what
// makes the tooltips testable.
const hooks = {state: [], index: 0};
const resetHookIndex = () => {
    hooks.index = 0;
};
const clearHooks = () => {
    hooks.state = [];
    hooks.index = 0;
};

// Just enough of React to render the extension: function components are invoked,
// everything else becomes a plain {type, props, children} node.
const stubReact = {
    Fragment: 'React.Fragment',
    createElement: (type, props, ...children) => {
        const resolved = props || {};
        if (typeof type === 'function') {
            return type(Object.assign({}, resolved, {children: children.length === 1 ? children[0] : children}));
        }
        return {type, props: resolved, children};
    },
    cloneElement: (element, props) => ({
        type: element.type,
        props: Object.assign({}, element.props, props),
        children: element.children
    }),
    useState: (initial) => {
        const index = hooks.index++;
        if (!(index in hooks.state)) {
            hooks.state[index] = initial;
        }
        return [hooks.state[index], (value) => {
            hooks.state[index] = typeof value === 'function' ? value(hooks.state[index]) : value;
        }];
    },
    // the real React assigns the DOM node, the stub hands out a fake one so that
    // the tooltip can measure its target
    useRef: () => {
        const index = hooks.index++;
        if (!(index in hooks.state)) {
            hooks.state[index] = {current: FAKE_NODE};
        }
        return hooks.state[index];
    },
    // recorded rather than run, so a test can decide when they fire
    useEffect: (fn) => {
        effects.push(fn);
    }
};

// Effects queued by the last render, and the cleanups they returned.
const effects = [];
const cleanups = [];
const clearEffects = () => {
    effects.length = 0;
    cleanups.length = 0;
};
const runEffects = () => {
    effects.splice(0).forEach((fn) => {
        const cleanup = fn();
        if (typeof cleanup === 'function') {
            cleanups.push(cleanup);
        }
    });
};
const runCleanups = () => cleanups.splice(0).forEach((fn) => fn());

// Lets queued promise callbacks run.
const flush = async () => {
    for (let i = 0; i < 20; i++) {
        await Promise.resolve();
    }
    await new Promise((resolve) => setImmediate(resolve));
};

// The fetch stub and the calls it recorded.
const fetchCalls = [];
let fetchHandler = () => Promise.reject(new Error('no fetch handler installed'));
const jsonResponse = (body, status) => Promise.resolve({
    ok: status === undefined || (status >= 200 && status < 300),
    status: status === undefined ? 200 : status,
    json: () => Promise.resolve(body)
});

let baseURI = 'https://argocd.example.com/';

// Loads the extension with a stub window and returns the registration it made.
function loadExtension() {
    const source = fs.readFileSync(EXTENSION_PATH, 'utf8');
    const registrations = [];
    const errors = [];
    const stubWindow = {
        React: stubReact,
        ReactDOM: {
            createPortal: (node, container) => ({type: 'portal', props: {container}, children: [node]})
        },
        document: {body: {tagName: 'BODY'}, get baseURI() {
            return baseURI;
        }},
        URL: URL,
        innerHeight: 900,
        // isValidURL falls back to resolving relative URLs against the origin
        location: {origin: 'https://argocd.example.com'},
        fetch: (url, options) => {
            fetchCalls.push({url, options});
            return fetchHandler(url, options);
        },
        setTimeout: (fn) => {
            timers.push(fn);
            return timers.length;
        },
        addEventListener: () => undefined,
        removeEventListener: () => undefined,
        extensionsAPI: {
            registerStatusPanelExtension: (component, title, id, flyout) => {
                registrations.push({component, title, id, flyout});
            }
        },
        console: {error: (message) => errors.push(message)}
    };
    // eslint-disable-next-line no-new-func
    new Function('window', 'console', source)(stubWindow, stubWindow.console);
    assert.strictEqual(errors.length, 0, 'extension logged errors: ' + errors.join(', '));
    assert.strictEqual(registrations.length, 1, 'expected exactly one registration');
    return registrations[0];
}

// Collects the rendered text nodes, in order. Whitespace only nodes are layout,
// not content, so they are skipped.
function texts(element, collected) {
    const out = collected || [];
    if (element === null || element === undefined || element === false) {
        return out;
    }
    if (Array.isArray(element)) {
        element.forEach((child) => texts(child, out));
        return out;
    }
    if (typeof element === 'string' || typeof element === 'number') {
        const text = String(element);
        if (text.trim() !== '') {
            out.push(text);
        }
        return out;
    }
    texts(element.children, out);
    return out;
}

// Returns every element in the tree matching a predicate.
function findAll(element, predicate, collected) {
    const out = collected || [];
    if (!element || typeof element !== 'object') {
        return out;
    }
    if (Array.isArray(element)) {
        element.forEach((child) => findAll(child, predicate, out));
        return out;
    }
    if (predicate(element)) {
        out.push(element);
    }
    findAll(element.children, predicate, out);
    return out;
}

// Collects the title attributes of the rendered elements, in order.
function titles(element, collected) {
    const out = collected || [];
    if (!element || typeof element !== 'object') {
        return out;
    }
    if (Array.isArray(element)) {
        element.forEach((child) => titles(child, out));
        return out;
    }
    if (element.props && element.props.title !== undefined) {
        out.push(element.props.title);
    }
    titles(element.children, out);
    return out;
}

const registration = loadExtension();

const render = (application) => {
    clearHooks();
    return registration.component({application, openFlyout: () => undefined});
};

// Renders, hovers every tooltip target, then renders again so the popups appear.
const renderHovered = (application) => {
    const props = {application, openFlyout: () => undefined};
    clearHooks();
    const first = registration.component(props);
    findAll(first, (el) => typeof el.props.onMouseEnter === 'function').forEach((el) => el.props.onMouseEnter());
    resetHookIndex();
    return registration.component(props);
};

// The tooltip popups in a rendered tree, as their content elements.
const popups = (tree) => findAll(tree, (el) => el.props.className === 'tippy-tooltip light-theme');

// Renders the flyout, runs its effect, waits for the requests, then renders again.
const renderFlyout = async (application, handler) => {
    clearHooks();
    clearEffects();
    fetchCalls.length = 0;
    fetchHandler = handler || (() => Promise.reject(new Error('unexpected request')));

    registration.flyout({application, tree: {}});
    runEffects();
    await flush();

    resetHookIndex();
    clearEffects();
    const tree = registration.flyout({application, tree: {}});
    clearEffects();
    return tree;
};

// The label of each detail row, paired with its rendered value.
const detailRows = (tree) =>
    findAll(tree, (el) => el.props.className === 'row white-box__details-row').map((row) => ({
        label: texts(row.children[0]).join(''),
        value: texts(row.children[1]).join('')
    }));

const app = (spec, annotations) => ({metadata: {name: 'test-app', namespace: 'argocd', annotations}, spec});
const gitSource = (repoURL, extra) => Object.assign({repoURL, path: '.', targetRevision: 'dev'}, extra || {});

const checks = [];
function check(name, fn) {
    checks.push({name, fn});
}

check('registration metadata', () => {
    assert.strictEqual(registration.title, 'Change Revision');
    assert.strictEqual(registration.id, 'monorepo_change_revision');
    assert.strictEqual(typeof registration.flyout, 'function', 'a flyout must be registered');
});

check('single spec.source shows the abbreviated change revision', () => {
    const tree = render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')}, {
        [CHANGE_REVISION_ANN]: SHA_1,
        [CHANGE_REVISIONS_ANN]: JSON.stringify([SHA_1])
    }));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISION', '2563efe']);
    // the abbreviated value alone is in the panel, nothing is truncated silently
    assert.strictEqual(titles(tree).length, 0, 'popup tooltips replace title attributes');
});

check('single entry under spec.sources renders like a single source app', () => {
    const tree = render(app({sources: [gitSource('https://github.com/dudinea/cfrepo02.git')]}, {
        [CHANGE_REVISION_ANN]: SHA_1,
        [CHANGE_REVISIONS_ANN]: JSON.stringify([SHA_1])
    }));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISION', '2563efe']);
});

check('only the singular annotation present', () => {
    const tree = render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')}, {
        [CHANGE_REVISION_ANN]: SHA_1
    }));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISION', '2563efe']);
});

check('multi source: one row per source, index aligned, chart source included', () => {
    const tree = render(app({
        sources: [
            gitSource('https://github.com/dudinea/cfrepo02.git'),
            {repoURL: 'https://charts.example.com', chart: 'my-chart', targetRevision: '1.4.2'},
            gitSource('https://github.com/dudinea/cfrepo01.git')
        ]
    }, {
        [CHANGE_REVISION_ANN]: SHA_1,
        [CHANGE_REVISIONS_ANN]: JSON.stringify([SHA_1, '1.4.2', SHA_2])
    }));
    assert.deepStrictEqual(texts(tree), [
        'CHANGE REVISIONS',
        'cfrepo02', '2563efe',
        'my-chart', '1.4.2',
        'cfrepo01', '075c2df'
    ]);
});

check('a single chart source prefixes the chart name, having no name column', () => {
    const tree = render(app({source: {repoURL: 'https://charts.example.com', chart: 'my-chart', targetRevision: '1.4.2'}},
        {[CHANGE_REVISION_ANN]: '1.4.2'}));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISION', 'my-chart:1.4.2']);
});

check('source name prefers spec.sources[].name over the repository name', () => {
    const tree = render(app({
        sources: [
            gitSource('https://github.com/dudinea/cfrepo02.git', {name: 'manifests'}),
            gitSource('https://github.com/dudinea/cfrepo01.git', {name: 'config'})
        ]
    }, {[CHANGE_REVISIONS_ANN]: JSON.stringify([SHA_1, SHA_2])}));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISIONS', 'manifests', '2563efe', 'config', '075c2df']);
});

check('scp style repository URL yields a usable source name', () => {
    const tree = render(app({
        sources: [gitSource('git@github.com:dudinea/cfrepo02.git'), gitSource('git@host:cfrepo01.git')]
    }, {[CHANGE_REVISIONS_ANN]: JSON.stringify([SHA_1, SHA_2])}));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISIONS', 'cfrepo02', '2563efe', 'cfrepo01', '075c2df']);
});

check('empty revision renders as an em dash with an explanatory tooltip', () => {
    const tree = render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')}, {
        [CHANGE_REVISION_ANN]: '',
        [CHANGE_REVISIONS_ANN]: '[""]'
    }));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISION', EM_DASH]);

    const hovered = renderHovered(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')}, {
        [CHANGE_REVISION_ANN]: '',
        [CHANGE_REVISIONS_ANN]: '[""]'
    }));
    assert.ok(texts(hovered).includes('No change revision has been calculated for this source yet'),
        'the reason should be explained on hover');
});

check('multi source with a partially calculated array', () => {
    const tree = render(app({
        sources: [gitSource('https://github.com/dudinea/repo-a.git'), gitSource('https://github.com/dudinea/repo-b.git')]
    }, {[CHANGE_REVISIONS_ANN]: JSON.stringify(['', SHA_2])}));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISIONS', 'repo-a', EM_DASH, 'repo-b', '075c2df']);
});

check('no annotations at all renders nothing', () => {
    assert.strictEqual(render(app({source: gitSource('https://github.com/argoproj/argo-cd.git')}, {})), null);
    assert.strictEqual(render(app({source: gitSource('https://github.com/argoproj/argo-cd.git')}, undefined)), null);
    assert.strictEqual(render(app({source: gitSource('https://github.com/argoproj/argo-cd.git')},
        {'argocd.argoproj.io/manifest-generate-paths': '.'})), null);
});

check('malformed plural annotation falls back to the singular one', () => {
    const tree = render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')}, {
        [CHANGE_REVISION_ANN]: SHA_1,
        [CHANGE_REVISIONS_ANN]: 'not json at all'
    }));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISION', '2563efe']);
});

check('malformed plural annotation with no singular renders nothing', () => {
    assert.strictEqual(render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISIONS_ANN]: '{"not": "an array"}'})), null);
    assert.strictEqual(render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISIONS_ANN]: '[]'})), null);
});

check('non string elements are treated as not calculated', () => {
    const tree = render(app({
        sources: [gitSource('https://github.com/dudinea/repo-a.git'), gitSource('https://github.com/dudinea/repo-b.git')]
    }, {[CHANGE_REVISIONS_ANN]: '[null,"' + SHA_2 + '"]'}));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISIONS', 'repo-a', EM_DASH, 'repo-b', '075c2df']);
});

check('more revisions than sources renders only the covered rows', () => {
    const tree = render(app({
        sources: [gitSource('https://github.com/dudinea/repo-a.git'), gitSource('https://github.com/dudinea/repo-b.git')]
    }, {[CHANGE_REVISIONS_ANN]: JSON.stringify([SHA_1, SHA_2, SHA_3])}));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISIONS', 'repo-a', '2563efe', 'repo-b', '075c2df']);
});

check('fewer revisions than sources renders only the covered rows', () => {
    const tree = render(app({
        sources: [
            gitSource('https://github.com/dudinea/repo-a.git'),
            gitSource('https://github.com/dudinea/repo-b.git'),
            gitSource('https://github.com/dudinea/repo-c.git')
        ]
    }, {[CHANGE_REVISIONS_ANN]: JSON.stringify([SHA_1, SHA_2])}));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISIONS', 'repo-a', '2563efe', 'repo-b', '075c2df']);
});

check('revisions that are not full SHAs are shown verbatim', () => {
    const branch = render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: 'dev'}));
    assert.deepStrictEqual(texts(branch), ['CHANGE REVISION', 'dev']);

    const short = render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: '2563efe'}));
    assert.deepStrictEqual(texts(short), ['CHANGE REVISION', '2563efe']);

    const tag = render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: 'v1.2.3'}));
    assert.deepStrictEqual(texts(tag), ['CHANGE REVISION', 'v1.2.3']);
});

check('the label carries a help icon, like built in items', () => {
    const tree = render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: SHA_1}));

    // same markup argo-ui's HelpIcon produces for the built in panel items
    const icons = findAll(tree, (el) => el.props.className === 'fa fa-question-circle help-tip');
    assert.strictEqual(icons.length, 1, 'expected exactly one help icon');

    // in the label, not in the value
    assert.strictEqual(findAll(tree.children[0], (el) => el.props.className === 'fa fa-question-circle help-tip').length, 1);
    assert.strictEqual(findAll(tree.children[1], (el) => el.props.className === 'fa fa-question-circle help-tip').length, 0);

    // nothing is shown until it is hovered
    assert.strictEqual(popups(tree).length, 0);
});

check('tooltips reuse tippy.js markup so they get the built in styling', () => {
    const hovered = renderHovered(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: SHA_1}));

    // one for the help icon, one for the revision
    const shown = popups(hovered);
    assert.strictEqual(shown.length, 2, 'expected the help and revision tooltips');

    shown.forEach((tooltip) => {
        // the classes Argo CD's own tooltips carry: tippy.js 5 plus its light theme
        assert.strictEqual(tooltip.props.className, 'tippy-tooltip light-theme');
        assert.strictEqual(tooltip.props['data-state'], 'visible');
        // placement itself is covered by its own checks below
        assert.strictEqual(tooltip.props['data-placement'], 'bottom');
        assert.strictEqual(findAll(tooltip, (el) => el.props.className === 'tippy-arrow').length, 1);
        assert.strictEqual(findAll(tooltip, (el) => el.props.className === 'tippy-content').length, 1);
    });

    const outer = findAll(hovered, (el) => el.props.className === 'tippy-popper');
    assert.strictEqual(outer.length, 2);
    outer.forEach((popper) => assert.strictEqual(popper.props.style.position, 'fixed'));

    // rendered into document.body, the way argo-ui's Tooltip does
    const portals = findAll(hovered, (el) => el.type === 'portal');
    assert.strictEqual(portals.length, 2);
    portals.forEach((portal) => assert.strictEqual(portal.props.container.tagName, 'BODY'));
});

check('the help tooltip explains what a change revision is', () => {
    const hovered = renderHovered(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: SHA_1}));
    const shown = texts(hovered).join(' ');
    assert.match(shown, /commit that actually changed the manifests/);
    assert.match(shown, /Monorepo Controller/);
});

check('the revision tooltip shows the full unabbreviated revision', () => {
    const hovered = renderHovered(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: SHA_1}));
    const shown = texts(hovered);
    assert.ok(shown.includes('2563efe'), 'the panel keeps showing the abbreviation');
    assert.ok(shown.includes(SHA_1), 'the tooltip shows the full revision');
});

check('tooltips open downwards, like the built in help tooltips do', () => {
    const application = app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: SHA_1});

    // the default target sits at top 200 in a 900px viewport, so below
    const tree = renderHovered(application);
    popups(tree).forEach((tooltip) => assert.strictEqual(tooltip.props['data-placement'], 'bottom'));
    const popper = findAll(tree, (el) => el.props.className === 'tippy-popper')[0];
    assert.strictEqual(popper.props['data-placement'], 'bottom');
    assert.strictEqual(popper.props.style.top, 226); // 216 + the 10px gap
    assert.strictEqual(popper.props.style.transform, 'translateX(-50%)');
    assert.strictEqual(popper.props.style.left, 120); // centred on the target
});

check('a tooltip opens upwards only when there is no room below', () => {
    const application = app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: SHA_1});

    // near the bottom of the viewport, with more room above than below
    withRect({top: 840, bottom: 856}, () => {
        const tree = renderHovered(application);
        popups(tree).forEach((tooltip) => assert.strictEqual(tooltip.props['data-placement'], 'top'));
        const popper = findAll(tree, (el) => el.props.className === 'tippy-popper')[0];
        assert.strictEqual(popper.props.style.top, 830); // 840 - the 10px gap
        assert.strictEqual(popper.props.style.transform, 'translate(-50%, -100%)');
    });

    // cramped both ways, but still more room below: stay below
    withRect({top: 40, bottom: 56}, () => {
        const tree = renderHovered(application);
        assert.strictEqual(popups(tree)[0].props['data-placement'], 'bottom');
    });
});

check('renders the item as a status panel item', () => {
    const tree = render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: SHA_1}));
    assert.strictEqual(tree.props.className, 'application-status-panel__item');
    assert.strictEqual(tree.children[1].props.className, 'application-status-panel__item-value');
});

check('an application with no source at all does not throw', () => {
    const tree = render(app({}, {[CHANGE_REVISION_ANN]: SHA_1}));
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISION', '2563efe']);
});

// ---------------------------------------------------------------------------
// commit links
// ---------------------------------------------------------------------------

// Every anchor in a rendered tree, with the attributes that matter.
const links = (tree) =>
    findAll(tree, (el) => el.type === 'a').map((el) => ({
        href: el.props.href,
        target: el.props.target,
        rel: el.props.rel,
        text: texts(el).join(''),
        icons: findAll(el, (child) => child.props.className === 'fa fa-external-link-alt').length
    }));

const panelLinks = (repoURL, revision, extra) => {
    clearHooks();
    return links(registration.component({
        application: app({source: gitSource(repoURL, extra), project: 'default'},
            {[CHANGE_REVISION_ANN]: revision}),
        openFlyout: () => undefined
    }));
};

check('the panel revision links to the commit, per host', () => {
    // values produced by the ported Argo CD helpers, verified against git-url-parse
    const expected = [
        ['https://github.com/dudinea/cfrepo02.git', 'https://github.com/dudinea/cfrepo02/commit/' + SHA_3],
        ['git@github.com:dudinea/cfrepo02.git', 'https://github.com/dudinea/cfrepo02/commit/' + SHA_3],
        ['https://gitlab.com/group/project.git', 'https://gitlab.com/group/project/-/commit/' + SHA_3],
        ['https://bitbucket.org/team/repo.git', 'https://bitbucket.org/team/repo/commits/' + SHA_3],
        ['https://bb.example.com/scm/PROJ/repo.git', 'https://bb.example.com/projects/PROJ/repos/repo/commits/' + SHA_3],
        ['https://bb.example.com/scm/~jane/repo.git', 'https://bb.example.com/users/jane/repos/repo/commits/' + SHA_3],
        ['ssh://git@bitbucket.example.com:7999/PROJ/repo.git',
            'https://bitbucket.example.com:7999/projects/PROJ/repos/repo/commits/' + SHA_3]
    ];
    expected.forEach(([repoURL, href]) => {
        const found = panelLinks(repoURL, SHA_3);
        assert.strictEqual(found.length, 1, 'expected one link for ' + repoURL);
        assert.strictEqual(found[0].href, href, 'for ' + repoURL);
    });
});

check('links carry the same attributes and icon as Argo CD uses', () => {
    const found = panelLinks('https://github.com/dudinea/cfrepo02.git', SHA_3);
    assert.strictEqual(found[0].target, '_blank');
    assert.strictEqual(found[0].rel, 'noopener noreferrer');
    assert.strictEqual(found[0].icons, 1, 'expected the fa-external-link-alt icon');
    assert.strictEqual(found[0].text, 'fb119d0', 'the link text stays abbreviated');
});

check('a revision that is not a SHA links to the tree, not a commit', () => {
    const found = panelLinks('https://github.com/dudinea/cfrepo02.git', 'dev');
    assert.strictEqual(found[0].href, 'https://github.com/dudinea/cfrepo02/tree/dev');
});

check('an unsupported host is rendered without a link', () => {
    // Argo CD itself only builds URLs for github, gitlab.com, bitbucket.org and
    // Bitbucket Server, so self hosted GitLab and Gitea get no link
    assert.deepStrictEqual(panelLinks('https://gitlab.example.com/group/project.git', SHA_3), []);
    assert.deepStrictEqual(panelLinks('https://gitea.example.com/owner/repo.git', SHA_3), []);

    // still shown, just as text
    clearHooks();
    const tree = registration.component({
        application: app({source: gitSource('https://gitea.example.com/owner/repo.git')},
            {[CHANGE_REVISION_ANN]: SHA_3}),
        openFlyout: () => undefined
    });
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISION', 'fb119d0']);
});

check('a chart version is never linked', () => {
    clearHooks();
    const tree = registration.component({
        application: app({source: {repoURL: 'https://github.com/dudinea/charts.git', chart: 'my-chart', targetRevision: '1.4.2'}},
            {[CHANGE_REVISION_ANN]: '1.4.2'}),
        openFlyout: () => undefined
    });
    assert.deepStrictEqual(links(tree), [], 'a chart version is not a commit');
    assert.deepStrictEqual(texts(tree), ['CHANGE REVISION', 'my-chart:1.4.2']);
});

check('an empty revision is never linked', () => {
    assert.deepStrictEqual(panelLinks('https://github.com/dudinea/cfrepo02.git', ''), []);
});

check('multi source links each row to its own repository', () => {
    clearHooks();
    const tree = registration.component({
        application: app({
            sources: [
                gitSource('https://github.com/dudinea/repo-a.git'),
                gitSource('https://gitlab.com/group/repo-b.git')
            ]
        }, {[CHANGE_REVISIONS_ANN]: JSON.stringify([SHA_1, SHA_2])}),
        openFlyout: () => undefined
    });
    assert.deepStrictEqual(links(tree).map((link) => link.href), [
        'https://github.com/dudinea/repo-a/commit/' + SHA_1,
        'https://gitlab.com/group/repo-b/-/commit/' + SHA_2
    ]);
});

// ---------------------------------------------------------------------------
// the flyout
// ---------------------------------------------------------------------------

const COMMIT = {
    author: 'Jane Roe <jane@example.com>',
    date: '2026-09-29T08:15:00Z',
    message: 'fix: correct the replica count\n\nrefs #42',
    tags: ['v1.2.3', 'stable'],
    signatureInfo: 'signed by jane@example.com'
};

const singleSourceApp = app({source: gitSource('https://github.com/dudinea/cfrepo02.git'), project: 'default'},
    {[CHANGE_REVISION_ANN]: SHA_1, [CHANGE_REVISIONS_ANN]: JSON.stringify([SHA_1])});

check('the panel item has an ellipsis button wired to openFlyout', () => {
    let opened = 0;
    clearHooks();
    const tree = registration.component({application: singleSourceApp, openFlyout: () => {
        opened++;
    }});

    const buttons = findAll(tree, (el) => el.props.className === 'argo-button application-status-panel__more-button');
    assert.strictEqual(buttons.length, 1, 'expected one ellipsis button');
    assert.strictEqual(findAll(buttons[0], (el) => el.props.className === 'fa fa-ellipsis-h').length, 1);

    buttons[0].props.onClick();
    assert.strictEqual(opened, 1, 'clicking must open the flyout');
});

check('the panel item survives an absent openFlyout', () => {
    clearHooks();
    const tree = registration.component({application: singleSourceApp});
    const button = findAll(tree, (el) => el.props.className === 'argo-button application-status-panel__more-button')[0];
    assert.doesNotThrow(() => button.props.onClick());
});

check('the flyout requests commit metadata for the change revision', async () => {
    await renderFlyout(singleSourceApp, () => jsonResponse(COMMIT));

    assert.strictEqual(fetchCalls.length, 1, 'expected exactly one request');
    assert.strictEqual(
        fetchCalls[0].url,
        'https://argocd.example.com/api/v1/applications/test-app/revisions/' + SHA_1 +
            '/metadata?appNamespace=argocd&project=default&sourceIndex=0'
    );
    // the session is a cookie, and versionId is left off so the live spec is used
    assert.strictEqual(fetchCalls[0].options.credentials, 'same-origin');
    assert.ok(!fetchCalls[0].url.includes('versionId'));
});

check('the flyout honours the page base href, for sub path installations', async () => {
    baseURI = 'https://example.com/argo-cd/';
    try {
        await renderFlyout(singleSourceApp, () => jsonResponse(COMMIT));
        assert.ok(fetchCalls[0].url.startsWith('https://example.com/argo-cd/api/v1/applications/'),
            'got ' + fetchCalls[0].url);
    } finally {
        baseURI = 'https://argocd.example.com/';
    }
});

check('the flyout renders the commit details', async () => {
    const rows = detailRows(await renderFlyout(singleSourceApp, () => jsonResponse(COMMIT)));
    const byLabel = {};
    rows.forEach((row) => {
        byLabel[row.label] = row.value;
    });

    assert.strictEqual(byLabel.Repository, 'https://github.com/dudinea/cfrepo02.git');
    assert.strictEqual(byLabel.Path, '.');
    assert.strictEqual(byLabel['Target revision'], 'dev');
    assert.strictEqual(byLabel['Change revision'], SHA_1, 'the full revision, not the abbreviation');
    assert.strictEqual(byLabel.Author, 'Jane Roe <jane@example.com> - signed by jane@example.com');
    assert.strictEqual(byLabel.Tags, 'v1.2.3, stable');
    assert.strictEqual(byLabel.Message, COMMIT.message, 'the message must not be truncated');
    assert.ok(byLabel.Date && byLabel.Date !== '', 'a formatted date is expected');
});

check('the flyout links the repository and the change revision', async () => {
    const tree = await renderFlyout(singleSourceApp, () => jsonResponse(COMMIT));
    const found = links(tree);

    const repo = found.filter((link) => link.text === 'https://github.com/dudinea/cfrepo02.git');
    assert.strictEqual(repo.length, 1, 'the Repository row must link to the repository');
    assert.strictEqual(repo[0].href, 'https://github.com/dudinea/cfrepo02', 'built with repoUrl');

    const revision = found.filter((link) => link.text === SHA_1);
    assert.strictEqual(revision.length, 1, 'the Change revision row must link to the commit');
    assert.strictEqual(revision[0].href, 'https://github.com/dudinea/cfrepo02/commit/' + SHA_1);
    assert.strictEqual(revision[0].icons, 1);
});

check('the flyout leaves a chart source unlinked but still links its repository', async () => {
    const application = app({
        source: {repoURL: 'https://github.com/dudinea/charts.git', chart: 'my-chart', targetRevision: '1.4.2'},
        project: 'default'
    }, {[CHANGE_REVISION_ANN]: '1.4.2'});

    const found = links(await renderFlyout(application));
    assert.strictEqual(fetchCalls.length, 0);
    assert.deepStrictEqual(found.map((link) => link.text), ['https://github.com/dudinea/charts.git']);
});

check('the flyout shows the server error inline', async () => {
    const rows = detailRows(await renderFlyout(singleSourceApp,
        () => jsonResponse({error: 'permission denied: applications, get', code: 7}, 403)));
    const commit = rows.filter((row) => row.label === 'Commit');
    assert.strictEqual(commit.length, 1);
    assert.strictEqual(commit[0].value, 'permission denied: applications, get');

    // the revision itself still comes from the annotation, so it is still shown
    assert.ok(rows.some((row) => row.label === 'Change revision' && row.value === SHA_1));
});

check('the flyout reports a failure with no parsable body', async () => {
    const rows = detailRows(await renderFlyout(singleSourceApp, () => Promise.resolve({
        ok: false,
        status: 502,
        json: () => Promise.reject(new Error('not json'))
    })));
    assert.ok(rows.some((row) => row.label === 'Commit' && row.value === 'request failed with status 502'));
});

check('the flyout asks for each source by its own index, skipping chart sources', async () => {
    const application = app({
        sources: [
            gitSource('https://github.com/dudinea/repo-a.git'),
            {repoURL: 'https://charts.example.com', chart: 'my-chart', targetRevision: '1.4.2'},
            gitSource('https://github.com/dudinea/repo-b.git')
        ],
        project: 'monorepo'
    }, {[CHANGE_REVISIONS_ANN]: JSON.stringify([SHA_1, '1.4.2', SHA_2])});

    const tree = await renderFlyout(application, (url) => jsonResponse(url.includes(SHA_1)
        ? {author: 'first'}
        : {author: 'second'}));

    // the chart source is not looked up: a chart version is not a commit
    assert.strictEqual(fetchCalls.length, 2);
    const indexes = fetchCalls.map((call) => new URL(call.url).searchParams.get('sourceIndex'));
    assert.deepStrictEqual(indexes.sort(), ['0', '2'], 'each source must use its own index');
    assert.ok(fetchCalls.some((call) => call.url.includes('/revisions/' + SHA_1 + '/metadata')));
    assert.ok(fetchCalls.some((call) => call.url.includes('/revisions/' + SHA_2 + '/metadata')));
    fetchCalls.forEach((call) => assert.ok(new URL(call.url).searchParams.get('project') === 'monorepo'));

    const rows = detailRows(tree);
    assert.ok(rows.some((row) => row.label === 'Chart' && row.value === 'my-chart'));
    assert.ok(rows.some((row) => row.label === 'Author' && row.value === 'first'));
    assert.ok(rows.some((row) => row.label === 'Author' && row.value === 'second'));
});

check('the flyout makes no request for a revision that was never calculated', async () => {
    const application = app({source: gitSource('https://github.com/dudinea/cfrepo02.git'), project: 'default'},
        {[CHANGE_REVISIONS_ANN]: '[""]'});
    const rows = detailRows(await renderFlyout(application));
    assert.strictEqual(fetchCalls.length, 0);
    assert.ok(rows.some((row) => row.label === 'Change revision' && row.value === EM_DASH));
});

check('the flyout copes with an application that has no change revision', async () => {
    clearHooks();
    clearEffects();
    const tree = registration.flyout({
        application: app({source: gitSource('https://github.com/argoproj/argo-cd.git')}, {}),
        tree: {}
    });
    clearEffects();
    assert.ok(texts(tree).join(' ').includes('no change revision'));
});

// ---------------------------------------------------------------------------
// sliding panel width
// ---------------------------------------------------------------------------

// Renders the flyout and runs its effects, leaving the cleanups pending.
const mountFlyout = (application) => {
    clearHooks();
    clearEffects();
    panelClasses.clear();
    timers.length = 0;
    fetchHandler = () => new Promise(() => undefined);
    const tree = registration.flyout({application, tree: {}});
    runEffects();
    return tree;
};

check('the flyout widens the sliding panel to the built in width', () => {
    mountFlyout(singleSourceApp);
    // Argo CD passes no isMiddle for status panel extension flyouts, so without
    // this they render at the default 90% instead of the built in 600px
    assert.ok(panelClasses.has('sliding-panel--is-middle'), 'expected the middle width class');
});

check('the class outlives the closing animation before being removed', () => {
    mountFlyout(singleSourceApp);
    runCleanups();
    assert.ok(panelClasses.has('sliding-panel--is-middle'),
        'removing it at once would make the panel jump to full width mid animation');

    runTimers();
    assert.ok(!panelClasses.has('sliding-panel--is-middle'), 'it must not be left behind for other flyouts');
});

check('a panel Argo CD already widens is left alone', () => {
    clearHooks();
    clearEffects();
    panelClasses.clear();
    timers.length = 0;
    panelClasses.add('sliding-panel--is-middle');
    fetchHandler = () => new Promise(() => undefined);

    registration.flyout({application: singleSourceApp, tree: {}});
    runEffects();
    runCleanups();
    runTimers();

    // we never claimed it, so we must not take it away
    assert.ok(panelClasses.has('sliding-panel--is-middle'));
});

(async () => {
    let passed = 0;
    for (const item of checks) {
        await item.fn();
        passed++;
        console.log('ok - ' + item.name);
    }
    console.log('\n' + passed + ' checks passed');
})().catch((error) => {
    console.error(error);
    process.exit(1);
});
