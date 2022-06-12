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
    $('#img-popup-src').attr('src', $src);

    // Set pin info.
    $pinName = $(this).attr('pin-name');
    $pinUrl = $(this).attr('pin-url');
    $pinElement = document.getElementById("img-popup-pin-name");
    $pinElement.innerText = $pinName;
    $pinElement.attr('href', $pinUrl);

    // Set board info.
    $boardName = $(this).attr('board-name');
    $boardUrl = $(this).attr('board-url');
    $boardElement = document.getElementById("img-popup-board-name");
    $boardElement.innerText = $boardName;
    $boardElement.attr('href', $boardUrl);

  });

  // Hide image popup.
  $("#img-popup").on('click', function(){
    $('#img-popup').modal('hide');
  });
});
