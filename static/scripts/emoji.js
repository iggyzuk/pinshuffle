// Swaps emojis for the flat Twemoji images, the same on every device. If the CDN
// can't be reached the native emojis stay, which is fine.
function flatEmoji(node) {
  if (typeof twemoji !== 'undefined') {
    twemoji.parse(node, { folder: 'svg', ext: '.svg' });
  }
}

document.addEventListener('DOMContentLoaded', () => flatEmoji(document.body));
