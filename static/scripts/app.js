const selectedThemeKey = 'selected_theme';
const settingsKey = 'settings';

const root = document.documentElement;

const shuffleDuration = '0.4s';
const taskPollInterval = 2500;
const largeWidth = 750;

let settings = {
  large: false,
  wide: true,
  tall: true,
  borders: true,
  hoverInfo: true,
  grayscale: false,
  contrast: false,
  invert: false,
  saturate: false,
};

// Maps checkboxes to the setting they control.
const settingInputs = {
  'settings-large': 'large',
  'settings-wide': 'wide',
  'settings-tall': 'tall',
  'settings-borders': 'borders',
  'settings-hover-info': 'hoverInfo',
  'settings-grayscale': 'grayscale',
  'settings-contrast': 'contrast',
  'settings-invert': 'invert',
  'settings-saturate': 'saturate',
};

// Null when the CDN is unreachable. Items then just float, in a working grid.
let packery = null;

function main() {
  loadCookies();

  const grid = document.querySelector('.grid');

  initPackery(grid);
  initShuffle();
  initImagePopup(grid);
  initThemes();
  initSettings();
  initIntro();
  pollTask(document.body.dataset.taskId);
}

function loadCookies() {
  try {
    const theme = JSON.parse(getCookie(selectedThemeKey));
    applyTheme(theme.name, theme.link);
  } catch (e) {
    // No theme was selected.
  }

  try {
    // Merge, because the cookie might be from before a setting was added.
    settings = { ...settings, ...JSON.parse(getCookie(settingsKey)) };
  } catch (e) {
    // Nothing was changed.
  }

  applySettings();
}

function initPackery(grid) {
  if (typeof Packery === 'undefined') return;

  packery = new Packery(grid, {
    itemSelector: '.grid-item',
    columnWidth: '.grid-sizer',
    percentPosition: true,
    gutter: 0,
    resize: false, // Handled below, it needs to change the columns first.
    transitionDuration: 0,
  });

  // Includes the scrollbar coming and going, which resizes every item.
  let gridWidth = grid.clientWidth;

  new ResizeObserver(() => {
    if (grid.clientWidth !== gridWidth) {
      gridWidth = grid.clientWidth;
      applyColumns();
      scheduleLayout();
    }
  }).observe(grid);

  // Items only size themselves after their image when there's no fixed aspect
  // ratio. Load events don't bubble, so catch them on the way down.
  const onImageSettled = () => {
    if (settings.large) scheduleLayout();
  };
  grid.addEventListener('load', onImageSettled, true);
  grid.addEventListener('error', onImageSettled, true);
}

// Many images finish loading at once, only lay out once for all of them.
let layoutTimeout = null;

function scheduleLayout() {
  clearTimeout(layoutTimeout);
  layoutTimeout = setTimeout(() => packery?.layout(), 50);
}

function initShuffle() {
  const shuffleButton = document.getElementById('shuffle-button');
  if (shuffleButton == null) return;

  shuffleButton.addEventListener('click', () => {
    if (packery == null) return;

    // Fisher-Yates.
    const items = packery.items;
    for (let i = items.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [items[i], items[j]] = [items[j], items[i]];
    }

    // Only this should animate, resizing should be instant.
    packery.options.transitionDuration = shuffleDuration;
    packery.layout();
    packery.options.transitionDuration = 0;
  });
}

function initImagePopup(grid) {
  const popupElem = document.getElementById('img-popup');
  const popup = new bootstrap.Modal(popupElem, { keyboard: false });

  grid.addEventListener('click', event => {
    const item = event.target.closest('.grid-item');
    if (item == null) return;

    const image = item.querySelector('.grid-image');

    document.getElementById('img-popup-pin-name').href = image.dataset.pinUrl;
    document.getElementById('img-popup-board-name').textContent = image.dataset.boardName;
    document.getElementById('img-popup-src').src = image.src;

    popup.show();
  });
}

function initThemes() {
  for (const themeButton of document.getElementsByClassName('theme-button')) {
    themeButton.addEventListener('click', () => {
      const name = themeButton.textContent;
      const link = themeButton.getAttribute('data-theme');
      saveTheme(name, link);
      applyTheme(name, link);
    });
  }
}

function initSettings() {
  for (const [id, key] of Object.entries(settingInputs)) {
    const input = document.getElementById(id);
    input.checked = settings[key];
    input.addEventListener('change', () => {
      settings[key] = input.checked;
      saveSettings();
      applySettings();
      scheduleLayout();
    });
  }
}

function initIntro() {
  const introModalElem = document.getElementById('intro-modal');
  if (introModalElem != null) {
    new bootstrap.Modal(introModalElem, { keyboard: false }).show();
  }
}

// Ask the server about the task until it's done, then reload to show its pins.
async function pollTask(taskId) {
  if (!taskId) return;

  try {
    const response = await fetch('/task/' + encodeURIComponent(taskId), { cache: 'no-store' });

    if (response.ok) {
      const status = await response.text();

      if (status === 'complete') {
        location.reload();
        return;
      }
      if (status === 'error') {
        location.replace('/');
        return;
      }
    }
  } catch (e) {
    // The network hiccuped, try again next time.
  }

  setTimeout(() => pollTask(taskId), taskPollInterval);
}

function saveTheme(name, link) {
  setCookie(selectedThemeKey, JSON.stringify({ name: name, link: link }), 365);
}

function applyTheme(name, link) {
  if (name && link) {
    document.getElementById('theme').setAttribute('href', link);
    document.getElementById('theme-dropdown').textContent = name;
  }
}

function saveSettings() {
  setCookie(settingsKey, JSON.stringify(settings), 365);
}

// Pick the column count that's closest to the wanted image width, and let the
// CSS stretch the columns to fill the page exactly. It goes by the window so
// that the scrollbar can't change the count.
function applyColumns() {
  const width = settings.large ? largeWidth : (settings.wide ? 150 : 75);
  root.style.setProperty('--columns', Math.max(1, Math.round(window.innerWidth / width)));
}

function applySettings() {
  applyColumns();

  if (settings.large) {
    root.style.setProperty('--cell-aspect', 'initial');
  } else {
    root.style.setProperty('--cell-aspect', (settings.wide ? 2 : 1) / (settings.tall ? 2 : 1));
  }

  root.style.setProperty('--border', settings.borders ? '2px' : '0px');
  root.style.setProperty('--border-radius-out', settings.borders ? '6px' : '0px');
  root.style.setProperty('--border-radius-in', settings.borders ? '4px' : '0px');
  root.style.setProperty('--hover-info-img', settings.hoverInfo ? 0.5 : 0.9);
  root.style.setProperty('--hover-info-pin', settings.hoverInfo ? 1 : 0);

  let filter = '';
  if (settings.grayscale) filter += 'grayscale(100%) ';
  if (settings.contrast) filter += 'contrast(1000%) brightness(1000%) ';
  if (settings.invert) filter += 'invert(100%) ';
  if (settings.saturate) filter += 'saturate(1000%) ';
  root.style.setProperty('--filter', filter.length > 0 ? filter : 'none');
}

document.addEventListener('DOMContentLoaded', main);
