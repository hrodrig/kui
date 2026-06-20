function kuiT(key) {
  const i18n = window.kuiI18n || {};
  return i18n[key] || key;
}

function closeConfirmModal() {
  const modal = document.getElementById("confirm-modal");
  if (!modal) return;
  modal.hidden = true;
  modal.setAttribute("aria-hidden", "true");
  document.body.classList.remove("modal-open");
}

function currentTheme() {
  return document.documentElement.getAttribute("data-theme") === "light" ? "light" : "dark";
}

function applyTheme(theme) {
  const next = theme === "light" ? "light" : "dark";
  document.documentElement.setAttribute("data-theme", next);
  localStorage.setItem("kui-theme", next);
  document.querySelectorAll("[data-theme-toggle]").forEach((btn) => {
    const light = next === "light";
    btn.setAttribute("aria-pressed", light ? "true" : "false");
    btn.setAttribute("aria-label", light ? kuiT("common.theme_dark") : kuiT("common.theme_light"));
  });
  document.dispatchEvent(new CustomEvent("kui-theme-change", { detail: { theme: next } }));
}

function initThemeToggle() {
  applyTheme(currentTheme());
}

document.addEventListener("click", (e) => {
  const themeBtn = e.target.closest("[data-theme-toggle]");
  if (themeBtn) {
    applyTheme(currentTheme() === "light" ? "dark" : "light");
    return;
  }

  const btn = e.target.closest("[data-password-toggle]");
  if (btn) {
    const input = document.getElementById(btn.dataset.passwordToggle);
    if (!input) return;

    const show = input.type === "password";
    input.type = show ? "text" : "password";
    btn.setAttribute("aria-label", show ? kuiT("common.hide_password") : kuiT("common.show_password"));
    btn.setAttribute("aria-pressed", show ? "true" : "false");
    return;
  }

  const open = e.target.closest("[data-confirm-open]");
  if (open) {
    const modal = document.getElementById("confirm-modal");
    const form = document.getElementById("confirm-form");
    if (!modal || !form) return;

    form.action = open.dataset.confirmAction || "";
    const title = modal.querySelector("[data-confirm-title]");
    const body = modal.querySelector("[data-confirm-body]");
    if (title) title.textContent = open.dataset.confirmTitle || "Confirm";
    if (body) body.textContent = open.dataset.confirmBody || "Are you sure?";
    const submit = form.querySelector("[data-confirm-submit]");
    if (submit) submit.textContent = open.dataset.confirmSubmit || "Delete";

    modal.hidden = false;
    modal.setAttribute("aria-hidden", "false");
    document.body.classList.add("modal-open");
    modal.querySelector(".modal-actions [data-confirm-cancel]")?.focus();
    return;
  }

  if (e.target.closest("[data-confirm-cancel]")) {
    closeConfirmModal();
  }
});

document.addEventListener("keydown", (e) => {
  if (e.key === "Escape") closeConfirmModal();
});

initThemeToggle();
