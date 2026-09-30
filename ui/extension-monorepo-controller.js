// Argo CD UI extension for the Monorepo Controller.
//
// Adds an item to the Application status panel showing the application's
// Change Revision.
//
// This file is served by argocd-server as part of /extensions.js: every file
// under /tmp/extensions whose name matches ^extension(.*)\.js$ is concatenated
// into that response, each one wrapped in its own try/catch block. Therefore:
//
//   * the file must be self contained - no imports, no module system,
//   * React is taken from the global scope, it must not be bundled,
//   * nothing may leak into the global scope,
//   * a failure here is only visible in the browser console as
//     "Extension extension-monorepo-controller.js failed to load: ...".
//
// WIP: this is a proof of concept, it renders a fixed string. Reading the
// actual value from the mrp-controller.argoproj.io/change-revision(s)
// annotations comes next.

((window) => {
    const extensionsAPI = window.extensionsAPI;
    if (!extensionsAPI || !extensionsAPI.registerStatusPanelExtension) {
        console.error('argocd-monorepo-controller: extensionsAPI is not available, extension not registered');
        return;
    }

    const React = window.React;
    if (!React) {
        console.error('argocd-monorepo-controller: React is not available, extension not registered');
        return;
    }

    const TITLE = 'Change Revision';
    const ID = 'monorepo_change_revision';

    // Argo CD styles the status panel item labels inline (see sectionLabel() in
    // application-status-panel.tsx), so there is no class to reuse for them.
    // #6d7f8b is argo-ui's ARGO_GRAY6_COLOR.
    const labelStyle = {
        display: 'flex',
        alignItems: 'flex-start',
        fontSize: '12px',
        fontWeight: 600,
        color: '#6d7f8b',
        minHeight: '18px'
    };

    // The status panel renders extensions without any wrapper markup, so the
    // component has to provide the application-status-panel__item block itself.
    const ChangeRevisionPanelItem = () =>
        React.createElement(
            'div',
            {className: 'application-status-panel__item'},
            React.createElement('label', {style: labelStyle}, 'CHANGE REVISION'),
            React.createElement('div', {className: 'application-status-panel__item-value'}, 'monorepo controller POC')
        );

    extensionsAPI.registerStatusPanelExtension(ChangeRevisionPanelItem, TITLE, ID);
})(window);
