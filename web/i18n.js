const translations = {
  en: {
    homeTitle: "MyTurn",
    settingsTitle: "MyTurn Settings",
    settings: "Settings",
    back: "Back",
    today: "Today",
    tomorrow: "Tomorrow",
    serverUnavailable: "Server unavailable",
    comeBackLater: "Please come back later.",
    loading: "Loading…",
    updated: "Updated {time}",
    noTurn: "No turn",
    adminPassword: "Admin password:",
    cancelled: "Cancelled",
    configLoadError: "Could not load settings.",
    todayLoadError: "Could not load today's turn.",
    memberName: "Member name {number}",
    moveUp: "Move up",
    moveDown: "Move down",
    removeMember: "Remove member",
    newMember: "New member",
    general: "General",
    groupName: "Group name",
    timezone: "Time zone (IANA name)",
    startDate: "Start date",
    members: "Members",
    add: "Add",
    todaysTurn: "Today's turn",
    skipToday: "Skip today; today's person moves to tomorrow and following turns shift by one day.",
    saveChanges: "Save changes",
    saved: "Saved.",
    incorrectPassword: "Incorrect admin password.",
    saveFailed: "Could not save changes (HTTP {status}).",
    error: "Error: {message}",
  },
  es: {
    homeTitle: "MyTurn",
    settingsTitle: "Ajustes de MyTurn",
    settings: "Ajustes",
    back: "Volver",
    today: "Hoy",
    tomorrow: "Mañana",
    serverUnavailable: "Servidor no disponible",
    comeBackLater: "Vuelve a intentarlo más tarde.",
    loading: "Cargando…",
    updated: "Actualizado {time}",
    noTurn: "Sin turno",
    adminPassword: "Contraseña de administrador:",
    cancelled: "Cancelado",
    configLoadError: "No se pudieron cargar los ajustes.",
    todayLoadError: "No se pudo cargar el turno de hoy.",
    memberName: "Nombre del miembro {number}",
    moveUp: "Subir",
    moveDown: "Bajar",
    removeMember: "Eliminar miembro",
    newMember: "Nuevo miembro",
    general: "General",
    groupName: "Nombre del grupo",
    timezone: "Zona horaria (nombre IANA)",
    startDate: "Fecha de inicio",
    members: "Miembros",
    add: "Añadir",
    todaysTurn: "Turno de hoy",
    skipToday: "Saltar hoy; la persona de hoy pasa a mañana y los siguientes turnos se desplazan un día.",
    saveChanges: "Guardar cambios",
    saved: "Guardado.",
    incorrectPassword: "Contraseña de administrador incorrecta.",
    saveFailed: "No se pudieron guardar los cambios (HTTP {status}).",
    error: "Error: {message}",
  },
};

const preferredLanguage = (navigator.languages && navigator.languages[0]) || navigator.language || "en";
const language = preferredLanguage.toLowerCase().startsWith("es") ? "es" : "en";
const messages = translations[language];

function translate(key, replacements = {}) {
  return (messages[key] || translations.en[key] || key).replace(/\{(\w+)\}/g, (_, name) => replacements[name] ?? "");
}

document.documentElement.lang = language;
document.querySelectorAll("[data-i18n]").forEach((element) => {
  element.textContent = translate(element.dataset.i18n);
});

window.myTurnI18n = { language, translate };