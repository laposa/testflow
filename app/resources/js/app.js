import './bootstrap';

document.body.addEventListener('click', function (e) {
    if (e.target.classList.contains('expand')) {
        const target = document.getElementById(e.target.getAttribute('data-target'));
        const button = e.target;

        if (target.style.display === 'table-row') {
            target.style.display = 'none';
            button.innerText = 'Expand';
        } else {
            target.style.display = 'table-row';
            button.innerText = 'Collapse';
        }
    }
});

// Select all tests in a suite
// When a test checkbox is checked, the suite checkbox should be checked if all tests in the suite are checked
document.body.addEventListener('change', function (e) {
    if (e.target.classList.contains('suite-selector')) {
        const suite = e.target.closest('tr').nextElementSibling;
        suite.querySelectorAll('input').forEach((test) => {
            test.checked = e.target.checked;
        });
    }

    if (e.target.classList.contains('test-selector')) {
        const suiteCheckbox = document.getElementById('selector-suite-' + e.target.dataset.parent);
        const allTests = document
            .getElementById('selector-suite-tests-' + e.target.dataset.parent)
            .querySelectorAll('input');
        const checkedTests = Array.from(allTests).filter((test) => test.checked);
        suiteCheckbox.checked = checkedTests.length === allTests.length;
    }
});
