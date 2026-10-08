// MyTurn settings page: loads /api/config, lets the user edit group name,
// members, start date, and timezone. Access is gated by a shared admin
// secret sent as the X-Admin-Secret header, re-prompted on every page load.

const el = {
  form: document.getElementById("config-form"),
  groupName: document.getElementById("group-name-input"),
  timezone: document.getElementById("timezone-input"),
  startDate: document.getElementById("start-date-input"),
  membersContainer: document.getElementById("members-container"),
  addMemberBtn: document.getElementById("add-member-btn"),
  configStatus: document.getElementById("config-status"),
  skipToday: document.getElementById("skip-today-input"),
};

const t = window.myTurnI18n.translate;
let config = null;
// Held only in memory for this page load, never persisted, so navigating
// away and back to settings always asks for the password again.
let currentSecret = null;

const ICONS = {
  up: '<i class="bi bi-arrow-up" aria-hidden="true"></i>',
  down: '<i class="bi bi-arrow-down" aria-hidden="true"></i>',
  remove: '<i class="bi bi-x-lg" aria-hidden="true"></i>',
};

function getStoredSecret() {
  return currentSecret;
}

// requestSecret prompts once. Canceling sends the user back to the home
// screen instead of prompting again.
function requestSecret() {
  const input = window.prompt(t("adminPassword"));
  if (input === null) {
    window.location.href = "/";
    throw new Error(t("cancelled"));
  }
  currentSecret = input;
  return input;
}

async function authFetch(url, options = {}) {
  const secret = getStoredSecret() ?? requestSecret();
  const res = await fetch(url, {
    ...options,
    headers: { ...(options.headers || {}), "X-Admin-Secret": secret },
  });
  if (res.status === 401) {
    currentSecret = null;
    window.location.href = "/";
    throw new Error(t("incorrectPassword"));
  }
  return res;
}

function setStatus(elm, message, isError) {
  elm.textContent = message;
  elm.classList.remove("text-success", "text-danger");
  elm.classList.add(isError ? "text-danger" : "text-success");
}

function renderMembers() {
  el.membersContainer.innerHTML = "";
  config.members.forEach((name, i) => {
    const row = document.createElement("div");
    row.className = "d-flex align-items-center gap-2 mb-2";

    const input = document.createElement("input");
    input.type = "text";
    input.className = "form-control";
    input.value = name;
    input.setAttribute("aria-label", t("memberName", { number: i + 1 }));
    input.addEventListener("input", () => {
      config.members[i] = input.value;
    });

    const controls = document.createElement("div");
    controls.className = "btn-group flex-shrink-0";
    controls.setAttribute("role", "group");
    controls.setAttribute("aria-label", "Acciones del miembro " + (i + 1));

    const upBtn = document.createElement("button");
    upBtn.type = "button";
    upBtn.className = "btn btn-outline-secondary";
    upBtn.innerHTML = ICONS.up;
    upBtn.setAttribute("aria-label", t("moveUp"));
    upBtn.disabled = i === 0;
    upBtn.addEventListener("click", () => {
      [config.members[i - 1], config.members[i]] = [config.members[i], config.members[i - 1]];
      renderMembers();
    });

    const downBtn = document.createElement("button");
    downBtn.type = "button";
    downBtn.className = "btn btn-outline-secondary";
    downBtn.innerHTML = ICONS.down;
    downBtn.setAttribute("aria-label", t("moveDown"));
    downBtn.disabled = i === config.members.length - 1;
    downBtn.addEventListener("click", () => {
      [config.members[i + 1], config.members[i]] = [config.members[i], config.members[i + 1]];
      renderMembers();
    });

    const removeBtn = document.createElement("button");
    removeBtn.type = "button";
    removeBtn.className = "btn btn-outline-danger";
    removeBtn.innerHTML = ICONS.remove;
    removeBtn.setAttribute("aria-label", t("removeMember"));
    removeBtn.disabled = config.members.length <= 1;
    removeBtn.addEventListener("click", () => {
      config.members.splice(i, 1);
      renderMembers();
    });

    controls.appendChild(upBtn);
    controls.appendChild(downBtn);
    controls.appendChild(removeBtn);
    row.appendChild(input);
    row.appendChild(controls);
    el.membersContainer.appendChild(row);
  });
}

async function loadConfig() {
  const res = await authFetch("/api/config", { cache: "no-store" });
  if (!res.ok) throw new Error(t("configLoadError"));
  config = await res.json();

  el.groupName.value = config.group_name;
  el.timezone.value = config.timezone;
  el.startDate.value = config.start_date;
  renderMembers();

  const stateResponse = await fetch("/api/state", { cache: "no-store" });
  if (!stateResponse.ok) throw new Error(t("todayLoadError"));
  const state = await stateResponse.json();
  el.skipToday.checked = state.today_skipped;
}

el.addMemberBtn.addEventListener("click", () => {
  config.members.push(t("newMember"));
  renderMembers();
});

el.form.addEventListener("submit", async (e) => {
  e.preventDefault();
  config.group_name = el.groupName.value.trim();
  config.timezone = el.timezone.value.trim();
  config.start_date = el.startDate.value;
  config.members = config.members.map((m) => m.trim());

  try {
    const res = await authFetch("/api/config", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ ...config, skip_today: el.skipToday.checked }),
    });
    if (!res.ok) {
      throw new Error(t("saveFailed", { status: res.status }));
    }
    config = await res.json();
    renderMembers();
    setStatus(el.configStatus, t("saved"), false);
  } catch (err) {
    setStatus(el.configStatus, t("error", { message: err.message }), true);
  }
});

loadConfig().catch((err) => setStatus(el.configStatus, t("error", { message: err.message }), true));
