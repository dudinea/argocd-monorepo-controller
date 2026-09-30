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

// Loads the extension with a stub window and returns the registration it made.
function loadExtension() {
    const source = fs.readFileSync(EXTENSION_PATH, 'utf8');
    const registrations = [];
    const errors = [];
    const stubWindow = {
        React: {
            createElement: (type, props, ...children) => ({type, props: props || {}, children})
        },
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
const render = (application) => registration.component({application, openFlyout: () => undefined});

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
    // the full revision stays reachable as a tooltip, since it is not a link.
    // Scoped to the value, the label has the help icon's own tooltip.
    assert.deepStrictEqual(titles(tree.children[1]), [SHA_1]);
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
    assert.deepStrictEqual(titles(tree.children[1]), ['No change revision has been calculated for this source yet']);
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

check('the label carries a help icon with a description, like built in items', () => {
    const tree = render(app({source: gitSource('https://github.com/dudinea/cfrepo02.git')},
        {[CHANGE_REVISION_ANN]: SHA_1}));

    // same markup argo-ui's HelpIcon produces for the built in panel items
    const icons = findAll(tree, (el) => el.props.className === 'fa fa-question-circle help-tip');
    assert.strictEqual(icons.length, 1, 'expected exactly one help icon');

    const label = tree.children[0];
    const described = findAll(label, (el) => typeof el.props.title === 'string' && el.props.title.length > 0);
    assert.strictEqual(described.length, 1, 'expected the help icon to carry a title');
    assert.match(described[0].props.title, /commit that actually changed the manifests/);

    // the icon must not leak into the value, which has its own revision tooltips
    assert.strictEqual(findAll(tree.children[1], (el) => el.props.className === 'fa fa-question-circle help-tip').length, 0);
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
