"use strict";

const qualityMethods = [
  { key: "ransac", name: "RANSAC", color: "#21834a" },
  { key: "svd", name: "SVD", color: "#2563eb" }
];
const qualityState = { data: null, errors: null, loading: false, tolerance: 5 };
const qualityNumber = new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 3 });
const qualityPercent = new Intl.NumberFormat("ru-RU", { maximumFractionDigits: 2 });

function qualityTable(containerId, headers, rows) {
  const table = element("table", "metric-table");
  const head = element("thead");
  const heading = element("tr");
  for (const text of headers) {
    const cell = element("th", "", text);
    cell.scope = "col";
    heading.append(cell);
  }
  head.append(heading);
  const body = element("tbody");
  for (const values of rows) {
    const row = element("tr");
    values.forEach((value, index) => {
      const cell = element(index === 0 ? "th" : "td", "", value);
      if (index === 0) cell.scope = "row";
      row.append(cell);
    });
    body.append(row);
  }
  table.append(head, body);
  document.getElementById(containerId).replaceChildren(table);
}

function referencePointTotal() {
  return qualityState.data.images.reduce((sum, image) => sum + image.referenceCount, 0);
}

function renderQualityTables() {
  const images = qualityState.data.images;
  const sources = [
    { name: "Датасет", counts: images.filter(image => image.referenceCount > 0).map(image => image.referenceCount) },
    ...qualityMethods.map(method => ({
      name: method.name,
      counts: images.filter(image => image[method.key].processed).map(image => image[method.key].pointCount)
    }))
  ];
  const hasExtraPoints = sources.some(source => source.counts.some(count => count > 3));
  const headers = ["Источник", "Изображений с данными, шт.", "С 0 точек, шт.", "С 1 точкой, шт.", "С 2 точками, шт.", "С 3 точками, шт."];
  if (hasExtraPoints) headers.push("С >3 точками, шт.");
  qualityTable("completeness-table", headers, sources.map(source => {
    const row = [source.name, source.counts.length, ...[0, 1, 2, 3].map(count => source.counts.filter(value => value === count).length)];
    if (hasExtraPoints) row.push(source.counts.filter(count => count > 3).length);
    return row;
  }));
  qualityTable("angular-summary", ["Метод", "Средняя ошибка, °"], qualityMethods.map(method => {
    const errors = qualityState.errors[method.key];
    return [method.name, errors.length ? qualityNumber.format(errors.reduce((sum, value) => sum + value, 0) / errors.length) : "Нет оценки"];
  }));

  const issues = [];
  for (const image of images) {
    if (image.referenceIssue) issues.push(`${image.name}, эталон: ${image.referenceIssue}`);
    for (const method of qualityMethods) {
      if (image[method.key].issue) issues.push(`${image.name}, ${method.name}: ${image[method.key].issue}`);
    }
  }
  document.getElementById("quality-issues").hidden = issues.length === 0;
  document.getElementById("quality-issues-list").replaceChildren(...issues.map(issue => element("li", "", issue)));
}

function renderSuccessTable() {
  const images = qualityState.data.images;
  const pointTotal = referencePointTotal();
  const imageTotal = images.filter(image => image.referenceCount === 3).length;
  const percent = (count, total) => total ? `${qualityPercent.format(100 * count / total)} %` : "Нет эталона";
  qualityTable("success-table", ["Метод", "Успешных точек, шт.", "Доля успешных точек, %", "Успешных изображений, шт.", "Доля успешных изображений, %"], qualityMethods.map(method => {
    const correctPoints = qualityState.errors[method.key].filter(error => error <= qualityState.tolerance).length;
    const correctImages = images.filter(image => image.referenceCount === 3 && image[method.key].maximumError !== null && image[method.key].maximumError <= qualityState.tolerance).length;
    return [method.name, `${correctPoints} из ${pointTotal}`, percent(correctPoints, pointTotal), `${correctImages} из ${imageTotal}`, percent(correctImages, imageTotal)];
  }));
}

function qualitySvg(tag, attributes = {}, text) {
  const node = document.createElementNS("http://www.w3.org/2000/svg", tag);
  for (const [name, value] of Object.entries(attributes)) node.setAttribute(name, value);
  if (text !== undefined) node.textContent = text;
  return node;
}

function drawQualityChart(containerId, options) {
  const container = document.getElementById(containerId);
  const width = container.clientWidth;
  if (!width) return;
  const height = 330;
  const margin = { left: width < 450 ? 54 : 72, right: 22, top: 24, bottom: 62 };
  const plotWidth = width - margin.left - margin.right;
  const plotHeight = height - margin.top - margin.bottom;
  const xPosition = value => margin.left + (value - options.xMin) / (options.xMax - options.xMin || 1) * plotWidth;
  const yPosition = value => margin.top + (1 - value / options.yMax) * plotHeight;
  const svg = qualitySvg("svg", { class: "metric-chart", viewBox: `0 0 ${width} ${height}`, role: "img", "aria-label": options.title, tabindex: "0" });
  svg.append(qualitySvg("title", {}, options.title));
  svg.append(qualitySvg("desc", {}, `${options.xLabel}; ${options.yLabel}. RANSAC — зелёная сплошная линия, SVD — синяя сплошная. Стрелки влево и вправо показывают численные значения.`));
  const tickCount = width < 450 ? 2 : 6;
  for (let index = 0; index <= 4; index++) {
    const value = options.yMax * index / 4;
    const y = yPosition(value);
    svg.append(qualitySvg("line", { x1: margin.left, x2: width - margin.right, y1: y, y2: y, stroke: "#e2e7ec" }));
    svg.append(qualitySvg("text", { x: margin.left - 9, y: y + 4, "text-anchor": "end", fill: "#526377" }, qualityPercent.format(value)));
  }
  const xTicks = new Set(Array.from({ length: tickCount + 1 }, (_, index) => options.integerX ? Math.round(options.xMin + (options.xMax - options.xMin) * index / tickCount) : options.xMin + (options.xMax - options.xMin) * index / tickCount));
  for (const value of xTicks) {
    const x = xPosition(value);
    svg.append(qualitySvg("line", { x1: x, x2: x, y1: margin.top, y2: height - margin.bottom, stroke: "#eef1f4" }));
    svg.append(qualitySvg("text", { x, y: height - margin.bottom + 22, "text-anchor": "middle", fill: "#526377" }, qualityNumber.format(value)));
  }
  svg.append(qualitySvg("line", { x1: margin.left, x2: width - margin.right, y1: height - margin.bottom, y2: height - margin.bottom, stroke: "#8a9aaa" }));
  svg.append(qualitySvg("line", { x1: margin.left, x2: margin.left, y1: margin.top, y2: height - margin.bottom, stroke: "#8a9aaa" }));
  svg.append(qualitySvg("text", { x: margin.left + plotWidth / 2, y: height - 14, "text-anchor": "middle", fill: "#223248" }, options.xLabel));
  svg.append(qualitySvg("text", { transform: `translate(17 ${margin.top + plotHeight / 2}) rotate(-90)`, "text-anchor": "middle", fill: "#223248" }, options.yLabel));
  for (const series of options.series) {
    let path = "";
    let open = false;
    for (const point of series.points) {
      if (point === null) {
        open = false;
        continue;
      }
      const [x, y] = point;
      path += `${open ? "L" : "M"}${xPosition(x).toFixed(3)},${yPosition(y).toFixed(3)} `;
      open = true;
    }
    svg.append(qualitySvg("path", { d: path, fill: "none", stroke: series.color, "stroke-width": "2", "stroke-linejoin": "round" }));
    if (options.markers) {
      for (const point of series.points.filter(Boolean)) {
        svg.append(qualitySvg("circle", { cx: xPosition(point[0]), cy: yPosition(point[1]), r: "2.8", fill: series.color }));
      }
    }
  }
  const cursor = qualitySvg("line", { x1: margin.left, x2: margin.left, y1: margin.top, y2: height - margin.bottom, stroke: "#8a9aaa", visibility: "hidden", "pointer-events": "none" });
  svg.append(cursor);
  const legend = element("div", "chart-legend");
  for (const method of qualityMethods) {
    const label = element("span");
    label.append(element("i", `legend-line ${method.key}`), document.createTextNode(method.name));
    legend.append(label);
  }
  const readout = element("div", "chart-readout", options.readout(options.xMin));
  readout.setAttribute("aria-live", "polite");
  let selectedX = options.xMin;
  const selectValue = value => {
    selectedX = Math.max(options.xMin, Math.min(options.xMax, options.integerX ? Math.round(value) : value));
    cursor.setAttribute("x1", xPosition(selectedX));
    cursor.setAttribute("x2", xPosition(selectedX));
    cursor.setAttribute("visibility", "visible");
    readout.textContent = options.readout(selectedX);
  };
  const selectPointer = event => {
    const bounds = svg.getBoundingClientRect();
    selectValue(options.xMin + ((event.clientX - bounds.left) * width / bounds.width - margin.left) / plotWidth * (options.xMax - options.xMin));
  };
  svg.addEventListener("pointermove", selectPointer);
  svg.addEventListener("pointerdown", selectPointer);
  svg.addEventListener("keydown", event => {
    if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
    event.preventDefault();
    selectValue(selectedX + (event.key === "ArrowRight" ? 1 : -1) * (options.integerX ? 1 : .1));
  });
  container.replaceChildren(svg, legend, readout);
}

function renderQualityCharts() {
  const images = qualityState.data.images;
  const pointTotal = referencePointTotal();
  if (pointTotal) {
    drawQualityChart("angle-chart", {
      title: "Накопленное распределение угловых ошибок RANSAC и SVD",
      xMin: 0, xMax: 90, yMax: 100,
      xLabel: "Допустимая угловая ошибка, °", yLabel: "Точки в пределах допуска, %",
      series: qualityMethods.map(method => {
        const errors = qualityState.errors[method.key];
        const points = [[0, 100 * errors.filter(error => error === 0).length / pointTotal]];
        let count = errors.filter(error => error === 0).length;
        for (const error of errors) {
          if (error === 0 || error > 90) continue;
          points.push([error, 100 * count / pointTotal], [error, 100 * ++count / pointTotal]);
        }
        points.push([90, 100 * count / pointTotal]);
        return { ...method, points };
      }),
      readout: value => `Допуск ${qualityNumber.format(value)}° · ${qualityMethods.map(method => {
        const count = qualityState.errors[method.key].filter(error => error <= value).length;
        return `${method.name}: ${count} из ${pointTotal} точек (${qualityPercent.format(100 * count / pointTotal)} %)`;
      }).join(" · ")}`
    });
  } else {
    document.getElementById("angle-chart").textContent = "График недоступен: нет эталонных точек.";
  }
  if (images.length) {
    const maxima = images.flatMap(image => qualityMethods.map(method => image[method.key].maximumError)).filter(value => value !== null);
    drawQualityChart("maximum-chart", {
      title: "Максимальная угловая ошибка на каждом изображении после RANSAC и SVD",
      xMin: 1, xMax: Math.max(1, images.length), yMax: Math.min(90, Math.max(10, Math.ceil(Math.max(0, ...maxima) / 10) * 10)),
      xLabel: "Номер изображения", yLabel: "Максимальная ошибка, °", integerX: true, markers: true,
      series: qualityMethods.map(method => ({ ...method, points: images.map((image, index) => image[method.key].maximumError === null ? null : [index + 1, image[method.key].maximumError]) })),
      readout: value => {
        const image = images[Math.round(value) - 1];
        return `№ ${Math.round(value)} · ${image.name} · ${qualityMethods.map(method => `${method.name}: ${image[method.key].maximumError === null ? "нет полной оценки" : `${qualityNumber.format(image[method.key].maximumError)}°`}`).join(" · ")}`;
      }
    });
  }
}

async function renderQualityPage() {
  if (qualityState.data) {
    requestAnimationFrame(renderQualityCharts);
    return;
  }
  if (qualityState.loading) return;
  qualityState.loading = true;
  const status = document.getElementById("quality-status");
  try {
    const response = await fetch("./quality.json", { cache: "no-cache" });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    qualityState.data = await response.json();
    if (!Array.isArray(qualityState.data.images)) throw new Error("Некорректные данные оценки");
    qualityState.errors = Object.fromEntries(qualityMethods.map(method => [
      method.key, qualityState.data.images.flatMap(image => image[method.key].errors).sort((a, b) => a - b)
    ]));
    document.getElementById("quality-content").hidden = false;
    status.hidden = true;
    renderQualityTables();
    renderSuccessTable();
    renderQualityCharts();
    document.getElementById("angle-tolerance").addEventListener("input", event => {
      const input = event.target;
      const valid = input.value.trim() !== "" && input.validity.valid && Number.isFinite(input.valueAsNumber);
      input.setAttribute("aria-invalid", String(!valid));
      document.getElementById("tolerance-error").hidden = valid;
      if (!valid) return;
      qualityState.tolerance = input.valueAsNumber;
      renderSuccessTable();
    });
    const observer = new ResizeObserver(() => {
      if (!document.getElementById("quality").hidden) renderQualityCharts();
    });
    observer.observe(document.getElementById("angle-chart"));
  } catch (error) {
    qualityState.data = null;
    document.getElementById("quality-content").hidden = true;
    status.hidden = false;
    status.textContent = `Не удалось загрузить оценку точек схода. ${error.message}`;
  } finally {
    qualityState.loading = false;
  }
}
