// MyTurn frontend: fetches /api/state, renders the rotation, and keeps the
// page in sync via polling and day-change detection.

const POLL_INTERVAL_MS = 30000;
const t = window.myTurnI18n.translate;

const el = {
  groupName: document.getElementById("group-name"),
  todayPerson: document.getElementById("today-person"),
  tomorrowPerson: document.getElementById("tomorrow-person"),
  content: document.getElementById("content"),
  errorState: document.getElementById("error-state"),
  lastUpdated: document.getElementById("last-updated"),
};

let lastKnownDate = null;

// formatTimestamp renders "HH:MM AM/PM" for today, or "DD MM YYYY HH:MM AM/PM"
// for any other day.
function formatTimestamp(date) {
  const now = new Date();
  const isToday =
    date.getFullYear() === now.getFullYear() &&
    date.getMonth() === now.getMonth() &&
    date.getDate() === now.getDate();

  let hours = date.getHours();
  const ampm = hours >= 12 ? "PM" : "AM";
  hours = hours % 12 || 12;
  const minutes = String(date.getMinutes()).padStart(2, "0");
  const time = `${hours}:${minutes} ${ampm}`;

  if (isToday) return time;

  const day = String(date.getDate()).padStart(2, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  return `${day} ${month} ${date.getFullYear()} ${time}`;
}

function render(state) {
  el.content.classList.remove("d-none");
  el.errorState.classList.add("d-none");

  el.groupName.textContent = state.group_name || "MyTurn";
  el.todayPerson.textContent = state.today_skipped ? t("noTurn") : state.today || "—";
  el.tomorrowPerson.textContent = state.tomorrow || "—";
  el.lastUpdated.textContent = t("updated", { time: formatTimestamp(new Date()) });

  lastKnownDate = state.date;
}

function showError() {
  el.content.classList.add("d-none");
  el.errorState.classList.remove("d-none");
  el.lastUpdated.textContent = "";
}

async function fetchState() {
  try {
    const res = await fetch("/api/state", { cache: "no-store" });
    if (!res.ok) throw new Error("bad status " + res.status);
    const state = await res.json();
    render(state);
  } catch (err) {
    showError();
  }
}

function checkForDayChange() {
  const today = new Date().toLocaleDateString("en-CA"); // YYYY-MM-DD local
  if (lastKnownDate && today !== lastKnownDate) {
    fetchState();
  }
}

fetchState();
setInterval(() => {
  fetchState();
  checkForDayChange();
}, POLL_INTERVAL_MS);

document.addEventListener("visibilitychange", () => {
  if (document.visibilityState === "visible") {
    fetchState();
  }
});

if ("serviceWorker" in navigator) {
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("service-worker.js").catch(() => {
      // Non-fatal: app still works without offline shell caching.
    });
  });
}
