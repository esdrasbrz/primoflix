package ddns

import (
	"time"

	"github.com/esdrasbrz/primodns/metrics"
	"github.com/esdrasbrz/primodns/services/cloudflare"
	externalip "github.com/glendc/go-external-ip"
	"go.uber.org/zap"
)

type DDNS struct {
	consensus  *externalip.Consensus
	cloudflare cloudflare.CloudflareService
	logger     *zap.Logger
	lastIP     string
	updatedAt  int64
}

func New(logger *zap.Logger, cloudflare cloudflare.CloudflareService) *DDNS {
	consensus := externalip.DefaultConsensus(nil, nil)

	return &DDNS{
		consensus:  consensus,
		cloudflare: cloudflare,
		logger:     logger,
	}
}

func (d *DDNS) update() {
	ip, err := d.consensus.ExternalIP()

	if err != nil {
		d.logger.Error("error while fetching external IP", zap.Error(err))
		metrics.ExternalIPRequests.WithLabelValues("error").Inc()
		return
	}
	metrics.ExternalIPRequests.WithLabelValues("success").Inc()

	// check if the IP is the same as before, then do nothing
	if ip.String() == d.lastIP {
		return
	}

	err = d.cloudflare.UpdateDomains(ip.String())

	if err != nil {
		d.logger.Error("error while updating Cloudflare", zap.Error(err))
		return
	}

	metrics.LastUpdatedAt.DeleteLabelValues(d.lastIP)
	d.lastIP = ip.String()
	d.updatedAt = time.Now().UTC().Unix()
	metrics.LastUpdatedAt.WithLabelValues(d.lastIP).Set(float64(d.updatedAt))

	d.logger.Info("Domains updated", zap.String("ip", ip.String()))
}

func (d *DDNS) RunDDNSUpdater() {
	for {
		d.update()

		time.Sleep(15 * time.Second)
	}
}
