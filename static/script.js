// cookies
function getCookie(cname) {
  var name = cname + "=";
  var ca = document.cookie.split(';');
  for(var i = 0; i < ca.length; i++) {
    var c = ca[i];
    while (c.charAt(0) == ' ') {
      c = c.substring(1);
    }
    if (c.indexOf(name) == 0) {
      return c.substring(name.length, c.length);
    }
  }
  return "";
}

function setCookie(cname, cvalue, exdays) {
  var d = new Date();
  d.setTime(d.getTime() + (exdays * 24 * 60 * 60 * 1000));
  var expires = "expires="+d.toUTCString();
  document.cookie = cname + "=" + cvalue + ";" + expires + ";path=/";
}

function applyThemeFromCookie() {
  var theme = getCookie("selected_theme");
  if (theme != "") {
    $("head link#theme").attr("href", theme);
  }
}

// shuffle all elements
Packery.prototype.shuffle = function(){
    var m = this.items.length, t, i;
    while (m) {
        i = Math.floor(Math.random() * m--);
        t = this.items[m];
        this.items[m] = this.items[i];
        this.items[i] = t;
    }
    this.layout();
}

// // init
// $(function () {

//   applyThemeFromCookie();

//   // init packery
//   var pckry = $('.grid').packery({
//     itemSelector: '.grid-item',
//   });

//   // on click -> zoom element
//   pckry.on( 'click', '.grid-image', function( event ) {
//     $(event.currentTarget.parentNode).toggleClass('grid-item--large');
//     pckry.packery('layout');
//   });

//   // on click -> shuffle all elements
//   document.querySelector('#shuffle-button').onclick = function() {
//     pckry.packery('shuffle');
//   };
// });
