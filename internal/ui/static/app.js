const root = document.documentElement;
const isMobile = () => window.innerWidth <= 768;

function setSidebar(collapsed) {
  root.classList.toggle("sidebar-collapsed", collapsed);
  document.getElementById("sidebar-toggle").setAttribute("aria-expanded", String(!collapsed));
  if (!isMobile()) localStorage.setItem("sidebar", collapsed ? "collapsed" : "expanded");
}

document.getElementById("sidebar-toggle").addEventListener("click", () => {
  setSidebar(!root.classList.contains("sidebar-collapsed"));
});
document.getElementById("sidebar-backdrop").addEventListener("click", () => setSidebar(true));
document.addEventListener("keydown", (e) => {
  if (e.key === "Escape" && isMobile()) setSidebar(true);
});

// On small screens the drawer closes after navigating, without overwriting the desktop preference.
document.querySelectorAll("#sidebar .nav-link").forEach((a) => {
  a.addEventListener("click", () => {
    if (isMobile()) root.classList.add("sidebar-collapsed");
  });
});

document.getElementById("sidebar-toggle").setAttribute("aria-expanded", String(!root.classList.contains("sidebar-collapsed")));

// Routine editor: add, remove and drag-to-reorder happen client-side; Save posts the rows in DOM order.
function initRoutineLists(scope) {
  scope.querySelectorAll(".rt-list").forEach((list) => {
    if (list.dataset.sortable) return;
    list.dataset.sortable = "1";
    Sortable.create(list, {
      handle: ".rt-handle",
      animation: 150,
      // Pointer-based dragging behaves the same for mouse and touch, unlike native drag-and-drop.
      forceFallback: true,
      fallbackOnBody: true,
      fallbackClass: "rt-fallback",
      ghostClass: "rt-ghost",
      chosenClass: "rt-chosen",
    });
  });
}
document.body.addEventListener("htmx:load", (e) => initRoutineLists(e.target));
initRoutineLists(document);

document.addEventListener("click", (e) => {
  const remove = e.target.closest("[data-rt-remove]");
  if (remove) remove.closest(".rt-row").remove();

  const add = e.target.closest("[data-rt-add]");
  if (add) {
    const form = add.closest("form");
    const row = form.querySelector(".rt-template").content.cloneNode(true);
    form.querySelector(".rt-list").appendChild(row);
    const added = form.querySelector(".rt-list").lastElementChild;
    added.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }
});

// Recorder: choosing "Add New Exercise..." opens the dialog instead of switching exercise.
document.addEventListener("change", (e) => {
  const select = e.target.closest("[data-workout-select]");
  if (select && select.value === "__add__") {
    select.value = select.dataset.current;
    document.getElementById("add-exercise-dialog").showModal();
  }
});

// Historical analytics: max weight (left axis) and volume moved (right axis) over time.
function initCharts(scope) {
  scope.querySelectorAll("canvas[data-chart=history]").forEach((canvas) => {
    if (canvas.dataset.ready) return;
    canvas.dataset.ready = "1";
    const points = JSON.parse(canvas.dataset.points);
    const ts = (p) => new Date(p.date + "T00:00:00").getTime();
    const fmt = (ms, opts) => new Date(ms).toLocaleDateString("en-US", opts);
    const text = "#f0eeeb", grid = "rgba(240, 238, 235, 0.08)";

    new Chart(canvas, {
      type: "line",
      data: {
        datasets: [
          {
            label: "Max Weight",
            data: points.map((p) => ({ x: ts(p), y: p.weight })),
            borderColor: "rgba(255, 145, 164, 1)",
            backgroundColor: "rgba(255, 145, 164, 1)",
            yAxisID: "y",
          },
          {
            label: "Volume Moved",
            data: points.filter((p) => p.volume != null).map((p) => ({ x: ts(p), y: p.volume })),
            borderColor: "rgba(0, 160, 160, 1)",
            backgroundColor: "rgba(0, 160, 160, 1)",
            yAxisID: "y2",
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: "index", intersect: false },
        elements: { line: { borderWidth: 3, tension: 0.4 }, point: { radius: 0, hoverRadius: 5 } },
        scales: {
          x: {
            type: "linear",
            min: Math.min(...points.map(ts)),
            max: Math.max(...points.map(ts)),
            ticks: { color: text, maxTicksLimit: 6, callback: (v) => fmt(v, { month: "short", day: "numeric" }) },
            grid: { display: false },
          },
          y: { position: "left", ticks: { color: text, maxTicksLimit: 5 }, grid: { color: grid } },
          y2: { position: "right", ticks: { color: text, maxTicksLimit: 5 }, grid: { display: false } },
        },
        plugins: {
          legend: { position: "top", labels: { color: text, usePointStyle: true, boxWidth: 8, boxHeight: 8 } },
          tooltip: {
            callbacks: {
              title: (items) => fmt(items[0].parsed.x, { month: "short", day: "numeric", year: "numeric" }),
              label: (item) =>
                item.dataset.yAxisID === "y"
                  ? `Weight: ${item.parsed.y.toFixed(1)} kg`
                  : `Volume: ${item.parsed.y.toFixed(0)} kg`,
            },
          },
        },
      },
    });
  });
}
document.body.addEventListener("htmx:load", (e) => initCharts(e.target));
initCharts(document);

// Keep the sidebar highlight and tab title in sync with the URL after HTMX swaps,
// including navigations triggered by server responses (HX-Location / HX-Push-Url).
function syncNavigation() {
  const path = location.pathname;
  document.querySelectorAll("#sidebar .nav-link").forEach((a) => {
    a.classList.toggle("active", a.getAttribute("href") === path);
  });
  const h1 = document.querySelector("#content h1");
  if (h1) document.title = h1.textContent + " · Fitness Tracker";
}
document.body.addEventListener("htmx:pushedIntoHistory", syncNavigation);
document.body.addEventListener("htmx:afterSettle", () => setTimeout(syncNavigation, 0));
