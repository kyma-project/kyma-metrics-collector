package edp

import (
	"context"
	"encoding/json"
	"time"

	"errors"
	"fmt"
	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"net/http"
)

type Collector struct {
	EDPClient *Client
	scanners  []resource.Scanner
}

var _ collector.CollectorSender = &Collector{}

func NewCollector(EDPClient *Client, scanner ...resource.Scanner) collector.CollectorSender {
	return &Collector{
		EDPClient: EDPClient,
		scanners:  scanner,
	}
}

func (c *Collector) CollectAndSend(ctx context.Context, runtime *runtime.Info, previousScans collector.ScanMap) (collector.ScanMap, error) {
	var errs []error

	childCtx, span := otel.Tracer("").Start(ctx, "kmc.collect_scans_and_send_measurements",
		trace.WithAttributes(
			attribute.String("provider", runtime.ProviderType),
			attribute.String("runtime_id", runtime.RuntimeID),
			attribute.String("subaccount_id", runtime.SubAccountID),
			attribute.String("shoot_name", runtime.ShootName),
		),
	)
	defer span.End()

	currentTimestamp := getTimestampNow()

	scans, err := c.executeScans(childCtx, runtime)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to execute all scans: %w", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	EDPMeasurements, err := c.convertScansToEDPMeasurements(scans, previousScans)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to convert all scans to EDP measurements: %w", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	payload := NewPayload(
		runtime.RuntimeID,
		runtime.SubAccountID,
		runtime.ShootName,
		currentTimestamp,
		EDPMeasurements,
	)
	err = c.sendPayload(payload, runtime.SubAccountID)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to send payload to EDP: %w", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return scans, errors.Join(errs...)
}

func (c *Collector) executeScans(ctx context.Context, runtime *runtime.Info) (collector.ScanMap, error) {
	var errs []error
	scans := make(collector.ScanMap)

	for _, s := range c.scanners {
		scan, err := s.Scan(ctx, runtime)
		if err != nil {
			errs = append(errs, fmt.Errorf("scanner with ID(%s) failed during scanning: %w", s.ID(), err))
			continue
		}
		// store only successful scans in the scan map
		scans[s.ID()] = scan
	}

	return scans, errors.Join(errs...)
}

func (c *Collector) convertScansToEDPMeasurements(currentScans collector.ScanMap, previousScans collector.ScanMap) ([]resource.EDPMeasurement, error) {
	var errs []error
	EDPMeasurements := []resource.EDPMeasurement{}

	for _, s := range c.scanners {
		scan, currentScanExists := currentScans[s.ID()]
		// if the current scan doesn't exist (because of a failure during execution), attempt to get the previous scan
		if !currentScanExists {
			previousScan, previousScanExists := previousScans[s.ID()]
			// if the previous scan also doesn't exist, nothing else we can do here
			if !previousScanExists {
				errs = append(errs, fmt.Errorf("no previous scan found for scanner with ID(%s)", s.ID()))
				continue
			}
			currentScans[s.ID()] = previousScan
			scan = previousScan
		}

		EDPMeasurement, err := scan.EDP()
		// if conversion to an EDP measurement fails, attempt to get the previous scan and convert it to EDP measurement
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to convert scan to an EDP measurement for scanner with ID(%s): %w", s.ID(), err))
			previousScan, previousScanExists := previousScans[s.ID()]
			// if the previous scan doesn't exist, nothing else we can do here
			if !previousScanExists {
				errs = append(errs, fmt.Errorf("no previous scan found for scanner with ID(%s)", s.ID()))
				continue
			}
			EDPMeasurement, err = previousScan.EDP()
			// if conversion of previous scan to an EDP measurement also fails, nothing else we can do here
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to convert previous scan to an EDP measurement for scanner with ID(%s): %w", s.ID(), err))
				continue
			}
			currentScans[s.ID()] = previousScan
		}

		EDPMeasurements = append(EDPMeasurements, EDPMeasurement)
	}

	return EDPMeasurements, errors.Join(errs...)
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
