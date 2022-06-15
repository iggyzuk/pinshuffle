// window.onload = main;

window.addEventListener('load', (event) => {
  console.log('page is fully loaded');
  main()
});

const selectedThemeKey = 'selected_theme';

function main () {

  applyTheme(getCookie(selectedThemeKey))

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
      saveTheme(theme);
      applyTheme(theme);
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

function saveTheme(theme) {
  setCookie(selectedThemeKey, theme, 365)
}

function applyTheme(theme) {
  if (theme != "") {
    document.getElementById('theme').setAttribute('href', theme);
  }
}
