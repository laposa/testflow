import './bootstrap';

const expands = document.querySelectorAll('.expand');
const groupSelects = document.querySelectorAll('.checkbox.all input[type="checkbox"]');

expands.forEach((expand) => {
    expand.addEventListener('click', (e) => {
        if(e.target) {
            const target = document.getElementById(e.target.getAttribute('data-target'));
            const button = e.target;

            if (target.style.display === "table-row") {
                target.style.display = "none";
                button.innerText = "Expand";
            } else {
                target.style.display = "table-row";
                button.innerText = "Collapse";
            }
        }
    });
});
