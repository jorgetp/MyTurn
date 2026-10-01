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
};

let config = null;
// Held only in memory for this page load, never persisted, so navigating
// away and back to settings always asks for the password again.
let currentSecret = null;

const ICONS = {
  up: '<svg xmlns="http://www.w3.org/2000/svg" class="icon" viewBox="0 0 16 16"><path fill-rule="evenodd" d="M8 15a.5.5 0 0 0 .5-.5V2.707l3.146 3.147a.5.5 0 0 0 .708-.708l-4-4a.5.5 0 0 0-.708 0l-4 4a.5.5 0 1 0 .708.708L7.5 2.707V14.5a.5.5 0 0 0 .5.5"/></svg>',
  down: '<svg xmlns="http://www.w3.org/2000/svg" class="icon" viewBox="0 0 16 16"><path fill-rule="evenodd" d="M8 1a.5.5 0 0 1 .5.5v11.793l3.146-3.147a.5.5 0 0 1 .708.708l-4 4a.5.5 0 0 1-.708 0l-4-4a.5.5 0 0 1 .708-.708L7.5 13.293V1.5A.5.5 0 0 1 8 1"/></svg>',
  remove: '<svg xmlns="http://www.w3.org/2000/svg" class="icon" viewBox="0 0 16 16"><path d="M2.146 2.854a.5.5 0 1 1 .708-.708L8 7.293l5.146-5.147a.5.5 0 0 1 .708.708L8.707 8l5.147 5.146a.5.5 0 0 1-.708.708L8 8.707l-5.146 5.147a.5.5 0 0 1-.708-.708L7.293 8z"/></svg>',
};

function getStoredSecret() {
  return currentSecret;
}

// requestSecret prompts once. Canceling sends the user back to the home
// screen instead of prompting again.
function requestSecret() {
  const input = window.prompt("Contraseña de administrador:");
  if (input === null) {
    window.location.href = "/";
    throw new Error("cancelado");
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
    throw new Error("contraseña incorrecta");
  }
  return res;
}

function setStatus(elm, message, isError) {
  elm.textContent = message;
  elm.classList.toggle("ok", !isError);
  elm.classList.toggle("error", !!isError);
}

function renderMembers() {
  el.membersContainer.innerHTML = "";
  config.members.forEach((name, i) => {
    const row = document.createElement("div");
    row.className = "member-row";

    const input = document.createElement("input");
    input.type = "text";
    input.value = name;
    input.addEventListener("input", () => {
      config.members[i] = input.value;
    });

    const controls = document.createElement("div");
    controls.className = "controls";

    const upBtn = document.createElement("button");
    upBtn.type = "button";
    upBtn.className = "btn";
    upBtn.innerHTML = ICONS.up;
    upBtn.setAttribute("aria-label", "Subir");
    upBtn.disabled = i === 0;
    upBtn.addEventListener("click", () => {
      [config.members[i - 1], config.members[i]] = [config.members[i], config.members[i - 1]];
      renderMembers();
    });

    const downBtn = document.createElement("button");
    downBtn.type = "button";
    downBtn.className = "btn";
    downBtn.innerHTML = ICONS.down;
    downBtn.setAttribute("aria-label", "Bajar");
    downBtn.disabled = i === config.members.length - 1;
    downBtn.addEventListener("click", () => {
      [config.members[i + 1], config.members[i]] = [config.members[i], config.members[i + 1]];
      renderMembers();
    });

    const removeBtn = document.createElement("button");
    removeBtn.type = "button";
    removeBtn.className = "btn danger";
    removeBtn.innerHTML = ICONS.remove;
    removeBtn.setAttribute("aria-label", "Eliminar miembro");
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
  if (!res.ok) throw new Error("no se pudo cargar la configuración");
  config = await res.json();

  el.groupName.value = config.group_name;
  el.timezone.value = config.timezone;
  el.startDate.value = config.start_date;
  renderMembers();
}

el.addMemberBtn.addEventListener("click", () => {
  config.members.push("Nuevo miembro");
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
      body: JSON.stringify(config),
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.error || "no se pudo guardar (" + res.status + ")");
    }
    config = await res.json();
    renderMembers();
    setStatus(el.configStatus, "Guardado.", false);
  } catch (err) {
    setStatus(el.configStatus, "Error: " + err.message, true);
  }
});

loadConfig().catch((err) => setStatus(el.configStatus, "Error al cargar la configuración: " + err.message, true));
