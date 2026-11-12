"use strict";

let examples = [];

function element(tag, className, text) {
  const node = document.createElement(tag);
  if (className) {
    node.className = className;
  }
  if (text !== undefined) {
    node.textContent = text;
  }

  return node;
}

function renderExamples() {
  const list = document.getElementById("examples-list");

  for (const name of examples) {
    const item = element("li");
    const link = element("a", "example-link", name);
    link.href = `#examples/${encodeURIComponent(name)}`;
    item.append(link);
    list.append(item);
  }

  if (!examples.length) {
    list.append(element("li", "empty-message", "Примеры не найдены."));
  }
}

function renderExample(name) {
  const body = document.getElementById("example-body");
  body.replaceChildren();

  if (!examples.includes(name)) {
    body.append(element("p", "empty-message", "Пример не найден."));
    return;
  }

  const images = [
    ["original.jpg", "Оригинальная фотография"],
    ["lsd.png", "Линии выделенные LSD"],
    ["ransac.png", "Группы линий после RANSAC"],
    ["tsp.png", "Точки схода после SVD"]
  ];
  const gallery = element("div", "image-gallery");

  for (const [file, title] of images) {
    const figure = element("figure", "image-card");
    const image = element("img");
    image.src = `./images/${encodeURIComponent(name)}/${file}`;
    image.alt = `${title} · ${name}`;

    image.onerror = () => {
      image.remove();
      figure.append(element("p", "image-unavailable", "Изображение недоступно."));
    };

    figure.append(element("figcaption", "", title), image);
    gallery.append(figure);
  }

  body.append(gallery);
}

function showPage() {
  const match = location.hash.match(/^#examples\/(.+)$/);
  const isQuality = location.hash === "#quality";

  document.getElementById("examples").hidden = Boolean(match) || isQuality;
  document.getElementById("example").hidden = !match;
  document.getElementById("quality").hidden = !isQuality;

  for (const link of document.querySelectorAll(".topbar nav a")) {
    if (link.hash === (isQuality ? "#quality" : "#examples")) {
      link.setAttribute("aria-current", "page");
    } else {
      link.removeAttribute("aria-current");
    }
  }

  if (match) {
    renderExample(match[1]);
  }
  if (isQuality) {
    renderQualityPage();
  }

  window.scrollTo({ top: 0 });
}

window.addEventListener("hashchange", showPage);

async function start() {
  try {
    const response = await fetch("./data.json", { cache: "no-cache" });
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    examples = await response.json();
    renderExamples();

    document.getElementById("loading").hidden = true;

    document.getElementById("app").hidden = false;
    showPage();
  } catch (error) {
    document.getElementById("loading").hidden = true;
    const message = document.getElementById("error");
    message.hidden = false;
    message.textContent = `Не удалось загрузить примеры. ${error.message}`;
  }
}

start();
