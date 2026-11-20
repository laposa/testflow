// TestFlow client behaviour that native HTML and htmx don't cover (spec §10, D17).
// Loaded as an external file so it fits the self-only Content-Security-Policy.
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

// Run list button label (legacy: "Run Selected Tests (n)" or "Run All
// Tests"). The server renders the right label; this keeps it in step with
// the checkboxes. The run list is replaced by htmx, so listen on document.
(function () {
    document.addEventListener('change', function (e) {
        const box = e.target;
        if (!(box instanceof HTMLInputElement) || !box.classList.contains('run-selector') || !box.form) return;
        const button = box.form.querySelector('[data-run-button]');
        if (!button) return;
        const n = box.form.querySelectorAll('.run-selector:checked').length;
        button.textContent = n > 0 ? 'Run Selected Tests (' + n + ')' : 'Run All Tests';
    });
})();

// Test file dialog on the run page (legacy: openModal in
// sessions/run.blade.php). A test name links to the raw file; this loads it
// into the <dialog> instead of a new tab. The text goes in as textContent,
// so the file is never parsed as HTML. Escape and the focus trap come from
// the browser; a click on the backdrop closes the dialog.
(function () {
    const dialog = document.getElementById('test-dialog');
    if (!dialog) return;
    const title = dialog.querySelector('h2');
    const loading = dialog.querySelector('.loading');
    const content = dialog.querySelector('.content');
    let current = null; // the request in flight, aborted by the next one

    function show(text, failed) {
        loading.hidden = true;
        content.hidden = false;
        content.textContent = text;
        content.classList.toggle('error', failed);
    }

    document.addEventListener('click', function (e) {
        const link = e.target instanceof Element && e.target.closest('a[data-test-file]');
        // Keep the browser's own handling for a new tab or window.
        if (!link || e.button !== 0 || e.ctrlKey || e.metaKey || e.shiftKey || e.altKey) return;
        e.preventDefault();

        if (current) current.abort();
        const request = new AbortController();
        current = request;
        title.textContent = link.textContent;
        loading.hidden = false;
        content.hidden = true;
        dialog.showModal();

        fetch(link.href, { signal: request.signal, credentials: 'same-origin' })
            .then(function (res) {
                // An expired sign-in redirects to the login page: go there
                // instead of showing its HTML as the file.
                if (res.redirected) {
                    window.location.assign(res.url);
                    return;
                }
                const plain = (res.headers.get('Content-Type') || '').startsWith('text/plain');
                if (res.ok && plain) return res.text().then(function (t) { show(t, false); });
                if (res.ok) {
                    show('Could not load the file.', true);
                    return;
                }
                show(res.status === 404 ? 'The file is not in the repository at this commit.'
                    : 'Could not load the file from GitHub.', true);
            })
            .catch(function (err) {
                if (err.name !== 'AbortError') show('Could not load the file.', true);
            });
    });

    // The dialog has no padding of its own, so a click on it is a click
    // on the backdrop.
    dialog.addEventListener('click', function (e) {
        if (e.target === dialog) dialog.close();
    });
    dialog.addEventListener('close', function () {
        if (current) current.abort();
        current = null;
    });
})();

// Manual run wizard (legacy: livewire/runs/manual-test-run in a modal). A
// "Complete manual tests" link loads the wizard into the session page's
// <dialog> with htmx; this opens the dialog with the spinner meanwhile.
// When the run finishes (HX-Trigger manual-run-finished) the dialog closes,
// and closing it reloads the run list for the new counts.
(function () {
    const dialog = document.getElementById('manual-run');
    if (!dialog) return;
    const content = document.getElementById('manual-run-content');
    const spinner = content.firstElementChild && content.firstElementChild.cloneNode(true);

    document.addEventListener('click', function (e) {
        const link = e.target instanceof Element && e.target.closest('a[data-manual-run]');
        // htmx prevents the default when it loads the link; otherwise (no
        // htmx, or a new tab) the browser opens the wizard page.
        if (!link || !e.defaultPrevented) return;
        content.replaceChildren();
        if (spinner) content.append(spinner.cloneNode(true));
        dialog.showModal();
    });

    document.addEventListener('manual-run-finished', function () {
        if (dialog.open) dialog.close();
    });

    // A new test starts at the top, as legacy scrolled the modal.
    document.addEventListener('htmx:after:swap', function (e) {
        if (dialog.open && e.target instanceof Node && dialog.contains(e.target)) dialog.scrollTop = 0;
    });

    // The dialog has no padding, so a click on it is on the backdrop.
    dialog.addEventListener('click', function (e) {
        if (e.target === dialog) dialog.close();
    });
    dialog.addEventListener('close', function () {
        document.body.dispatchEvent(new CustomEvent('runs-refresh'));
    });
})();

// Manual test steps: marking every step passed picks Pass, and marking one
// failed picks Fail (legacy: list-runs.blade.php markTestAs). The step
// marks are not sent.
(function () {
    document.addEventListener('change', function (e) {
        const box = e.target;
        if (!(box instanceof HTMLInputElement) || !box.closest('.custom-radio')) return;
        const test = box.closest('.manual-test');
        if (!test) return;
        const passes = Array.from(test.querySelectorAll('.custom-radio .radio-pass'));
        const fails = Array.from(test.querySelectorAll('.custom-radio .radio-fail'));
        let pick = null;
        if (fails.some(function (f) { return f.checked; })) pick = test.querySelector('.result-fail');
        else if (passes.length > 0 && passes.every(function (p) { return p.checked; })) pick = test.querySelector('.result-pass');
        if (pick) pick.checked = true;
    });
})();

// Copy a test input's value (legacy: copyValueToClipboard). The button
// says "Copied" for two seconds through a class.
(function () {
    document.addEventListener('click', function (e) {
        const button = e.target instanceof Element && e.target.closest('.copy-to-clipboard');
        if (!button || !navigator.clipboard) return;
        navigator.clipboard.writeText(button.textContent).then(function () {
            button.classList.add('copied');
            setTimeout(function () { button.classList.remove('copied'); }, 2000);
        });
    });
})();
