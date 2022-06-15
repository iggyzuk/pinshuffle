window.addEventListener('load', (event) => {
  console.log('page is fully loaded');
  main()
});

const selectedThemeKey = 'selected_theme';

function main () {

  var themeJson = JSON.parse(getCookie(selectedThemeKey));
  applyTheme(themeJson.name, themeJson.link)

  // Init packery.
  var elem = document.querySelector('.grid');
  var pckry = new Packery( elem, {
    itemSelector: '.grid-item',
    gutter: 0
  });

  // Shuffle – On click: shuffle all elements.
  const shuffleButton = document.getElementById('shuffle-button');
  shuffleButton.addEventListener('click', event => {
    pckry.shuffle();
  });

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
  if(logoutButton != null) {
    logoutButton.addEventListener('click', event => {
      eraseCookie("access_token");
      location.reload();
    });
  }

  // Intro – show when user is not authenticated.
  const introModalElem = document.getElementById('intro-modal');
  if(introModalElem != null) {
    const introModal = new bootstrap.Modal(introModalElem, {
      keyboard: false
    })
    introModal.show();
  }
}

function saveTheme(name, link) {
  setCookie(selectedThemeKey, JSON.stringify({name: name, link: link}), 365)
}

function applyTheme(name, link) {
  if (name != "" && link != "") {
    document.getElementById('theme').setAttribute('href', link);

    const themeDropdown = document.getElementById('theme-dropdown');
    themeDropdown.innerHTML = name;
  }
}
