package edp

import (
	"context"
	"encoding/json"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"fmt"
	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	"net/http"
)

type Collector struct {
	EDPClient *Client
	scanners  []resource.Scanner
	logger    *zap.Logger
}

var _ collector.CollectorSender = &Collector{}

func NewCollector(EDPClient *Client, runtimeID, subAccountID, shootName string, scanner ...resource.Scanner) collector.CollectorSender {
	return &Collector{
		EDPClient: EDPClient,
		scanners:  scanner,
	}
}

func (c *Collector) CollectAndSend(ctx context.Context, runtime *runtime.Info, previousScans collector.ScanMap) (collector.ScanMap, error) {
	childCtx, span := otel.Tracer("").Start(ctx, "collect",
		trace.WithAttributes(
			attribute.String("provider", runtime.ProviderType),
			attribute.String("shoot_id", runtime.ShootInfo.ShootName),
		),
	)
	defer span.End()

	currentTimestamp := getTimestampNow()
	scans := c.executeScans(childCtx, runtime)
	EDPMeasurements := c.convertScansToEDPMeasurements(scans, previousScans)
	payload := NewPayload(
		runtime.ShootInfo.RuntimeID,
		runtime.ShootInfo.SubAccountID,
		runtime.ShootInfo.ShootName,
		currentTimestamp,
		EDPMeasurements,
	)
	err := c.sendPayload(payload, runtime.ShootInfo.SubAccountID)

	return scans, err
}

func (c *Collector) executeScans(childCtx context.Context, runtime *runtime.Info) collector.ScanMap {
	scans := make(collector.ScanMap)

	for _, s := range c.scanners {
		scan, err := s.Scan(childCtx, runtime)
		if err != nil {
			c.logger.Error("error scanning", zap.Error(err), zap.String("scanner ID", string(s.ID())))
			continue
		}
		// store only successful scans in the scan map
		scans[s.ID()] = scan
	}

	return scans
}

func (c *Collector) convertScansToEDPMeasurements(currentScans collector.ScanMap, previousScans collector.ScanMap) []resource.EDPMeasurement {
	EDPMeasurements := []resource.EDPMeasurement{}

	for _, s := range c.scanners {
		scan, currentScanExists := currentScans[s.ID()]
		// if current scan doesn't exist (because of a failure during execution), attempt to use the previous scan
		if !currentScanExists {
			previousScan, previousScanExists := previousScans[s.ID()]
			if !previousScanExists {
				c.logger.Error("no previous scan found", zap.String("scanner", string(s.ID())))
				continue
			}
			currentScans[s.ID()] = previousScan
			scan = previousScan
		}

		EDPMeasurement, err := scan.EDP()
		if err != nil {
			c.logger.Error("error converting scan to an EDP measurement", zap.Error(err), zap.String("scanner", string(s.ID())))
			// attempt to get the previous scan and convert it to EDP measurement
			previousScan, previousScanExists := previousScans[s.ID()]
			if !previousScanExists {
				c.logger.Error("no previous scan found", zap.String("scanner", string(s.ID())))
				continue
			}
			EDPMeasurement, err = previousScan.EDP()
			if err != nil {
				c.logger.Error("error converting previous scan to an EDP measurement", zap.Error(err), zap.String("scanner", string(s.ID())))
				continue
			}
			currentScans[s.ID()] = previousScan
		}

		EDPMeasurements = append(EDPMeasurements, EDPMeasurement)
	}

	return EDPMeasurements
}

// sendPayload sends the payload to the EDP backend.
func (c *Collector) sendPayload(payload Payload, subAccountID string) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload for subAccountID (%s): %w", subAccountID, err)
	}

	edpRequest, err := c.EDPClient.NewRequest(subAccountID)
	if err != nil {
		return fmt.Errorf("failed to create a new request for EDP for subAccountID (%s): %w", subAccountID, err)
	}

	resp, err := c.EDPClient.Send(edpRequest, payloadJSON)
	if err != nil {
		return fmt.Errorf("failed to send payload to EDP for subAccountID (%s): %w", subAccountID, err)
	}

	if !isSuccess(resp.StatusCode) {
		return fmt.Errorf("failed to send payload to EDP for subAccountID (%s) as it returned HTTP status code %d", subAccountID, resp.StatusCode)
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
