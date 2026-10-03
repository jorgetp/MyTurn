const colorPreference = window.matchMedia("(prefers-color-scheme: dark)");

function applyColorPreference() {
  document.documentElement.setAttribute("data-bs-theme", colorPreference.matches ? "dark" : "light");
}

applyColorPreference();
colorPreference.addEventListener("change", applyColorPreference);
