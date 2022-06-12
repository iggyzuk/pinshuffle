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

function eraseCookie(cname) {   
  document.cookie = cname + '=; Max-Age=-99999999;';  
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

// init
$(function () {

  applyThemeFromCookie();

  // init packery
  var pckry = $('.grid').packery({
    itemSelector: '.grid-item',
  });

  // on click -> shuffle all elements
  document.querySelector('#shuffle-button').onclick = function() {
    pckry.packery('shuffle');
  };

  // Show image popup.
  $(".grid-image").on('click', function(){
    $('#img-popup').modal('show');

    // Set image.
    $src = $(this).attr('src');
    $('#img-popup-src').attr('src', $src); // <img id="img-popup-src" src="#"/>

    // Set pin info.
    // $pinName = $(this).attr('pin-name');      // pin-name attribute from .grid-image
    $pinUrl = $(this).attr('pin-url');        // pin-url attribute from .grid-image
    // TODO: uncomment this when the pin.title starts working again.
    // document.getElementById("img-popup-pin-name").innerText = $pinName; // <a id="img-popup-pin-name" href="#">Title</a>
    $('#img-popup-pin-name').attr('href', $pinUrl);

    // Set board info.
    $boardName = $(this).attr('board-name');  // board-name attribute from .grid-image
    // $boardUrl = $(this).attr('board-url');    // board-name attribute from .grid-image
    document.getElementById("img-popup-board-name").innerText = $boardName; // <p id="img-popup-board-name">Board</p>
    // $('#img-popup-board-name').attr('href', $boardUrl);

  });

  // Hide image popup.
  $("#img-popup").on('click', function(){
    $('#img-popup').modal('hide');
  });
});
