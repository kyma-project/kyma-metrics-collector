package edp

import (
	"context"
	"encoding/json"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"

	"fmt"
	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/process"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"net/http"
	"time"
)

type Collector struct {
	EDPClient    *Client
	runtimeID    string
	subAccountID string
	shootName    string
	scanners     []resource.Scanner
	logger       *zap.Logger
}

var _ collector.CollectorSender = &Collector{}

func NewCollector(EDPClient *Client, runtimeID, subAccountID, shootName string, scanner ...resource.Scanner) collector.CollectorSender {
	return &Collector{
		EDPClient:    EDPClient,
		runtimeID:    runtimeID,
		subAccountID: subAccountID,
		shootName:    shootName,
		scanners:     scanner,
	}
}

func (c *Collector) CollectAndSend(ctx context.Context, config *rest.Config, specs *process.PublicCloudSpecs, previousScans collector.ScanMap) (collector.ScanMap, error) {
	currentTimestamp := getTimestampNow()

	scans := c.executeScans(ctx, config, previousScans)
	EDPMeasurements := c.convertScansToEDPMeasurements(specs, scans, previousScans)
	payload := NewPayload(
		c.runtimeID,
		c.subAccountID,
		c.shootName,
		currentTimestamp,
		EDPMeasurements,
	)
	err := c.sendPayload(payload)

	return scans, err
}

func (c *Collector) executeScans(ctx context.Context, config *rest.Config, previousScans collector.ScanMap) collector.ScanMap {
	scans := make(collector.ScanMap)

	for _, s := range c.scanners {
		scan, err := s.Scan(ctx, config)
		if err != nil {
			c.logger.Error("error scanning", zap.Error(err), zap.String("scanner ID", string(s.ID())))
			// use previous scan
			previousScan, exists := previousScans[s.ID()]
			if !exists {
				c.logger.Error("no previous scan found", zap.String("scanner ID", string(s.ID())))
				continue
			}
			scan = previousScan
		}
		// use new or old measure
		scans[s.ID()] = scan
	}

	return scans
}

func (c *Collector) convertScansToEDPMeasurements(specs *process.PublicCloudSpecs, currentScans collector.ScanMap, previousScans collector.ScanMap) []resource.EDPMeasurement {
	EDPMeasurements := []resource.EDPMeasurement{}

	for scannerID, scan := range currentScans {
		edp, err := scan.EDP(specs)
		if err != nil {
			c.logger.Error("error converting scan to EDP measurements", zap.Error(err), zap.String("scanner", string(scannerID)))
			// attempt to get the previous scan and convert it to EDP measurement
			previousScan, exists := previousScans[scannerID]
			if !exists {
				c.logger.Error("no previous scan found", zap.String("scanner", string(scannerID)))
				continue
			}
			edp, err = previousScan.EDP(specs)
			if err != nil {
				c.logger.Error("error converting previous scan to EDP measurements", zap.Error(err), zap.String("scanner", string(scannerID)))
				continue
			}
		}

		EDPMeasurements = append(EDPMeasurements, edp)
	}

	return EDPMeasurements
}

// sendPayload sends the payload to the EDP backend.
func (c *Collector) sendPayload(payload Payload) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload for subAccountID (%s): %w", c.subAccountID, err)
	}

	edpRequest, err := c.EDPClient.NewRequest(c.subAccountID)
	if err != nil {
		return fmt.Errorf("failed to create a new request for EDP for subAccountID (%s): %w", c.subAccountID, err)
	}

	resp, err := c.EDPClient.Send(edpRequest, payloadJSON)
	if err != nil {
		return fmt.Errorf("failed to send payload to EDP for subAccountID (%s): %w", c.subAccountID, err)
	}

	if !isSuccess(resp.StatusCode) {
		return fmt.Errorf("failed to send payload to EDP for subAccountID (%s) as it returned HTTP status code %d", c.subAccountID, resp.StatusCode)
	}

	return nil
}

// getTimestampNow returns the time now in the format of RFC3339.
func getTimestampNow() string {
	return time.Now().Format(time.RFC3339)
}

func isSuccess(status int) bool {
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		return true
	}

	return false
}
