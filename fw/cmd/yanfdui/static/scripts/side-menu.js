(function (window, document) {

    var layout   = document.getElementById('layout'),
        menu     = document.getElementById('menu'),
        menuLink = document.getElementById('menuLink'),
        content  = document.getElementById('main');

    function toggleClass(element, className) {
        var classes = element.className.split(/\s+/),
            length = classes.length,
            i = 0;

        for(; i < length; i++) {
          if (classes[i] === className) {
            classes.splice(i, 1);
            break;
          }
        }
        // The className is not found
        if (length === classes.length) {
            classes.push(className);
        }

        element.className = classes.join(' ');
    }

    function toggleAll(e) {
        var active = 'active';

        e.preventDefault();
        toggleClass(layout, active);
        toggleClass(menu, active);
        toggleClass(menuLink, active);
    }

    menuLink.onclick = function (e) {
        toggleAll(e);
    };

    content.onclick = function(e) {
        if (menu.className.indexOf('active') !== -1) {
            toggleAll(e);
        }
    };

}(this, this.document));

/* Exact match */
function do_menu_search() {
    var input = document.getElementById("menu_search");
    var filter = input.value.toUpperCase();
    var menu = document.getElementById("div_menu_items");
    var buttons = menu.getElementsByTagName("li");

    for (var i = 0; i < buttons.length; i++) {
        var item = buttons[i].getElementsByClassName("pure-menu-link");
        buttons[i].hidden = item[0].childNodes[0].textContent.toUpperCase().indexOf(filter) <= -1;
    }
}

function do_face_search() {
    let input_edit = document.getElementById("face_search");
    let filter = input_edit.value.toUpperCase();
    let face_table = document.getElementById("div_face_list");
    let faces = face_table.getElementsByTagName("tr");

    for (let i = 0; i < faces.length; i++) {
        let columns = faces[i].getElementsByTagName("td");
        let face_id = columns[0].childNodes[0].textContent.trim().toUpperCase()
        let uri = columns[1].childNodes[0].textContent.trim().toUpperCase()
        let local_uri = columns[2].childNodes[0].textContent.trim().toUpperCase()
        faces[i].hidden = (face_id.indexOf(filter) <= -1 &&
            uri.indexOf(filter) <= -1 &&
            local_uri.indexOf(filter) <= -1);
    }
}
