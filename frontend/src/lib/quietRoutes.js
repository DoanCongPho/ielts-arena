// isQuietRoute reports whether a page must stay silent: taking a test (the
// Listening recording above all) or reviewing one. These are the routes
// App.jsx mounts with `focus`.
export function isQuietRoute(pathname) {
  return (
    /^\/practice\/(reading|listening|writing|speaking)\/\d+\/?$/.test(pathname) ||
    /^\/submissions\/\d+\/?$/.test(pathname)
  );
}
