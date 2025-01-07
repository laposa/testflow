import { AnsiUp } from 'ansi_up';
import './bootstrap';

window.AnsiUp = AnsiUp;

document.body.addEventListener('click', function (e) {
    repositoryListExpandCollapse(e);
    generalExpandCollapse(e);
    copyValueToClipboard(e);
});

document.body.addEventListener('change', function (e) {
    selectChildElements(e);
});

function repositoryListExpandCollapse(event) {
    if (event.target.classList.contains('expand') && !event.target.classList.contains('collapse')) {
        event.target.closest('.list').classList.add('expanded');
        event.target
            .closest('.list')
            .querySelectorAll(':scope > .list')
            .forEach((child) => {
                child.style.display = 'block';
            });
        event.target.classList.add('collapse');
    } else if (event.target.classList.contains('collapse')) {
        event.target.closest('.list').classList.remove('expanded');
        event.target.classList.remove('collapse');
        event.target
            .closest('.list')
            .querySelectorAll(':scope > .list')
            .forEach((child) => {
                child.style.display = 'none';
            });
    }
}

function generalExpandCollapse(event) {
    let content = event.target.nextElementSibling;
    if (event.target.classList.contains('expand-content') && !event.target.classList.contains('collapse-content')) {
        content.style.maxHeight = content.scrollHeight + "px";
        event.target.classList.add('collapse-content');
    } else if (event.target.classList.contains('collapse-content')) {
        content.style.maxHeight = null;
        event.target.classList.remove('collapse-content');
    }
}

function selectChildElements(event) {
    if (event.target.closest('.list-interactive')) {
        //group select
        if (!event.target.classList.contains('test-selector')) {
            const parent = event.target.closest('.list');
            parent.querySelectorAll('input').forEach((test) => {
                test.checked = event.target.checked;
            });
        }

        //TODO update parent(s) group select
        if (!event.target.classList.contains('repository-selector')) {
            // const parentGroup = e.target.closest('.list').parentNode;
            // const parentCheckbox = parentGroup.querySelectorAll(':scope > .title input');
            // parentGroup.querySelectorAll(':scope > .list').forEach((sibling) => {
            // });
        }
    }
}

function copyValueToClipboard(event) {
    if (event.target.classList.contains('copy-to-clipboard')) {
        const value = event.target.previousElementSibling.innerText;

        const alert = document.createElement('div');
        alert.classList.add('copied-alert');
        alert.style.top = `${event.pageY - 20}px`;
        alert.style.left = `${event.pageX - 55}px`;
        alert.innerText = 'Copied to clipboard';

        document.body.appendChild(alert);
        
        navigator.clipboard.writeText(value);
        /* sync duration with css */
        setTimeout(() => {
            alert.remove();
        }, 2000);
    }
}
