package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:19090", "HTTP listen address")
	fixtures := flag.String("fixtures", filepath.Join("internal", "collectors", "testdata"), "collector fixture directory")
	flag.Parse()

	root, err := filepath.Abs(*fixtures)
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler(root))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		stop()
	}()

	server := &http.Server{
		Addr:    *addr,
		Handler: mux,
	}
	log.Printf("browser mock listening on http://%s, fixtures=%s", *addr, root)
	if err := serve(ctx, server, 5*time.Second); err != nil {
		log.Fatal(err)
	}
}

func serve(ctx context.Context, server *http.Server, shutdownTimeout time.Duration) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
	}

	log.Printf("shutdown signal received; draining HTTP server for up to %s", shutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v; forcing HTTP server close", err)
		if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			log.Printf("forced HTTP server close failed: %v", closeErr)
		}
		return err
	}

	if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Printf("shutdown complete")
	return nil
}

func handler(root string) http.HandlerFunc {
	routes := map[string]string{
		"/adguardhome/control/status":                 "adguard_status.json",
		"/adguardhome/control/stats":                  "adguard_stats.json",
		"/docuseal/version":                           "docuseal_version.txt",
		"/dockersocketproxy/containers/json?all=true": "dockersocketproxy_containers.json",
		"/emby/System/info/public":                    "emby_info.json",
		"/emby/items/counts":                          "emby_counts.json",
		"/freshrss/api/greader.php/accounts/ClientLogin?Email=user&Passwd=pass": "freshrss_login.txt",
		"/freshrss/api/greader.php/reader/api/0/subscription/list?output=json":  "freshrss_subscriptions.json",
		"/freshrss/api/greader.php/reader/api/0/unread-count?output=json":       "freshrss_unread.json",
		"/gatus/api/v1/endpoints/statuses":                                      "gatus_statuses.json",
		"/gitea/swagger.v1.json":                                                "gitea_swagger.json",
		"/glances/api/4/quicklook":                                              "glances_quicklook.json",
		"/gotify/health":                                                        "gotify_health.json",
		"/gotify/message?limit=100":                                             "gotify_messages.json",
		"/healthchecks/api/v1/checks/":                                          "healthchecks_checks.json",
		"/homeassistant/api/":                                                   "homeassistant_api.json",
		"/homeassistant/api/config":                                             "homeassistant_config.json",
		"/homeassistant/api/states":                                             "homeassistant_states.json",
		"/hyperhdr/json-rpc?request=%7B%22command%22%3A%22serverinfo%22%7D":     "hyperhdr_serverinfo.json",
		"/immich/api/server/statistics":                                         "immich_statistics.json",
		"/jellyfin/Sessions":                                                    "jellyfin_sessions.json",
		"/lidarr/api/v1/health?apikey=arr-token":                                "arr_health.json",
		"/lidarr/api/v1/queue/status?apikey=arr-token":                          "arr_queue.json",
		"/lidarr/api/v1/wanted/missing?apikey=arr-token":                        "arr_missing.json",
		"/matrix/_matrix/federation/v1/version":                                 "matrix_version.json",
		"/mealie/api/groups/mealplans/today":                                    "mealie_today.json",
		"/mealie/api/admin/about/statistics":                                    "mealie_stats.json",
		"/medusa/api/v2/config":                                                 "medusa_config.json",
		"/miniflux/v1/feeds/counters":                                           "miniflux_counters.json",
		"/mylar/api?cmd=getUpcoming&apikey=mylar-token":                         "mylar_upcoming.json",
		"/mylar/api?cmd=getWanted&apikey=mylar-token":                           "mylar_wanted.json",
		"/netalertx/devices/totals":                                             "netalertx_totals.json",
		"/nextcloud/status.php":                                                 "nextcloud_status.json",
		"/openhab/rest/systeminfo":                                              "openhab_systeminfo.json",
		"/openhab/rest/things?summary=true":                                     "openhab_things.json",
		"/openhab/rest/items":                                                   "openhab_items.json",
		"/olivetin/webUiSettings.json":                                          "olivetin_settings.json",
		"/paperlessng/api/documents/":                                           "paperless_documents.json",
		"/peanut/api/v1/devices/ups":                                            "peanut_device.json",
		"/pialert/php/server/devices.php?action=getDevicesTotals":               "pialert_totals.json",
		"/portainer/api/endpoints":                                              "portainer_endpoints.json",
		"/portainer/api/endpoints/1/docker/containers/json?all=1":               "portainer_containers.json",
		"/portainer/api/status":                                                 "portainer_status.json",
		"/prometheus/api/v1/alerts":                                             "prometheus_alerts.json",
		"/prowlarr/api/v1/health?apikey=prowl-token":                            "prowlarr_health.json",
		"/proxmox/api2/json/nodes/node1/status":                                 "proxmox_status.json",
		"/proxmox/api2/json/nodes/node1/qemu":                                   "proxmox_qemu.json",
		"/proxmox/api2/json/nodes/node1/lxc":                                    "proxmox_lxc.json",
		"/qbittorrent/api/v2/torrents/info":                                     "qbittorrent_torrents.json",
		"/qbittorrent/api/v2/transfer/info":                                     "qbittorrent_transfer.json",
		"/radarr/api/v3/health?apikey=arr-token":                                "arr_health.json",
		"/radarr/api/v3/queue?apikey=arr-token":                                 "arr_queue.json",
		"/radarr/api/v3/queue/details?apikey=arr-token":                         "radarr_queue_details.json",
		"/radarr/api/v3/wanted/missing?apikey=arr-token":                        "arr_missing.json",
		"/readarr/api/v1/health?apikey=arr-token":                               "arr_health.json",
		"/readarr/api/v1/queue?apikey=arr-token":                                "arr_queue.json",
		"/readarr/api/v1/wanted/missing?apikey=arr-token":                       "arr_missing.json",
		"/sabnzbd/api?output=json&apikey=sab-token&mode=queue":                  "sabnzbd_queue.json",
		"/scrutiny/api/summary":                                                 "scrutiny_summary.json",
		"/speedtesttracker/api/v1/results/latest":                               "speedtest_latest.json",
		"/sonarr/api/v3/health?apikey=arr-token":                                "arr_health.json",
		"/sonarr/api/v3/queue?apikey=arr-token":                                 "arr_queue.json",
		"/sonarr/api/v3/wanted/missing?apikey=arr-token":                        "arr_missing.json",
		"/speedtesttracker/api/speedtest/latest":                                "speedtest_latest.json",
		"/tautulli/api/v2?apikey=tautulli-token&cmd=get_activity":               "tautulli_activity.json",
		"/tdarr/api/v2/cruddb":                                                  "tdarr_stats.json",
		"/traefik/api/version":                                                  "traefik_version.json",
		"/truenasscale/api/v2.0/system/version":                                 "truenasscale_version.json",
		"/uptimekuma/api/status-page/default":                                   "uptimekuma_page.json",
		"/uptimekuma/api/status-page/heartbeat/default":                         "uptimekuma_heartbeat.json",
		"/vaultwarden/api/version":                                              "vaultwarden_version.json",
		"/wallabag/api/version":                                                 "wallabag_version.json",
		"/wud/api/containers":                                                   "wud_containers.json",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead || r.URL.Path == "/ping" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		fixture, ok := routes[r.URL.RequestURI()]
		if !ok {
			fixture, ok = routes[r.URL.Path]
		}
		if !ok {
			http.NotFound(w, r)
			return
		}

		path := filepath.Join(root, fixture)
		body, err := os.ReadFile(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(body)
	}
}
