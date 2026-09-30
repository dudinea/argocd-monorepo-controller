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

const EXTENSION_PATH = path.join(__dirname, '..', 'extension-monorepo-controller.js');

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
const FAKE_NODE = {getBoundingClientRect: () => fakeRect};
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
            hooks.state[index] = value;
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
    useEffect: () => undefined
};

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
        document: {body: {tagName: 'BODY'}},
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

const app = (spec, annotations) => ({metadata: {name: 'test-app', annotations}, spec});
const gitSource = (repoURL, extra) => Object.assign({repoURL, path: '.', targetRevision: 'dev'}, extra || {});

let passed = 0;
function check(name, fn) {
    fn();
    passed++;
    console.log('ok - ' + name);
}

check('registration metadata', () => {
    assert.strictEqual(registration.title, 'Change Revision');
    assert.strictEqual(registration.id, 'monorepo_change_revision');
    assert.strictEqual(registration.flyout, undefined);
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
        assert.strictEqual(tooltip.props['data-placement'], 'top');
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

check('a tooltip flips below its target when there is no room above', () => {
    const application = app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: SHA_1});

    // the default target sits at top 200, with room above it
    const above = renderHovered(application);
    assert.strictEqual(popups(above)[0].props['data-placement'], 'top');
    const abovePopper = findAll(above, (el) => el.props.className === 'tippy-popper')[0];
    assert.strictEqual(abovePopper.props.style.top, 192); // 200 - the 8px arrow gap
    assert.strictEqual(abovePopper.props.style.transform, 'translate(-50%, -100%)');
    assert.strictEqual(abovePopper.props.style.left, 120); // centred on the target

    // near the top of the viewport it has to go below instead
    withRect({top: 10, bottom: 26}, () => {
        const flipped = renderHovered(application);
        popups(flipped).forEach((tooltip) => assert.strictEqual(tooltip.props['data-placement'], 'bottom'));
        const popper = findAll(flipped, (el) => el.props.className === 'tippy-popper')[0];
        assert.strictEqual(popper.props['data-placement'], 'bottom');
        assert.strictEqual(popper.props.style.top, 34); // 26 + the 8px arrow gap
        assert.strictEqual(popper.props.style.transform, 'translateX(-50%)');
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

console.log('\n' + passed + ' checks passed');
