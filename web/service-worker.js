// MyTurn service worker: caches the static app shell so it can still open
// when the local server is briefly unreachable. API requests always go to
// the network so turn information is never served stale from cache.

const CACHE_NAME = "myturn-shell-v15";
const SHELL_ASSETS = [
  "/",
  "/settings.html",
  "/bootstrap.min.css",
  "/bootstrap-theme.css",
  "/bootstrap-icons/bootstrap-icons.min.css",
  "/bootstrap-icons/fonts/bootstrap-icons.woff2",
  "/bootstrap-icons/fonts/bootstrap-icons.woff",
  "/theme.js",
  "/app.js",
  "/settings.js",
  "/manifest.json",
  "/icons/icon-192.png",
  "/icons/icon-512.png",
];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(SHELL_ASSETS)).then(() => self.skipWaiting())
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);

  // Never cache API responses: let the page's own JS handle offline state.
  if (url.pathname.startsWith("/api/")) {
    return;
  }

  event.respondWith(
    caches.match(event.request).then((cached) => {
      const network = fetch(event.request)
        .then((response) => {
          if (response.ok) {
            const copy = response.clone();
            caches.open(CACHE_NAME).then((cache) => cache.put(event.request, copy));
          }
          return response;
        })
        .catch(() => cached);
      return cached || network;
    })
  );
});
