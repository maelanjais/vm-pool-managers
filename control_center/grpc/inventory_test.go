package grpc

import (
	"control_center/models"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"
)

func TestProbeAppPort_Skipped(t *testing.T) {
	cases := []struct {
		name string
		vm   models.VMInstance
	}{
		{"not idle", models.VMInstance{ActivityStatus: "active", AppPort: 8888, IP: "1.2.3.4"}},
		{"no port", models.VMInstance{ActivityStatus: "idle", AppPort: 0, IP: "1.2.3.4"}},
		{"no ip", models.VMInstance{ActivityStatus: "idle", AppPort: 8888, IP: ""}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vm := c.vm
			probeAppPort(&vm)
			if vm.ActivityStatus != c.vm.ActivityStatus {
				t.Errorf("status changed unexpectedly: %q → %q", c.vm.ActivityStatus, vm.ActivityStatus)
			}
		})
	}
}

func TestProbeAppPort_Unreachable(t *testing.T) {
	// Port 1 is typically not listening anywhere
	vm := models.VMInstance{ActivityStatus: "idle", AppPort: 1, IP: "127.0.0.1"}
	probeAppPort(&vm)
	if vm.ActivityStatus != "idle" {
		t.Errorf("expected idle for unreachable port, got %q", vm.ActivityStatus)
	}
}

func TestProbeAppPort_Reachable(t *testing.T) {
	// probeAppPort interroge /api/status de Jupyter : actif si connections>0 ou kernels>0.
	// On simule un Jupyter avec une connexion ouverte.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/status" {
			_, _ = w.Write([]byte(`{"connections":1,"kernels":1}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	vm := models.VMInstance{ActivityStatus: "idle", AppPort: port, IP: u.Hostname()}
	probeAppPort(&vm)
	if vm.ActivityStatus != "active" {
		t.Errorf("attendu 'active' pour un Jupyter avec connexion, obtenu %q", vm.ActivityStatus)
	}
}

func TestProbeAppPort_ReachableButIdle(t *testing.T) {
	// Jupyter joignable mais aucune connexion/kernel => reste 'idle'.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"connections":0,"kernels":0}`))
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	vm := models.VMInstance{ActivityStatus: "idle", AppPort: port, IP: u.Hostname()}
	probeAppPort(&vm)
	if vm.ActivityStatus != "idle" {
		t.Errorf("attendu 'idle' sans connexion, obtenu %q", vm.ActivityStatus)
	}
}

func TestMergeInventoryVM_Defaults(t *testing.T) {
	srv := models.Server{
		Name:      "test-server",
		Status:    "ready",
		IP_Address: "10.0.0.1",
	}
	reg := models.VMInstance{}
	vm := mergeInventoryVM(srv, reg)
	if vm.ActivityStatus != "idle" {
		t.Errorf("expected idle, got %q", vm.ActivityStatus)
	}
	if vm.IP != "10.0.0.1" {
		t.Errorf("expected IP 10.0.0.1, got %q", vm.IP)
	}
}

func TestMergeInventoryVM_CacheOverride(t *testing.T) {
	srv := models.Server{Name: "cached-vm", Status: "ready", IP_Address: "10.0.0.2"}
	vmActivityCache.Store("cached-vm", "active")
	defer vmActivityCache.Delete("cached-vm")

	reg := models.VMInstance{}
	vm := mergeInventoryVM(srv, reg)
	if vm.ActivityStatus != "active" {
		t.Errorf("expected active from cache, got %q", vm.ActivityStatus)
	}
}

func TestMergeInventoryVM_RegistrarOverrides(t *testing.T) {
	srv := models.Server{Name: "reg-vm", Status: "starting", IP_Address: "10.0.0.3"}
	reg := models.VMInstance{
		Name:   "reg-vm",
		Status: "ready",
		IP:     "10.0.0.5",
		LastSeen: time.Now(),
	}
	vm := mergeInventoryVM(srv, reg)
	if vm.Status != "ready" {
		t.Errorf("expected registrar status 'ready', got %q", vm.Status)
	}
	if vm.IP != "10.0.0.5" {
		t.Errorf("expected registrar IP 10.0.0.5, got %q", vm.IP)
	}
}

func TestServerPoolID_Fallbacks(t *testing.T) {
	// From ServerpoolID field
	srv := models.Server{ServerpoolID: "pool1"}
	if got := serverPoolID(srv); got != "pool1" {
		t.Errorf("expected pool1, got %q", got)
	}
	// From metadata
	srv2 := models.Server{Metadata: map[string]string{"serverpool_id": "pool2"}}
	if got := serverPoolID(srv2); got != "pool2" {
		t.Errorf("expected pool2, got %q", got)
	}
	// Empty
	srv3 := models.Server{}
	if got := serverPoolID(srv3); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}
