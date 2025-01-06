import { AnsiUp } from 'ansi_up';
import './bootstrap';

window.AnsiUp = AnsiUp;

document.body.addEventListener('click', function (e) {
    // Expand for repositories
    if (e.target.classList.contains('expand') && !e.target.classList.contains('collapse')) {
        e.target.closest('.list').classList.add('expanded');
        e.target
            .closest('.list')
            .querySelectorAll(':scope > .list')
            .forEach((child) => {
                child.style.display = 'block';
            });
        e.target.classList.add('collapse');
    } else if (e.target.classList.contains('collapse')) {
        e.target.closest('.list').classList.remove('expanded');
        e.target.classList.remove('collapse');
        e.target
            .closest('.list')
            .querySelectorAll(':scope > .list')
            .forEach((child) => {
                child.style.display = 'none';
            });
    }

    //General expand/collapse
    let content = e.target.nextElementSibling;
    if (e.target.classList.contains('expand-content') && !e.target.classList.contains('collapse-content')) {
        content.style.maxHeight = content.scrollHeight + "px";
        e.target.classList.add('collapse-content');
    } else if (e.target.classList.contains('collapse-content')) {
        content.style.maxHeight = null;
        e.target.classList.remove('collapse-content');
    }
});

// Selecting all children
document.body.addEventListener('change', function (e) {
    if (e.target.closest('.list-interactive')) {
        //group select
        if (!e.target.classList.contains('test-selector')) {
            const parent = e.target.closest('.list');
            parent.querySelectorAll('input').forEach((test) => {
                test.checked = e.target.checked;
            });
        }

        //TODO update parent(s) group select
        if (!e.target.classList.contains('repository-selector')) {
            // const parentGroup = e.target.closest('.list').parentNode;
            // const parentCheckbox = parentGroup.querySelectorAll(':scope > .title input');
            // parentGroup.querySelectorAll(':scope > .list').forEach((sibling) => {
            // });
        }
    }
});
