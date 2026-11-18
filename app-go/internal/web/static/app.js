// TestFlow client behaviour that native HTML and htmx don't cover (spec §10, D17).
// Loaded as an external file so it fits the self-only Content-Security-Policy.
// Later iterations add: opening/closing the manual run <dialog>.
'use strict';

// Test tree "select all" (legacy: resources/js/app.js selectChildElements).
// A group checkbox in a <summary> checks or clears every test below it, and
// each group checkbox shows whether all, some (indeterminate) or none of its
// tests are selected.
(function () {
    function syncGroups(tree) {
        // Deepest groups first, so parents see their children's final state.
        const groups = Array.from(tree.querySelectorAll('.list')).reverse();
        for (const list of groups) {
            const box = list.querySelector(':scope > summary .group-selector');
            if (!box) continue;
            const tests = list.querySelectorAll('.test-selector');
            const checked = list.querySelectorAll('.test-selector:checked').length;
            box.checked = tests.length > 0 && checked === tests.length;
            box.indeterminate = checked > 0 && checked < tests.length;
        }
    }

    document.addEventListener('change', function (e) {
        const box = e.target;
        if (!(box instanceof HTMLInputElement) || box.type !== 'checkbox') return;
        const tree = box.closest('.list-interactive');
        if (!tree) return;
        if (box.classList.contains('group-selector')) {
            box.closest('.list').querySelectorAll('.test-selector').forEach(function (t) {
                t.checked = box.checked;
            });
        }
        syncGroups(tree);
    });

    // The script is deferred, so the DOM is parsed: reflect a selection kept
    // after a validation error.
    document.querySelectorAll('.list-interactive').forEach(syncGroups);
})();
