window.addEventListener('load', (event) => {
  console.log('page is fully loaded');
  main()
});

const selectedThemeKey = 'selected_theme';

function main() {

  try {
    var themeJson = JSON.parse(getCookie(selectedThemeKey));
    applyTheme(themeJson.name, themeJson.link);
  } catch (e) {
    console.log("Could not parse selected-theme JSON from cookie.")
  }

  // Init packery.
  var elem = document.querySelector('.grid');
  var pckry = new Packery(elem, {
    itemSelector: '.grid-item',
    gutter: 0
  });

  try {
    // Shuffle – On click: shuffle all elements.
    const shuffleButton = document.getElementById('shuffle-button');
    shuffleButton.addEventListener('click', event => {
      pckry.shuffle();
    });
  } catch (e) {
    console.log("Shuffle button doesn't exist, this is fine.")
  }

  // Image Popup – Find and cache.
  const imagePopupElem = document.getElementById('img-popup');

  const imagePopupModal = new bootstrap.Modal(imagePopupElem, {
    keyboard: false
  })

  // Grid Images – Add click events to all grid images.
  const gridImages = document.getElementsByClassName('grid-image');
  for (const gridImage of gridImages) {
    gridImage.addEventListener('click', event => {
      imagePopupModal.show();

      // Set pin url.
      const imagePopupPinElem = document.getElementById('img-popup-pin-name');
      imagePopupPinElem.setAttribute("href", gridImage.getAttribute('pin-url'));

      // Set board name.
      const imagePopupBoardElem = document.getElementById('img-popup-board-name');
      imagePopupBoardElem.innerText = gridImage.getAttribute('board-name');

      // Set image.
      const imagePopupSource = document.getElementById('img-popup-src');
      imagePopupSource.setAttribute("src", gridImage.getAttribute('src'));
    });
  }

  // Themes – Add click events to all theme buttons.
  const themeButtons = document.getElementsByClassName('theme-button');
  for (const themeButton of themeButtons) {
    themeButton.addEventListener('click', event => {
      const theme = themeButton.getAttribute('data-theme');
      saveTheme(themeButton.innerHTML, theme);
      applyTheme(themeButton.innerHTML, theme);
    });
  }

  // Logout – On click: logout.
  const logoutButton = document.getElementById('log-out-button');
  if (logoutButton != null) {
    logoutButton.addEventListener('click', event => {
      eraseCookie("access_token");
      location.reload();
    });
  }

  // Intro – show when user is not authenticated.
  const introModalElem = document.getElementById('intro-modal');
  if (introModalElem != null) {
    const introModal = new bootstrap.Modal(introModalElem, {
      keyboard: false
    })
    introModal.show();
  }

  var taskId = getCookie("task");
  if (taskId.length > 0) {
    var baseUrl = window.location.protocol + "//" + window.location.host;
    var taskUrl = baseUrl + "/task/" + taskId
    console.log(taskUrl)

    // Start polling the server for task data
    let checkTaskStatus = function () {

      var xhr = new XMLHttpRequest();
      xhr.open("GET", taskUrl, true);
      xhr.onreadystatechange = function () {
        if (xhr.readyState === XMLHttpRequest.DONE && xhr.status === 200) {
          console.log(xhr.responseText);

          if (xhr.responseText == "complete") {
            clearInterval(taskInterval);
            location.reload();
          } else if (xhr.responseText == "processing") {
            // ...
          } if (xhr.responseText == "error") {
            clearInterval(taskInterval);
            eraseCookie("task");
            window.location.replace(baseUrl);
          }
        }
      };
      xhr.send();
    }

    let taskInterval = setInterval(checkTaskStatus, 2500);
  }
}

function saveTheme(name, link) {
  setCookie(selectedThemeKey, JSON.stringify({ name: name, link: link }), 365)
}

function applyTheme(name, link) {
  if (name != "" && link != "") {
    document.getElementById('theme').setAttribute('href', link);

    const themeDropdown = document.getElementById('theme-dropdown');
    themeDropdown.innerHTML = name;
  }
}