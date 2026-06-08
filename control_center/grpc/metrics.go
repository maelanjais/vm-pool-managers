package grpc

import (
	"control_center/config"
	"control_center/models"

	"github.com/prometheus/client_golang/prometheus"
)

// cpmCollector exposes usage metrics by querying PostgreSQL on each scrape.
// Prometheus turns these gauges into time series (peak hours, occupancy, …).
type cpmCollector struct {
	pools      *prometheus.Desc
	servers    *prometheus.Desc
	vmsActive  *prometheus.Desc
	students   *prometheus.Desc
	ghSessions *prometheus.Desc
	poolUsage  *prometheus.Desc
}

func newCPMCollector() *cpmCollector {
	return &cpmCollector{
		pools:      prometheus.NewDesc("cpm_pools_total", "Nombre de serverpools.", nil, nil),
		servers:    prometheus.NewDesc("cpm_servers", "Nombre de VMs par statut.", []string{"status"}, nil),
		vmsActive:  prometheus.NewDesc("cpm_vms_active", "VMs avec activité récente (étudiant connecté).", nil, nil),
		students:   prometheus.NewDesc("cpm_students_total", "Nombre d'étudiants enregistrés.", nil, nil),
		ghSessions: prometheus.NewDesc("cpm_github_sessions_24h", "Connexions GitHub sur les dernières 24 h.", nil, nil),
		poolUsage:  prometheus.NewDesc("cpm_pool_students", "Étudiants rattachés, par pool.", []string{"pool", "owner"}, nil),
	}
}

func (c *cpmCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.pools
	ch <- c.servers
	ch <- c.vmsActive
	ch <- c.students
	ch <- c.ghSessions
	ch <- c.poolUsage
}

func (c *cpmCollector) Collect(ch chan<- prometheus.Metric) {
	db := config.Database
	if db == nil {
		return
	}

	var n int64
	db.Model(&models.Serverpool{}).Count(&n)
	ch <- prometheus.MustNewConstMetric(c.pools, prometheus.GaugeValue, float64(n))

	type statusCount struct {
		Status string
		C      int64
	}
	var rows []statusCount
	db.Model(&models.Server{}).
		Select("COALESCE(NULLIF(status,''),'unknown') as status, count(*) as c").
		Group("status").Scan(&rows)
	for _, r := range rows {
		ch <- prometheus.MustNewConstMetric(c.servers, prometheus.GaugeValue, float64(r.C), r.Status)
	}

	var active int64
	db.Model(&models.VMInstance{}).
		Where("activity_status <> 'idle' AND last_seen > now() - interval '10 minutes'").
		Count(&active)
	ch <- prometheus.MustNewConstMetric(c.vmsActive, prometheus.GaugeValue, float64(active))

	db.Model(&models.Student{}).Count(&n)
	ch <- prometheus.MustNewConstMetric(c.students, prometheus.GaugeValue, float64(n))

	var gh int64
	db.Model(&models.GitHubSession{}).Where("created_at > now() - interval '24 hours'").Count(&gh)
	ch <- prometheus.MustNewConstMetric(c.ghSessions, prometheus.GaugeValue, float64(gh))

	type poolCount struct {
		Pool  string
		Owner string
		C     int64
	}
	var pcs []poolCount
	db.Raw(`SELECT sp.serverpool_id AS pool, sp.user_id AS owner, count(st.id) AS c
	        FROM serverpools sp
	        LEFT JOIN list_students ls ON ls.pool_id = sp.id
	        LEFT JOIN students st ON st.list_id = ls.id
	        WHERE sp.serverpool_id <> ''
	        GROUP BY sp.serverpool_id, sp.user_id`).Scan(&pcs)
	for _, p := range pcs {
		ch <- prometheus.MustNewConstMetric(c.poolUsage, prometheus.GaugeValue, float64(p.C), p.Pool, p.Owner)
	}
}

// registerMetrics enregistre le collecteur (idempotent en cas de double appel).
func registerMetrics() {
	_ = prometheus.Register(newCPMCollector())
}
