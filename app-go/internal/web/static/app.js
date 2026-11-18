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

// A form marked data-submit-once disables its submit buttons once it is
// sent, so a double click does not submit it twice (e.g. two sessions).
// The browser skips the submit event while required fields are invalid, so
// the buttons stay usable until the form can actually be sent.
(function () {
    document.addEventListener('submit', function (e) {
        const form = e.target;
        if (!(form instanceof HTMLFormElement) || !form.hasAttribute('data-submit-once')) return;
        if (form.dataset.submitted) {
            e.preventDefault();
            return;
        }
        form.dataset.submitted = 'true';
        form.querySelectorAll('button[type="submit"]:enabled, input[type="submit"]:enabled').forEach(function (b) {
            b.disabled = true;
            b.dataset.disabledOnSubmit = 'true';
        });
    });

    // Going back to the page can restore it from the back/forward cache with
    // the buttons still disabled. Only the ones disabled here are restored;
    // the server disables Create when there are no tests.
    window.addEventListener('pageshow', function (e) {
        if (!e.persisted) return;
        document.querySelectorAll('form[data-submit-once]').forEach(function (form) {
            delete form.dataset.submitted;
            form.querySelectorAll('[data-disabled-on-submit]').forEach(function (b) {
                b.disabled = false;
                delete b.dataset.disabledOnSubmit;
            });
        });
    });
})();
