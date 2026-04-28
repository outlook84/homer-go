(function () {
  var minIntervalMs = 1000;
  var servicesRefreshing = false;
  var messageRefreshing = false;
  var offline = false;
  var connectivityCheckID = 0;
  var offlineRecoveryTimer = 0;
  var offlineRecoveryIntervalMs = 5000;

  function interval(value) {
    var n = Number(value || 0);
    if (n <= 0) return 0;
    return Math.max(n, minIntervalMs);
  }

  function appPath(path) {
    var base = String(window.HOMER_BASE_PATH || "").replace(/\/+$/, "");
    if (!path || path.charAt(0) !== "/") path = "/" + path;
    return base + path;
  }

  function fragmentURL(path) {
    var url = new URL(appPath(path), window.location.origin);
    url.search = window.location.search;
    if (window.HOMER_PAGE && window.HOMER_PAGE !== "default") {
      url.searchParams.set("page", window.HOMER_PAGE);
    }
    return url;
  }

  async function replaceFragment(selector, url) {
    var root = document.querySelector(selector);
    if (!root) return;

    var response = await fetch(url, { cache: "no-store" });
    if (!response.ok) return;

    root.outerHTML = await response.text();
  }

  async function refreshServices() {
    if (servicesRefreshing || document.hidden || offline) return;
    servicesRefreshing = true;
    try {
      await replaceFragment(
        "[data-services-fragment]",
        fragmentURL("/fragments/services"),
      );
    } finally {
      servicesRefreshing = false;
    }
  }

  async function refreshMessage() {
    if (messageRefreshing || document.hidden || offline) return;
    messageRefreshing = true;
    try {
      await replaceFragment(
        "[data-message-fragment]",
        fragmentURL("/fragments/message"),
      );
    } finally {
      messageRefreshing = false;
    }
  }

  function setOffline(isOffline) {
    offline = isOffline;
    var message = document.querySelector("[data-offline-message]");
    var content = document.querySelector("[data-online-content]");
    if (message) message.hidden = !isOffline;
    if (content) content.hidden = isOffline;
    syncOfflineRecoveryTimer();
  }

  function syncOfflineRecoveryTimer() {
    if (!window.HOMER_CONNECTIVITY_CHECK) return;

    if (!offline || document.hidden) {
      if (offlineRecoveryTimer) {
        window.clearTimeout(offlineRecoveryTimer);
        offlineRecoveryTimer = 0;
      }
      return;
    }

    if (!offlineRecoveryTimer) {
      offlineRecoveryTimer = window.setTimeout(function () {
        offlineRecoveryTimer = 0;
        checkOffline().finally(syncOfflineRecoveryTimer);
      }, offlineRecoveryIntervalMs);
    }
  }

  function clearConnectivityTimestamp() {
    if (!window.HOMER_CONNECTIVITY_CHECK || !/[?&]t=\d+/.test(window.location.href)) {
      return;
    }

    var cleanUrl = new URL(window.location.href);
    cleanUrl.searchParams.delete("t");
    window.history.replaceState(
      {},
      document.title,
      cleanUrl.pathname + cleanUrl.search + cleanUrl.hash,
    );
  }

  async function checkOffline() {
    if (!window.HOMER_CONNECTIVITY_CHECK) return;

    var checkID = ++connectivityCheckID;

    if (!window.navigator.onLine) {
      if (checkID === connectivityCheckID) setOffline(true);
      return;
    }

    var aliveCheckUrl = new URL(window.location.href);
    aliveCheckUrl.searchParams.set("t", Date.now());

    try {
      var response = await fetch(aliveCheckUrl, {
        method: "HEAD",
        cache: "no-store",
        redirect: "manual",
      });
      if (
        (response.type === "opaqueredirect" && !response.ok) ||
        response.status === 401 ||
        response.status === 403
      ) {
        if (checkID === connectivityCheckID) {
          window.location.href = aliveCheckUrl;
        }
        return;
      }
      if (checkID === connectivityCheckID) setOffline(!response.ok);
    } catch (_) {
      if (checkID === connectivityCheckID) setOffline(true);
    }
  }

  var servicesInterval = interval(window.HOMER_UPDATE_INTERVAL_MS);
  var messageInterval = interval(window.HOMER_MESSAGE_REFRESH_INTERVAL);

  clearConnectivityTimestamp();
  checkOffline().finally(function () {
    refreshMessage();
    refreshServices();
  });

  if (servicesInterval > 0) {
    window.setInterval(refreshServices, servicesInterval);
  }

  if (messageInterval > 0) {
    window.setInterval(refreshMessage, messageInterval);
  }

  document.addEventListener("visibilitychange", function () {
    if (document.hidden) {
      syncOfflineRecoveryTimer();
      return;
    }
    checkOffline();
    syncOfflineRecoveryTimer();
    if (servicesInterval > 0) refreshServices();
    if (messageInterval > 0) refreshMessage();
  });

  window.addEventListener("online", checkOffline);
  window.addEventListener("offline", function () {
    setOffline(true);
  });

  var retry = document.querySelector("[data-offline-retry]");
  if (retry) {
    retry.addEventListener("click", checkOffline);
  }
})();
