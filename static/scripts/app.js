window.addEventListener('load', (event) => {
  console.log('page is fully loaded');
  main()
});

const selectedThemeKey = 'selected_theme';
const settingsKey = 'settings';

let settings = {
  wide: true,
  tall: true,
  borders: true,
  hoverInfo: true,
  grayscale: false,
  contrast: false,
  invert: false,
  saturate: false,
};

function main() {

  var r = document.querySelector(':root');

  try {
    var themeObj = JSON.parse(getCookie(selectedThemeKey));
    applyTheme(themeObj.name, themeObj.link);
  } catch (e) {
    console.log("Could not parse selected-theme JSON from cookie.")
  }

  try {
    var settingsObj = JSON.parse(getCookie(settingsKey));
    settings = settingsObj;
    applySettings(r, settings);
    console.log(settings);
  } catch (e) {
    console.log("Could not parse settings JSON from cookie.")
  }

  // Init packery.
  var elem = document.querySelector('.grid');
  var packery = new Packery(elem, {
    itemSelector: '.grid-item',
    gutter: 0
  });

  try {
    // Shuffle – On click: shuffle all elements.
    const shuffleButton = document.getElementById('shuffle-button');
    shuffleButton.addEventListener('click', event => {
      packery.shuffle();
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

  // User Settings.
  const wide = document.getElementById('settings-wide');
  wide.checked = settings.wide;
  wide.addEventListener('change', event => {
    settings.wide = wide.checked;
    saveSettings(settings);

    applySettings(r, settings);
    packery.layout();
  });

  const tall = document.getElementById('settings-tall');
  tall.checked = settings.tall;
  tall.addEventListener('change', event => {
    settings.tall = tall.checked;
    saveSettings(settings);

    applySettings(r, settings);
    packery.layout();
  });

  const borders = document.getElementById('settings-borders');
  borders.checked = settings.borders;
  borders.addEventListener('change', event => {
    settings.borders = borders.checked;
    saveSettings(settings);

    applySettings(r, settings);
  });

  const hoverInfo = document.getElementById('settings-hover-info');
  hoverInfo.checked = settings.hoverInfo;
  hoverInfo.addEventListener('change', event => {
    settings.hoverInfo = hoverInfo.checked;
    saveSettings(settings);

    applySettings(r, settings);
  });

  const grayscale = document.getElementById('settings-grayscale');
  grayscale.checked = settings.grayscale;
  grayscale.addEventListener('change', event => {
    settings.grayscale = grayscale.checked;
    saveSettings(settings);

    applySettings(r, settings);
  });

  const contrast = document.getElementById('settings-contrast');
  contrast.checked = settings.contrast;
  contrast.addEventListener('change', event => {
    settings.contrast = contrast.checked;
    saveSettings(settings);

    applySettings(r, settings);
  });

  const invert = document.getElementById('settings-invert');
  invert.checked = settings.invert;
  invert.addEventListener('change', event => {
    settings.invert = invert.checked;
    saveSettings(settings);

    applySettings(r, settings);
  });

  const saturate = document.getElementById('settings-saturate');
  saturate.checked = settings.saturate;
  saturate.addEventListener('change', event => {
    settings.saturate = saturate.checked;
    saveSettings(settings);

    applySettings(r, settings);
  });

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

function saveSettings(settings) {
  setCookie(settingsKey, JSON.stringify(settings), 365);
}

function applySettings(r, settings) {
  r.style.setProperty('--img-width', settings.wide ? '150px' : '75px');
  r.style.setProperty('--img-height', settings.tall ? '150px' : '75px');
  r.style.setProperty('--border', settings.borders ? '2px' : '0px');
  r.style.setProperty('--border-radius-out', settings.borders ? '6px' : '0px');
  r.style.setProperty('--border-radius-in', settings.borders ? '4px' : '0px');
  r.style.setProperty('--hover-info-img', settings.hoverInfo ? 0.5 : 0.9);
  r.style.setProperty('--hover-info-pin', settings.hoverInfo ? 1 : 0);

  let filter = '';
  if (settings.grayscale) filter += 'grayscale(100%) ';
  if (settings.contrast) filter += 'contrast(1000%) brightness(1000%) ';
  if (settings.invert) filter += 'invert(100%) ';
  if (settings.saturate) filter += 'saturate(1000%) ';
  r.style.setProperty('--filter', filter.length > 0 ? filter : 'none');
}