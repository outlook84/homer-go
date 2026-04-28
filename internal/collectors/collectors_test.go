package collectors

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"homer-go/internal/config"
)

func TestPingAppliesProxyHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer global" {
			t.Fatalf("Authorization header = %q, want global header", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	registry := NewRegistry()
	registry.Register(Ping{})
	cfg := config.Config{
		Proxy: config.Proxy{Headers: map[string]string{"Authorization": "Bearer global"}},
		Services: []config.Group{{
			Items: []config.Item{{
				Name: "App",
				Type: "Ping",
				URL:  server.URL,
			}},
		}},
	}

	statuses := registry.Collect(context.Background(), cfg, time.Second)

	if got := statuses[Key(0, 0)].State; got != "online" {
		t.Fatalf("status = %q, want online", got)
	}
}

func TestVersionCollectors(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		fixture string
		item    config.Item
		want    string
		c       Collector
	}{
		{
			name:    "Vaultwarden",
			path:    "/api/version",
			fixture: "testdata/vaultwarden_version.json",
			item:    config.Item{Type: "Vaultwarden"},
			want:    "Version 1.32.7",
			c:       Vaultwarden{},
		},
		{
			name:    "Wallabag",
			path:    "/api/version",
			fixture: "testdata/wallabag_version.json",
			item:    config.Item{Type: "Wallabag"},
			want:    "Version 2.6.10",
			c:       Wallabag{},
		},
		{
			name:    "Traefik",
			path:    "/api/version",
			fixture: "testdata/traefik_version.json",
			item:    config.Item{Type: "Traefik", Raw: map[string]any{"basic_auth": "user:pass"}},
			want:    "Version 3.2.1",
			c:       Traefik{},
		},
		{
			name:    "Docuseal",
			path:    "/version",
			fixture: "testdata/docuseal_version.txt",
			item:    config.Item{Type: "Docuseal"},
			want:    "Version 1.8.2",
			c:       Docuseal{},
		},
		{
			name:    "Gitea",
			path:    "/swagger.v1.json",
			fixture: "testdata/gitea_swagger.json",
			item:    config.Item{Type: "Gitea"},
			want:    "Version 1.22.3",
			c:       Gitea{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					t.Fatalf("path = %q, want %q", r.URL.Path, tt.path)
				}
				if tt.name == "Traefik" && r.Header.Get("Authorization") != "Basic dXNlcjpwYXNz" {
					t.Fatalf("Authorization header = %q, want basic auth", r.Header.Get("Authorization"))
				}
				writeFixture(t, w, tt.fixture)
			}))
			defer server.Close()

			tt.item.URL = server.URL
			status := tt.c.Collect(context.Background(), tt.item, config.Proxy{})

			if status.State != "online" {
				t.Fatalf("state = %q, want online; detail=%s", status.State, status.Detail)
			}
			if status.Label != tt.want {
				t.Fatalf("label = %q, want %q", status.Label, tt.want)
			}
		})
	}
}

func TestGotifyCollectsHealthAndMessageStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/health":
			writeFixture(t, w, "testdata/gotify_health.json")
		case "/message?limit=100":
			if got := r.Header.Get("X-Gotify-Key"); got != "token" {
				t.Fatalf("X-Gotify-Key header = %q, want token", got)
			}
			writeFixture(t, w, "testdata/gotify_messages.json")
		default:
			t.Fatalf("unexpected request: %s", r.URL.RequestURI())
		}
	}))
	defer server.Close()

	status := Gotify{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"apikey": "token"},
	}, config.Proxy{})

	if status.State != "warning" || status.Tone != "warning" {
		t.Fatalf("status = %#v, want warning state and tone", status)
	}
	if status.Label != "2 messages" {
		t.Fatalf("label = %q, want 2 messages", status.Label)
	}
}

func TestAdGuardHomeCollectsProtectionAndBlockedPercentage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Basic YWRndWFyZDpzZWNyZXQ=" {
			t.Fatalf("Authorization = %q, want AdGuard Home basic auth", got)
		}
		switch r.URL.Path {
		case "/control/status":
			writeFixture(t, w, "testdata/adguard_status.json")
		case "/control/stats":
			writeFixture(t, w, "testdata/adguard_stats.json")
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	status := AdGuardHome{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"username": "adguard", "password": "secret"},
	}, config.Proxy{})

	if status.State != "enabled" || status.Tone != "success" {
		t.Fatalf("status = %#v, want enabled success", status)
	}
	if status.Label != "10.00% blocked" {
		t.Fatalf("label = %q, want 10.00%% blocked", status.Label)
	}
	if status.Indicator != "enabled" {
		t.Fatalf("indicator = %q, want enabled", status.Indicator)
	}
}

func TestHealthchecksCollectsStatusBadges(t *testing.T) {
	server := fixtureServer(t, map[string]string{
		"/api/v1/checks/": "testdata/healthchecks_checks.json",
	})
	defer server.Close()

	status := Healthchecks{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"apikey": "health-token"},
	}, config.Proxy{})

	assertBadge(t, status, "Up", "2", "success")
	assertBadge(t, status, "Down", "1", "danger")
	assertBadge(t, status, "Grace", "1", "warning")
}

func TestJellyfinCollectsPlayingBadge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Sessions" {
			t.Fatalf("path = %q, want /Sessions", r.URL.Path)
		}
		if got := r.Header.Get("X-Emby-Authorization"); got != `MediaBrowser Client="homer-go", Device="homer-go", DeviceId="homer-go", Version="1.0.0", Token="jelly-token"` {
			t.Fatalf("X-Emby-Authorization header = %q, want MediaBrowser token", got)
		}
		if got := r.Header.Get("X-Emby-Token"); got != "jelly-token" {
			t.Fatalf("X-Emby-Token header = %q, want jelly-token fallback", got)
		}
		writeFixture(t, w, "testdata/jellyfin_sessions.json")
	}))
	defer server.Close()

	status := Jellyfin{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"apikey": "jelly-token"},
	}, config.Proxy{})

	assertBadge(t, status, "Playing", "2", "info")
}

func TestMealiePrefersTodayMeal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/groups/mealplans/today":
			if got := r.Header.Get("Authorization"); got != "Bearer mealie-token" {
				t.Fatalf("Authorization header = %q, want bearer token", got)
			}
			writeFixture(t, w, "testdata/mealie_today.json")
		case "/api/admin/about/statistics":
			t.Fatal("statistics should not be requested when a meal is planned")
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	status := Mealie{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"apikey": "mealie-token"},
	}, config.Proxy{})

	if status.Label != "Today: Tomato Soup" {
		t.Fatalf("label = %q, want today's meal", status.Label)
	}
}

func TestMedusaCollectsConfigBadges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/config" {
			t.Fatalf("path = %q, want /api/v2/config", r.URL.Path)
		}
		if got := r.Header.Get("X-Api-Key"); got != "medusa-token" {
			t.Fatalf("X-Api-Key header = %q, want medusa-token", got)
		}
		writeFixture(t, w, "testdata/medusa_config.json")
	}))
	defer server.Close()

	status := Medusa{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"apikey": "medusa-token"},
	}, config.Proxy{})

	assertBadge(t, status, "News", "2", "neutral")
	assertBadge(t, status, "Warning", "3", "warning")
	assertBadge(t, status, "Error", "1", "danger")
}

func TestMinifluxCollectsUnreadStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/feeds/counters" {
			t.Fatalf("path = %q, want /v1/feeds/counters", r.URL.Path)
		}
		if got := r.Header.Get("X-Auth-Token"); got != "mini-token" {
			t.Fatalf("X-Auth-Token header = %q, want mini-token", got)
		}
		writeFixture(t, w, "testdata/miniflux_counters.json")
	}))
	defer server.Close()

	status := Miniflux{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"apikey": "mini-token"},
	}, config.Proxy{})

	if status.State != "unread" || status.Tone != "info" {
		t.Fatalf("status = %#v, want unread info", status)
	}
	if status.Label != "8 unread in 2 feeds" {
		t.Fatalf("label = %q, want unread feed summary", status.Label)
	}
	if status.Indicator != "Unread" {
		t.Fatalf("indicator = %q, want Unread", status.Indicator)
	}
	if len(status.Badges) != 0 {
		t.Fatalf("badges = %#v, want none for status-style Miniflux", status.Badges)
	}
}

func TestPaperlessNGCollectsDocumentCount(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/documents/" {
			t.Fatalf("path = %q, want /api/documents/", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Token paper-token" {
			t.Fatalf("Authorization header = %q, want token", got)
		}
		writeFixture(t, w, "testdata/paperless_documents.json")
	}))
	defer server.Close()

	status := PaperlessNG{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"apikey": "paper-token"},
	}, config.Proxy{})

	if status.State != "online" {
		t.Fatalf("state = %q, want online", status.State)
	}
	if status.Label != "happily storing 42 documents" {
		t.Fatalf("label = %q, want document count", status.Label)
	}
}

func TestProwlarrCollectsHealthBadges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/api/v1/health?apikey=prowl-token" {
			t.Fatalf("request URI = %q, want health API", r.URL.RequestURI())
		}
		writeFixture(t, w, "testdata/prowlarr_health.json")
	}))
	defer server.Close()

	status := Prowlarr{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"apikey": "prowl-token"},
	}, config.Proxy{})

	assertBadge(t, status, "Warning", "1", "warning")
	assertBadge(t, status, "Error", "2", "danger")
}

func TestPrometheusCollectsHighestSeverityAlertStatus(t *testing.T) {
	server := fixtureServer(t, map[string]string{
		"/api/v1/alerts": "testdata/prometheus_alerts.json",
	})
	defer server.Close()

	status := Prometheus{}.Collect(context.Background(), config.Item{
		URL: server.URL,
	}, config.Proxy{})

	if status.State != "firing" || status.Tone != "danger" {
		t.Fatalf("status = %#v, want firing danger", status)
	}
	if status.Label != "2 firing alerts" {
		t.Fatalf("label = %q, want 2 firing alerts", status.Label)
	}
	if status.Indicator != "2" {
		t.Fatalf("indicator = %q, want 2", status.Indicator)
	}
}

func TestTautulliCollectsPlayingBadge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/api/v2?apikey=tautulli-token&cmd=get_activity" {
			t.Fatalf("request URI = %q, want activity API", r.URL.RequestURI())
		}
		writeFixture(t, w, "testdata/tautulli_activity.json")
	}))
	defer server.Close()

	status := Tautulli{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"apikey": "tautulli-token"},
	}, config.Proxy{})

	assertBadge(t, status, "Playing", "3", "info")
}

func TestScrutinyCollectsDeviceBadges(t *testing.T) {
	server := fixtureServer(t, map[string]string{
		"/api/summary": "testdata/scrutiny_summary.json",
	})
	defer server.Close()

	status := Scrutiny{}.Collect(context.Background(), config.Item{
		URL: server.URL,
	}, config.Proxy{})

	assertBadge(t, status, "Passed", "1", "success")
	assertBadge(t, status, "Failed", "1", "danger")
	assertBadge(t, status, "Unknown", "1", "warning")
}

func TestWUDCollectsContainerBadges(t *testing.T) {
	server := fixtureServer(t, map[string]string{
		"/api/containers": "testdata/wud_containers.json",
	})
	defer server.Close()

	status := WUD{}.Collect(context.Background(), config.Item{
		URL: server.URL,
	}, config.Proxy{})

	assertBadge(t, status, "Running", "3", "warning")
	assertBadge(t, status, "Update", "2", "danger")
}

func TestHomerVersionCollectors(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		body    string
		item    config.Item
		want    string
		c       Collector
		header  string
		wantHdr string
	}{
		{name: "Matrix", path: "/_matrix/federation/v1/version", body: "testdata/matrix_version.json", item: config.Item{Type: "Matrix"}, want: "Version 1.99.0", c: Matrix{}},
		{name: "Nextcloud", path: "/status.php", body: "testdata/nextcloud_status.json", item: config.Item{Type: "Nextcloud"}, want: "Version 28.0.2", c: Nextcloud{}},
		{name: "TruenasScale", path: "/api/v2.0/system/version", body: "testdata/truenasscale_version.json", item: config.Item{Type: "TruenasScale", Raw: map[string]any{"api_token": "truenas-token"}}, want: "Version TrueNAS-SCALE-22.12.4.2", c: TruenasScale{}, header: "Authorization", wantHdr: "Bearer truenas-token"},
		{name: "Olivetin", path: "/webUiSettings.json", body: "testdata/olivetin_settings.json", item: config.Item{Type: "Olivetin"}, want: "Version 2024.11.24", c: Olivetin{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != tt.path {
					t.Fatalf("path = %q, want %q", r.URL.Path, tt.path)
				}
				if tt.header != "" && r.Header.Get(tt.header) != tt.wantHdr {
					t.Fatalf("%s header = %q, want %q", tt.header, r.Header.Get(tt.header), tt.wantHdr)
				}
				writeFixture(t, w, tt.body)
			}))
			defer server.Close()

			tt.item.URL = server.URL
			status := tt.c.Collect(context.Background(), tt.item, config.Proxy{})

			if status.State != "online" {
				t.Fatalf("state = %q, want online; detail=%s", status.State, status.Detail)
			}
			if status.Label != tt.want {
				t.Fatalf("label = %q, want %q", status.Label, tt.want)
			}
		})
	}
}

func TestNextcloudMaintenanceState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeFixture(t, w, "testdata/nextcloud_status_maintenance.json")
	}))
	defer server.Close()

	status := Nextcloud{}.Collect(context.Background(), config.Item{URL: server.URL}, config.Proxy{})

	if status.State != "maintenance" || status.Tone != "warning" {
		t.Fatalf("status = %#v, want maintenance warning", status)
	}
	if status.Indicator != "maintenance" {
		t.Fatalf("indicator = %q, want maintenance", status.Indicator)
	}
}

func TestImmichCollectsStatisticBadges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/server/statistics" {
			t.Fatalf("path = %q, want /api/server/statistics", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "immich-token" {
			t.Fatalf("x-api-key header = %q, want immich-token", got)
		}
		writeFixture(t, w, "testdata/immich_statistics.json")
	}))
	defer server.Close()

	status := Immich{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"apikey": "immich-token"},
	}, config.Proxy{})

	assertBadge(t, status, "Users", "3", "success")
	assertBadge(t, status, "Photos", "12847", "info")
	assertBadge(t, status, "Videos", "1523", "warning")
	assertBadge(t, status, "Usage", "231.51 GiB", "danger")
}

func TestDockerSocketProxyCollectsContainerBadges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/containers/json?all=true" {
			t.Fatalf("request URI = %q, want container list", r.URL.RequestURI())
		}
		writeFixture(t, w, "testdata/dockersocketproxy_containers.json")
	}))
	defer server.Close()

	status := DockerSocketProxy{}.Collect(context.Background(), config.Item{URL: server.URL}, config.Proxy{})

	assertBadge(t, status, "Running", "2", "info")
	assertBadge(t, status, "Stopped", "1", "warning")
}

func TestDockerSocketProxySupportsUnixSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix sockets are not available on Windows")
	}
	socketPath := filepath.Join(t.TempDir(), "docker.sock")
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() != "/containers/json?all=true" {
			t.Fatalf("request URI = %q, want container list", r.URL.RequestURI())
		}
		writeFixture(t, w, "testdata/dockersocketproxy_containers.json")
	})}
	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Close()

	status := DockerSocketProxy{}.Collect(context.Background(), config.Item{
		Raw: map[string]any{"socket": socketPath},
	}, config.Proxy{})

	assertBadge(t, status, "Running", "2", "info")
	assertBadge(t, status, "Stopped", "1", "warning")
}

func TestDeviceTotalCollectors(t *testing.T) {
	t.Run("PiAlert", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RequestURI() != "/php/server/devices.php?action=getDevicesTotals" {
				t.Fatalf("request URI = %q, want PiAlert totals", r.URL.RequestURI())
			}
			writeFixture(t, w, "testdata/pialert_totals.json")
		}))
		defer server.Close()

		status := PiAlert{}.Collect(context.Background(), config.Item{URL: server.URL}, config.Proxy{})
		assertBadge(t, status, "Total", "89", "info")
		assertBadge(t, status, "Connected", "82", "success")
		assertBadge(t, status, "New", "15", "warning")
		assertBadge(t, status, "Down", "3", "danger")
	})

	t.Run("NetAlertx", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer net-token" {
				t.Fatalf("Authorization header = %q, want bearer token", got)
			}
			writeFixture(t, w, "testdata/netalertx_totals.json")
		}))
		defer server.Close()

		status := NetAlertx{}.Collect(context.Background(), config.Item{
			URL: server.URL,
			Raw: map[string]any{"apikey": "net-token"},
		}, config.Proxy{})
		assertBadge(t, status, "Total", "45", "info")
		assertBadge(t, status, "Connected", "38", "success")
		assertBadge(t, status, "New", "2", "warning")
		assertBadge(t, status, "Down", "3", "danger")
	})
}

func TestSpeedtestPeanutGatusAndMylar(t *testing.T) {
	t.Run("SpeedtestTracker", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/results/latest" {
				t.Fatalf("path = %q, want current Speedtest Tracker API", r.URL.Path)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer speed-token" {
				t.Fatalf("Authorization = %q, want bearer token", got)
			}
			if got := r.Header.Get("Accept"); got != "application/json" {
				t.Fatalf("Accept = %q, want application/json", got)
			}
			_, _ = w.Write([]byte(`{"data":{"download":5300000,"upload":4299350,"download_bits":42452234,"upload_bits":34394800,"ping":12.9873}}`))
		}))
		defer server.Close()

		status := SpeedtestTracker{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"apikey": "speed-token"}}, config.Proxy{})
		if status.Label != "Down 42.45 Mbit/s | Up 34.39 Mbit/s | Ping 12.99 ms" {
			t.Fatalf("label = %q, want formatted speedtest", status.Label)
		}
	})

	t.Run("SpeedtestTracker falls back to legacy API", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/results/latest":
				http.NotFound(w, r)
			case "/api/speedtest/latest":
				writeFixture(t, w, "testdata/speedtest_latest.json")
			default:
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
		}))
		defer server.Close()

		status := SpeedtestTracker{}.Collect(context.Background(), config.Item{URL: server.URL}, config.Proxy{})
		if status.Label != "Down 42.45 Mbit/s | Up 34.39 Mbit/s | Ping 12.99 ms" {
			t.Fatalf("label = %q, want formatted legacy speedtest", status.Label)
		}
	})

	t.Run("PeaNUT", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/devices/ups" {
				t.Fatalf("path = %q, want PeaNUT device", r.URL.Path)
			}
			writeFixture(t, w, "testdata/peanut_device.json")
		}))
		defer server.Close()

		status := PeaNUT{}.Collect(context.Background(), config.Item{
			URL: server.URL,
			Raw: map[string]any{"device": "ups"},
		}, config.Proxy{})
		if status.State != "online" || status.Label != "50.0% UPS Load" {
			t.Fatalf("status = %#v, want online load label", status)
		}
	})

	t.Run("Gatus", func(t *testing.T) {
		server := fixtureServer(t, map[string]string{
			"/api/v1/endpoints/statuses": "testdata/gatus_statuses.json",
		})
		defer server.Close()

		status := Gatus{}.Collect(context.Background(), config.Item{
			URL: server.URL,
			Raw: map[string]any{"groups": []any{"Services"}},
		}, config.Proxy{})
		if status.State != "warn" || status.Label != "1/2 up | 15.00 ms avg." || status.Detail != "50%" || status.Indicator != "50%" {
			t.Fatalf("status = %#v, want filtered Gatus summary", status)
		}
	})

	t.Run("Mylar", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.RequestURI() {
			case "/api?cmd=getUpcoming&apikey=mylar-token":
				writeFixture(t, w, "testdata/mylar_upcoming.json")
			case "/api?cmd=getWanted&apikey=mylar-token":
				writeFixture(t, w, "testdata/mylar_wanted.json")
			default:
				t.Fatalf("unexpected request: %s", r.URL.RequestURI())
			}
		}))
		defer server.Close()

		status := Mylar{}.Collect(context.Background(), config.Item{
			URL: server.URL,
			Raw: map[string]any{"apikey": "mylar-token"},
		}, config.Proxy{})
		assertBadge(t, status, "Wanted", "3", "info")
		assertBadge(t, status, "Upcoming", "2", "neutral")
	})
}

func TestNewSimpleCollectors(t *testing.T) {
	t.Run("Emby", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/System/info/public":
				_, _ = w.Write([]byte(`{"Id":"emby"}`))
			case "/items/counts":
				if got := r.Header.Get("X-Emby-Token"); got != "emby-token" {
					t.Fatalf("X-Emby-Token = %q, want emby-token", got)
				}
				_, _ = w.Write([]byte(`{"MovieCount":12,"SeriesCount":3,"EpisodeCount":45,"SongCount":100,"AlbumCount":10}`))
			default:
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
		}))
		defer server.Close()

		status := Emby{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"apikey": "emby-token", "libraryType": "series"}}, config.Proxy{})
		if status.State != "running" || status.Label != "45 eps, 3 series" {
			t.Fatalf("status = %#v, want running series label", status)
		}
	})

	t.Run("Glances", func(t *testing.T) {
		server := fixtureServer(t, map[string]string{"/api/4/quicklook": "testdata/glances_quicklook.json"})
		defer server.Close()

		status := Glances{}.Collect(context.Background(), config.Item{URL: server.URL}, config.Proxy{})
		assertBadge(t, status, "CPU", "23.50%", "info")
		assertBadge(t, status, "Mem", "64.20%", "warning")
	})

	t.Run("Glances filters configured stats", func(t *testing.T) {
		server := fixtureServer(t, map[string]string{"/api/4/quicklook": "testdata/glances_quicklook.json"})
		defer server.Close()

		status := Glances{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"stats": []any{"cpu"}}}, config.Proxy{})
		assertBadge(t, status, "CPU", "23.50%", "info")
		if len(status.Badges) != 1 {
			t.Fatalf("badges = %#v, want only configured CPU badge", status.Badges)
		}
	})

	t.Run("HyperHDR", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/json-rpc" {
				t.Fatalf("path = %q, want /json-rpc", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"info":{"currentInstance":1,"instance":[{"instance":1,"friendly_name":"Living Room","running":true},{"instance":2,"friendly_name":"Desk","running":false}]}}`))
		}))
		defer server.Close()

		status := HyperHDR{}.Collect(context.Background(), config.Item{URL: server.URL}, config.Proxy{})
		if status.Label != "Current instance: Living Room" {
			t.Fatalf("label = %q, want current instance", status.Label)
		}
		assertBadge(t, status, "Running", "1", "success")
		assertBadge(t, status, "Stopped", "1", "danger")
	})
}

func TestHomeAssistantAndOpenHABCollectors(t *testing.T) {
	t.Run("HomeAssistant", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Bearer ha-token" {
				t.Fatalf("Authorization = %q, want bearer token", got)
			}
			switch r.URL.Path {
			case "/api/":
				_, _ = w.Write([]byte(`{"message":"API running."}`))
			case "/api/config":
				_, _ = w.Write([]byte(`{"version":"2026.4.1","location_name":"Home"}`))
			case "/api/states":
				_, _ = w.Write([]byte(`[{}, {}, {}]`))
			default:
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
		}))
		defer server.Close()

		status := HomeAssistant{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"apikey": "ha-token", "items": []any{"name", "version", "entities"}, "separator": " | "}}, config.Proxy{})
		if status.Label != "Home | v2026.4.1 | 3 entities" {
			t.Fatalf("label = %q, want Home Assistant details", status.Label)
		}
	})

	t.Run("OpenHAB", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "Basic b2hhYi10b2tlbjo=" {
				t.Fatalf("Authorization = %q, want basic token", got)
			}
			switch r.URL.RequestURI() {
			case "/rest/systeminfo":
				_, _ = w.Write([]byte(`{"systemInfo":{"configFolder":"/openhab/conf"}}`))
			case "/rest/things?summary=true":
				_, _ = w.Write([]byte(`[{"statusInfo":{"status":"ONLINE"}},{"statusInfo":{"status":"OFFLINE"}}]`))
			case "/rest/items":
				_, _ = w.Write([]byte(`[{}, {}, {}]`))
			default:
				t.Fatalf("unexpected request: %s", r.URL.RequestURI())
			}
		}))
		defer server.Close()

		status := OpenHAB{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"apikey": "ohab-token", "things": true, "items": true}}, config.Proxy{})
		if status.Label != "2 things (1 Online), 3 items" {
			t.Fatalf("label = %q, want OpenHAB details", status.Label)
		}
	})
}

func TestDownloadCollectors(t *testing.T) {
	t.Run("SABnzbd", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RequestURI() != "/api?output=json&apikey=sab-token&mode=queue" {
				t.Fatalf("request URI = %q, want SABnzbd queue", r.URL.RequestURI())
			}
			_, _ = w.Write([]byte(`{"queue":{"noofslots":2,"speed":"1.5 MB/s"}}`))
		}))
		defer server.Close()

		status := SABnzbd{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"apikey": "sab-token"}}, config.Proxy{})
		if status.Label != "Down 1.50 MB/s" {
			t.Fatalf("label = %q, want download speed", status.Label)
		}
		assertBadge(t, status, "Downloads", "2", "info")
	})

	t.Run("qBittorrent", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v2/auth/login":
				if r.Method != http.MethodPost {
					t.Fatalf("method = %s, want POST", r.Method)
				}
				if got := r.Header.Get("Content-Type"); got != "application/x-www-form-urlencoded" {
					t.Fatalf("Content-Type = %q, want form encoding", got)
				}
				if err := r.ParseForm(); err != nil {
					t.Fatal(err)
				}
				if r.Form.Get("username") != "qb-user" || r.Form.Get("password") != "qb-pass" {
					t.Fatalf("login form = %#v, want username/password", r.Form)
				}
				http.SetCookie(w, &http.Cookie{Name: "SID", Value: "qb-session"})
				_, _ = w.Write([]byte("Ok."))
			case "/api/v2/torrents/info":
				if got := r.Header.Get("Cookie"); got != "SID=qb-session" {
					t.Fatalf("Cookie = %q, want qBittorrent SID", got)
				}
				_, _ = w.Write([]byte(`[{}, {}, {}]`))
			case "/api/v2/transfer/info":
				if got := r.Header.Get("Cookie"); got != "SID=qb-session" {
					t.Fatalf("Cookie = %q, want qBittorrent SID", got)
				}
				_, _ = w.Write([]byte(`{"dl_info_speed":2048,"up_info_speed":1024}`))
			default:
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
		}))
		defer server.Close()

		status := QBittorrent{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"username": "qb-user", "password": "qb-pass"}}, config.Proxy{})
		assertBadge(t, status, "Torrents", "3", "info")
	})

	t.Run("Tdarr", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/api/v2/cruddb" {
				t.Fatalf("request = %s %s, want POST cruddb", r.Method, r.URL.Path)
			}
			if got := r.Header.Get("x-api-key"); got != "tdarr-token" {
				t.Fatalf("x-api-key header = %q, want tdarr-token", got)
			}
			_, _ = w.Write([]byte(`{"table1Count":4,"table6Count":1}`))
		}))
		defer server.Close()

		status := Tdarr{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"apikey": "tdarr-token"}}, config.Proxy{})
		assertBadge(t, status, "Queue", "4", "info")
		assertBadge(t, status, "Errored", "1", "danger")
	})
}

func TestMinifluxCounterStyleUsesBadge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/feeds/counters" {
			t.Fatalf("path = %q, want /v1/feeds/counters", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"unreads":{"1":3,"2":5}}`))
	}))
	defer server.Close()

	status := Miniflux{}.Collect(context.Background(), config.Item{
		URL: server.URL,
		Raw: map[string]any{"style": "counter"},
	}, config.Proxy{})

	if status.Indicator != "" {
		t.Fatalf("indicator = %q, want no status indicator for counter style", status.Indicator)
	}
	assertBadge(t, status, "Unread", "8", "info")
}

func TestFreshRSSCollector(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.RequestURI() {
		case "/api/greader.php/accounts/ClientLogin?Email=user%2Bname%40example.com&Passwd=p%26a%3Dss":
			_, _ = w.Write([]byte("SID=x\r\nLSID=y\r\nAuth=fresh-token\r\n"))
		case "/api/greader.php/reader/api/0/subscription/list?output=json":
			if got := r.Header.Get("Authorization"); got != "GoogleLogin auth=fresh-token" {
				t.Fatalf("Authorization = %q, want FreshRSS auth", got)
			}
			_, _ = w.Write([]byte(`{"subscriptions":[{},{}]}`))
		case "/api/greader.php/reader/api/0/unread-count?output=json":
			_, _ = w.Write([]byte(`{"max":"1000","unreadcounts":[{"id":"feed/1","count":3},{"id":"user/-/label/News","count":3},{"id":"feed/2","count":6},{"id":"user/-/state/com.google/reading-list","count":9}]}`))
		default:
			t.Fatalf("unexpected request: %s", r.URL.RequestURI())
		}
	}))
	defer server.Close()

	status := FreshRSS{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"username": "user+name@example.com", "password": "p&a=ss"}}, config.Proxy{})
	assertBadge(t, status, "Subscriptions", "2", "info")
	assertBadge(t, status, "Unread", "9", "warning")
}

func TestArrCollectors(t *testing.T) {
	tests := []struct {
		name string
		c    Collector
	}{
		{name: "Lidarr", c: Lidarr{}},
		{name: "Readarr", c: Readarr{}},
		{name: "Sonarr", c: Sonarr{}},
		{name: "Radarr", c: Radarr{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.RequestURI() {
				case "/api/v3/wanted/missing?pageSize=1&apikey=arr-token":
					_, _ = w.Write([]byte(`{"totalRecords":3}`))
					return
				case "/api/v3/wanted/missing?pageSize=3&apikey=arr-token":
					_, _ = w.Write([]byte(`{"records":[{"monitored":true,"isAvailable":true,"hasFile":false},{"monitored":true,"isAvailable":false,"hasFile":false},{"monitored":false,"isAvailable":true,"hasFile":false}]}`))
					return
				}
				switch r.URL.Path {
				case "/api/v1/health", "/api/v3/health":
					_, _ = w.Write([]byte(`[{"type":"warning"},{"type":"error"}]`))
				case "/api/v1/queue", "/api/v1/queue/status", "/api/v3/queue":
					_, _ = w.Write([]byte(`{"totalRecords":2,"totalCount":2}`))
				case "/api/v1/wanted/missing", "/api/v3/wanted/missing":
					_, _ = w.Write([]byte(`{"totalRecords":3,"records":[{"monitored":true,"hasFile":false}]}`))
				case "/api/v3/queue/details":
					_, _ = w.Write([]byte(`[{"trackedDownloadStatus":"warning"}]`))
				default:
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
			}))
			defer server.Close()

			status := tt.c.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"apikey": "arr-token"}}, config.Proxy{})
			assertBadge(t, status, "Activity", "2", "info")
			wantMissing := "3"
			if tt.name == "Radarr" {
				wantMissing = "1"
			}
			assertBadge(t, status, "Missing", wantMissing, "neutral")
			wantWarnings := "1"
			if tt.name == "Radarr" {
				wantWarnings = "2"
			}
			assertBadge(t, status, "Warning", wantWarnings, "warning")
			assertBadge(t, status, "Error", "1", "danger")
		})
	}
}

func TestUptimeKumaPortainerAndProxmoxCollectors(t *testing.T) {
	t.Run("UptimeKuma", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/status-page/default":
				_, _ = w.Write([]byte(`{"incident":null}`))
			case "/api/status-page/heartbeat/default":
				_, _ = w.Write([]byte(`{"heartbeatList":{"1":[{"status":1}],"2":[{"status":0}]},"uptimeList":{"1":0.99,"2":0.95}}`))
			default:
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
		}))
		defer server.Close()

		status := UptimeKuma{}.Collect(context.Background(), config.Item{URL: server.URL}, config.Proxy{})
		if status.State != "warn" || status.Indicator != "97.0%" {
			t.Fatalf("status = %#v, want warn 97.0%%", status)
		}
		if status.URL != server.URL+"/status/default" {
			t.Fatalf("URL = %q, want status page URL", status.URL)
		}
	})

	t.Run("UptimeKuma incident title", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/status-page/default":
				_, _ = w.Write([]byte(`{"incident":{"title":"Database maintenance"}}`))
			case "/api/status-page/heartbeat/default":
				_, _ = w.Write([]byte(`{"heartbeatList":{"1":[{"status":1}]},"uptimeList":{"1":0.99}}`))
			default:
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
		}))
		defer server.Close()

		status := UptimeKuma{}.Collect(context.Background(), config.Item{URL: server.URL}, config.Proxy{})
		if status.State != "bad" || status.Label != "Database maintenance" {
			t.Fatalf("status = %#v, want incident title", status)
		}
	})

	t.Run("Portainer", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("X-Api-Key"); got != "port-token" {
				t.Fatalf("X-Api-Key = %q, want port-token", got)
			}
			switch r.URL.RequestURI() {
			case "/api/endpoints":
				_, _ = w.Write([]byte(`[{"Id":1,"Name":"prod"}]`))
			case "/api/endpoints/1/docker/containers/json?all=1":
				_, _ = w.Write([]byte(`[{"State":"running"},{"State":"dead"},{"State":"exited"}]`))
			case "/api/status":
				_, _ = w.Write([]byte(`{"Version":"2.20.0"}`))
			default:
				t.Fatalf("unexpected request: %s", r.URL.RequestURI())
			}
		}))
		defer server.Close()

		status := Portainer{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"apikey": "port-token"}}, config.Proxy{})
		if status.Label != "Version 2.20.0" {
			t.Fatalf("label = %q, want version", status.Label)
		}
		assertBadge(t, status, "Running", "1", "success")
		assertBadge(t, status, "Dead", "1", "danger")
		assertBadge(t, status, "Other", "1", "info")
	})

	t.Run("Portainer status endpoint controls online state", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.RequestURI() {
			case "/api/status":
				http.Error(w, "unavailable", http.StatusServiceUnavailable)
			case "/api/endpoints":
				_, _ = w.Write([]byte(`[{"Id":1,"Name":"prod"}]`))
			case "/api/endpoints/1/docker/containers/json?all=1":
				_, _ = w.Write([]byte(`[{"State":"running"}]`))
			default:
				t.Fatalf("unexpected request: %s", r.URL.RequestURI())
			}
		}))
		defer server.Close()

		status := Portainer{}.Collect(context.Background(), config.Item{URL: server.URL}, config.Proxy{})
		if status.State != "offline" {
			t.Fatalf("status = %#v, want offline when /api/status fails", status)
		}
	})

	t.Run("Proxmox", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "PVEAPIToken=root@pam!homer=secret" {
				t.Fatalf("Authorization = %q, want proxmox token", got)
			}
			switch r.URL.Path {
			case "/api2/json/nodes/node1/status":
				_, _ = w.Write([]byte(`{"data":{"memory":{"used":512,"total":1024},"rootfs":{"used":25,"total":100},"cpu":0.125}}`))
			case "/api2/json/nodes/node1/qemu":
				_, _ = w.Write([]byte(`{"data":[{"status":"running"},{"status":"stopped"}]}`))
			case "/api2/json/nodes/node1/lxc":
				_, _ = w.Write([]byte(`{"data":[{"status":"running"}]}`))
			default:
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
		}))
		defer server.Close()

		status := Proxmox{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{"api_token": "PVEAPIToken=root@pam!homer=secret", "node": "node1"}}, config.Proxy{})
		assertBadge(t, status, "Disk", "25.0%", "info")
		assertBadge(t, status, "Mem", "50.0%", "info")
		assertBadge(t, status, "VMs", "1/2", "info")
	})

	t.Run("Proxmox thresholds and hidden totals", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api2/json/nodes/node1/status":
				_, _ = w.Write([]byte(`{"data":{"memory":{"used":75,"total":100},"rootfs":{"used":90,"total":100},"cpu":0.45}}`))
			case "/api2/json/nodes/node1/qemu":
				_, _ = w.Write([]byte(`{"data":[{"status":"running"},{"status":"stopped"}]}`))
			case "/api2/json/nodes/node1/lxc":
				_, _ = w.Write([]byte(`{"data":[{"status":"running"}]}`))
			default:
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
		}))
		defer server.Close()

		status := Proxmox{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{
			"node":          "node1",
			"warning_value": 50,
			"danger_value":  80,
			"hide":          []any{"vms_total", "lxcs_total"},
		}}, config.Proxy{})
		assertBadge(t, status, "Disk", "90.0%", "danger")
		assertBadge(t, status, "Mem", "75.0%", "warning")
		assertBadge(t, status, "VMs", "1", "info")
		assertBadge(t, status, "LXCs", "1", "neutral")
	})

	t.Run("Proxmox builds token header from split fields", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if got := r.Header.Get("Authorization"); got != "PVEAPIToken=root@pam!homer=secret" {
				t.Fatalf("Authorization = %q, want split proxmox token", got)
			}
			switch r.URL.Path {
			case "/api2/json/nodes/node1/status":
				_, _ = w.Write([]byte(`{"data":{"memory":{"used":1,"total":2},"rootfs":{"used":1,"total":4},"cpu":0.25}}`))
			case "/api2/json/nodes/node1/qemu":
				_, _ = w.Write([]byte(`{"data":[]}`))
			case "/api2/json/nodes/node1/lxc":
				_, _ = w.Write([]byte(`{"data":[]}`))
			default:
				t.Fatalf("unexpected path: %s", r.URL.Path)
			}
		}))
		defer server.Close()

		status := Proxmox{}.Collect(context.Background(), config.Item{URL: server.URL, Raw: map[string]any{
			"node":             "node1",
			"api_token_id":     "root@pam!homer",
			"api_token_secret": "secret",
		}}, config.Proxy{})
		assertBadge(t, status, "CPU", "25.0%", "info")
	})
}

func fixtureServer(t *testing.T, fixtures map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fixture, ok := fixtures[r.URL.Path]
		if !ok {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeFixture(t, w, fixture)
	}))
}

func writeFixture(t *testing.T, w http.ResponseWriter, path string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(body); err != nil {
		t.Fatal(err)
	}
}

func assertBadge(t *testing.T, status Status, label string, value string, tone string) {
	t.Helper()
	for _, badge := range status.Badges {
		if badge.Label == label {
			if badge.Value != value || badge.Tone != tone {
				t.Fatalf("%s badge = %#v, want value=%q tone=%q", label, badge, value, tone)
			}
			return
		}
	}
	t.Fatalf("missing %s badge in %#v", label, status.Badges)
}

func TestPingItemHeadersOverrideProxyHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Fatalf("Authorization header = %q, want no global header", got)
		}
		if got := r.Header.Get("X-Item"); got != "item" {
			t.Fatalf("X-Item header = %q, want item header", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	registry := NewRegistry()
	registry.Register(Ping{})
	cfg := config.Config{
		Proxy: config.Proxy{Headers: map[string]string{"Authorization": "Bearer global"}},
		Services: []config.Group{{
			Items: []config.Item{{
				Name:    "App",
				Type:    "Ping",
				URL:     server.URL,
				Headers: map[string]string{"X-Item": "item"},
				Raw:     map[string]any{"headers": map[string]any{"X-Item": "item"}},
			}},
		}},
	}

	statuses := registry.Collect(context.Background(), cfg, time.Second)

	if got := statuses[Key(0, 0)].State; got != "online" {
		t.Fatalf("status = %q, want online", got)
	}
}

func TestCollectItemAppliesProgrammaticItemHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Item"); got != "item" {
			t.Fatalf("X-Item header = %q, want item header", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	registry := NewRegistry()
	registry.Register(Ping{})
	status, ok := registry.CollectItem(context.Background(), config.Item{
		Name:    "App",
		Type:    "Ping",
		URL:     server.URL,
		Headers: map[string]string{"X-Item": "item"},
	}, time.Second)

	if !ok {
		t.Fatal("CollectItem() ok = false, want true")
	}
	if status.State != "online" {
		t.Fatalf("status = %q, want online", status.State)
	}
}

func TestRegistryReportsUnsupportedCollectors(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Ping{})

	cfg := config.Config{
		Services: []config.Group{
			{
				Name: "Apps",
				Items: []config.Item{
					{Name: "Plain Link"},
					{Name: "Generic Link", Type: "Generic"},
					{Name: "Known", Type: "Ping"},
					{Name: "Unknown", Type: "Plex"},
				},
			},
		},
	}

	unsupported := registry.UnsupportedCollectors(cfg)

	if len(unsupported) != 1 {
		t.Fatalf("unsupported = %#v, want one unsupported collector", unsupported)
	}
	got := unsupported[0]
	if got.Type != "Plex" || got.ItemName != "Unknown" || got.GroupName != "Apps" || got.GroupIndex != 0 || got.ItemIndex != 3 {
		t.Fatalf("unsupported[0] = %#v, want Plex in Apps item 3", got)
	}
}
