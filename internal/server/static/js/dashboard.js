function cssVar(name) {
  return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

function chartPalette() {
  const accent = cssVar("--accent");
  const success = cssVar("--success");
  const bg = cssVar("--bg");
  return {
    accent,
    accentDim: cssVar("--accent-dim"),
    grid: cssVar("--chart-grid"),
    text: cssVar("--text-muted"),
    success: cssVar("--success"),
    successDim: cssVar("--success-dim"),
    bg,
  };
}

let dashboardCharts = [];

function destroyDashboardCharts() {
  dashboardCharts.forEach((c) => c.destroy());
  dashboardCharts = [];
}

function initDashboardCharts(cfg) {
  if (!cfg || typeof Chart === "undefined") return;

  destroyDashboardCharts();
  const p = chartPalette();

  Chart.defaults.color = p.text;
  Chart.defaults.borderColor = p.grid;
  Chart.defaults.font.family = "'JetBrains Mono', ui-monospace, monospace";

  const timelineEl = document.getElementById("chart-timeline");
  if (timelineEl && cfg.labels) {
    dashboardCharts.push(new Chart(timelineEl, {
      type: "line",
      data: {
        labels: cfg.labels,
        datasets: [
          {
            label: kuiT("chart.page_views"),
            data: cfg.hits || [],
            borderColor: p.accent,
            backgroundColor: p.accentDim,
            fill: true,
            tension: 0.35,
            pointRadius: 2,
            pointHoverRadius: 4,
          },
          {
            label: kuiT("chart.uniques"),
            data: cfg.uniques || [],
            borderColor: p.success,
            backgroundColor: p.successDim,
            fill: false,
            tension: 0.35,
            pointRadius: 2,
          },
        ],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        interaction: { mode: "index", intersect: false },
        plugins: { legend: { labels: { boxWidth: 10, font: { size: 10 } } } },
        scales: {
          x: { ticks: { maxTicksLimit: 8, font: { size: 9 } } },
          y: { beginAtZero: true, ticks: { precision: 0 } },
        },
      },
    }));
  }

  const channelsEl = document.getElementById("chart-channels");
  if (channelsEl && cfg.chLabels && cfg.chLabels.length > 0) {
    dashboardCharts.push(new Chart(channelsEl, {
      type: "doughnut",
      data: {
        labels: cfg.chLabels,
        datasets: [{
          data: cfg.chHits || [],
          backgroundColor: [
            p.accent + "b3",
            p.success + "99",
            "rgba(251, 191, 36, 0.6)",
            "rgba(248, 113, 113, 0.6)",
            "rgba(167, 139, 250, 0.6)",
            "rgba(96, 165, 250, 0.6)",
            "rgba(244, 114, 182, 0.6)",
            "rgba(156, 163, 175, 0.5)",
          ],
          borderColor: p.bg,
          borderWidth: 2,
        }],
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: { legend: { position: "bottom", labels: { boxWidth: 10, font: { size: 9 } } } },
      },
    }));
  }
}

document.addEventListener("DOMContentLoaded", () => {
  document.querySelectorAll("[data-tab]").forEach((btn) => {
    btn.addEventListener("click", () => {
      const tab = btn.dataset.tab;
      document.querySelectorAll("[data-tab]").forEach((b) => b.classList.toggle("active", b.dataset.tab === tab));
      document.querySelectorAll("[data-panel]").forEach((p) => p.classList.toggle("active", p.dataset.panel === tab));
    });
  });

  const cfgEl = document.getElementById("dashboard-config");
  if (!cfgEl) return;

  let cfg;
  try {
    cfg = JSON.parse(cfgEl.dataset.json || "{}");
  } catch {
    return;
  }

  initDashboardCharts(cfg);
});

document.addEventListener("kui-theme-change", () => {
  const cfgEl = document.getElementById("dashboard-config");
  if (!cfgEl) return;
  try {
    initDashboardCharts(JSON.parse(cfgEl.dataset.json || "{}"));
  } catch {
    destroyDashboardCharts();
  }
});
