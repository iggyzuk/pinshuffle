const selectedThemeKey = 'selected_theme';
const settingsKey = 'settings';

const root = document.documentElement;

const shuffleDuration = '0.4s';
const taskPollInterval = 1000;
const largeWidth = 750;
const maxSectionRequests = 4;

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
  showSections: false,
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
  initBlockedImagesWarning(grid);
  initSections();
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
  const onImageSettled = event => {
    if (settings.large && event.target.matches('.grid-image')) scheduleLayout();
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
    // Nothing is loaded yet, so what's needed is picking something to shuffle.
    if (document.querySelector('.grid-item') == null) {
      bootstrap.Modal.getOrCreateInstance(document.getElementById('boards-modal')).show();
      return;
    }

    if (packery == null) return;

    // Fisher-Yates.
    const items = packery.items;
    for (let i = items.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [items[i], items[j]] = [items[j], items[i]];
    }

    rollDie();

    // Only this should animate, resizing should be instant.
    packery.options.transitionDuration = shuffleDuration;
    packery.layout();
    packery.options.transitionDuration = 0;
  });
}

// Every roll lands on a different face than the last one.
function rollDie() {
  const die = document.getElementById('shuffle-die');
  const faces = [1, 2, 3, 4, 5, 6].filter(face => String(face) !== die.dataset.face);

  die.dataset.face = faces[Math.floor(Math.random() * faces.length)];
}

function initImagePopup(grid) {
  const popupElem = document.getElementById('img-popup');
  const popup = new bootstrap.Modal(popupElem, { keyboard: false });

  grid.addEventListener('click', event => {
    const item = event.target.closest('.grid-item');
    if (item == null) return;

    const image = item.querySelector('.grid-image');

    document.getElementById('img-popup-pin-name').href = image.dataset.pinUrl;

    const boardName = document.getElementById('img-popup-board-name');
    boardName.textContent = image.dataset.boardName;
    flatEmoji(boardName);

    const popupImage = document.getElementById('img-popup-src');
    popupImage.src = image.src;

    // The grid might have small images. Show that one right away, and swap it
    // for the sharpest version when that's ready.
    const fullSrc = image.dataset.fullSrc;
    if (fullSrc && fullSrc !== image.src) {
      const full = new Image();
      full.onload = () => {
        // Unless another pin got opened in the meantime.
        if (popupImage.src === image.src) popupImage.src = fullSrc;
      };
      full.src = fullSrc;
    }

    popup.show();
  });
}

// Content blockers can silently stop the images from Pinterest, which only looks
// like a broken page.
function initBlockedImagesWarning(grid) {
  const failuresBeforeWarning = 3;
  let failures = 0;
  let loaded = false;

  // Only pins count: emojis in their badges come from a different place.
  grid.addEventListener('load', event => {
    if (!event.target.matches('.grid-image')) return;

    loaded = true;
    document.getElementById('images-blocked').hidden = true;
  }, true);

  grid.addEventListener('error', event => {
    if (!event.target.matches('.grid-image')) return;

    failures++;
    if (!loaded && failures >= failuresBeforeWarning) {
      document.getElementById('images-blocked').hidden = false;
    }
  }, true);
}

// A board is either selected as a whole, or by some of its sections. Listing the
// sections takes a request for every board, so that only happens on demand.
function initSections() {
  const boardsModal = document.getElementById('boards-modal');
  const showSections = document.getElementById('show-sections');

  // Sections in the URL have to be visible, or nobody could unselect them.
  const hasSelection = boardsModal.querySelector('.section-check:checked') != null;

  showSections.checked = settings.showSections || hasSelection;
  applyShowSections(showSections.checked);

  showSections.addEventListener('change', () => {
    settings.showSections = showSections.checked;
    saveSettings();
    applyShowSections(showSections.checked);

    if (showSections.checked) {
      loadAllSections();
    } else {
      // What's selected should be what can be seen.
      boardsModal.querySelectorAll('.section-check').forEach(check => check.checked = false);
      setSectionsStatus('');
    }
  });

  boardsModal.addEventListener('show.bs.modal', () => {
    if (showSections.checked) loadAllSections();
  });

  boardsModal.addEventListener('change', event => {
    if (!event.target.checked) return;

    if (event.target.matches('.board-check')) {
      const sections = event.target.parentElement.nextElementSibling;
      sections.querySelectorAll('.section-check').forEach(check => check.checked = false);
    }

    if (event.target.matches('.section-check')) {
      const board = event.target.closest('.board-sections').previousElementSibling;
      board.querySelector('.board-check').checked = false;
    }
  });
}

function applyShowSections(show) {
  document.getElementById('boards-list').classList.toggle('show-sections', show);
}

function setSectionsStatus(text) {
  document.getElementById('sections-status').textContent = text;
}

let sectionsLoading = false;
let sectionsLoaded = false;

async function loadAllSections() {
  if (sectionsLoading || sectionsLoaded) return;
  sectionsLoading = true;

  const all = [...document.querySelectorAll('.board-sections')];
  const queue = all.filter(boardSections => boardSections.querySelector('.section-check') == null);
  const total = queue.length;
  let done = 0;
  let failed = 0;

  const worker = async () => {
    while (queue.length > 0) {
      setSectionsStatus(`Loading ${done + 1} of ${total}...`);
      try {
        await loadSections(queue.shift());
      } catch (e) {
        failed++;
      }
      done++;
    }
  };

  await Promise.all(Array.from({ length: maxSectionRequests }, worker));

  sectionsLoading = false;
  sectionsLoaded = failed === 0;

  if (failed > 0) {
    setSectionsStatus(`Couldn't load ${failed} of ${total}, turn it off and on to retry.`);
  } else if (document.querySelector('.section-check') == null) {
    setSectionsStatus('None of your boards have sections.');
  } else {
    setSectionsStatus('');
  }
}

async function loadSections(boardSections) {
  const { boardId, boardName } = boardSections.dataset;

  const response = await fetch('/boards/' + encodeURIComponent(boardId) + '/sections');
  if (!response.ok) throw new Error(response.statusText);

  const template = document.getElementById('section-template');

  for (const section of await response.json()) {
    const chip = template.content.cloneNode(true);
    const id = 'section-' + boardId + '-' + section.id;

    const input = chip.querySelector('input');
    input.id = id;
    input.value = boardId + ':' + section.id;

    const label = chip.querySelector('label');
    label.htmlFor = id;
    label.textContent = boardName + ': ' + section.name;

    boardSections.appendChild(chip);
  }

  flatEmoji(boardSections);
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

function showTaskProgress(task) {
  const bar = document.getElementById('task-bar');
  bar.style.width = task.percent + '%';
  bar.setAttribute('aria-valuenow', task.percent);

  let status = `${task.done.toLocaleString()} of ${task.total.toLocaleString()} ${task.unit}`;
  status += ` · ${task.pins.toLocaleString()}`;
  if (task.expected > 0) status += ` of ${task.expected.toLocaleString()}`;
  status += ' pins';

  document.getElementById('task-status').textContent =
    task.status === 'complete' ? 'Shuffling...' : status;
}

// Ask the server about the task until it's done, then reload to show its pins.
async function pollTask(taskId) {
  if (!taskId) return;

  try {
    const response = await fetch('/task/' + encodeURIComponent(taskId), { cache: 'no-store' });

    if (response.ok) {
      const task = await response.json();

      if (task.status === 'error') {
        location.replace('/');
        return;
      }

      showTaskProgress(task);

      if (task.status === 'complete') {
        location.reload();
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
