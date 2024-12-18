package edp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"github.com/kyma-project/kyma-metrics-collector/pkg/collector"
	kmcotel "github.com/kyma-project/kyma-metrics-collector/pkg/otel"
	"github.com/kyma-project/kyma-metrics-collector/pkg/resource"
	"github.com/kyma-project/kyma-metrics-collector/pkg/runtime"
)

type Collector struct {
	EDPClient *Client
	scanners  []resource.Scanner
}

var errNoMeasurementsSent = errors.New("no measurements sent to EDP")

var _ collector.CollectorSender = &Collector{}

func NewCollector(EDPClient *Client, scanner ...resource.Scanner) collector.CollectorSender {
	return &Collector{
		EDPClient: EDPClient,
		scanners:  scanner,
	}
}

func (c *Collector) CollectAndSend(ctx context.Context, runtime *runtime.Info, previousScans collector.ScanMap) (collector.ScanMap, error) {
	var errs []error

	childCtx, span := otel.Tracer("").Start(ctx, "kmc.collect_scans_and_send_measurements", kmcotel.SpanAttributes(runtime))
	defer span.End()

	currentTimestamp := getTimestampNow()

	scans, err := c.executeScans(childCtx, previousScans, runtime)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to successfully execute one or more scans : %w", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	scans, EDPMeasurements, err := c.convertScansToEDPMeasurements(scans, previousScans, runtime)
	if err != nil {
		errs = append(errs, fmt.Errorf("failed to convert one or more scans to EDP measurements: %w", err))
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	if len(EDPMeasurements) == 0 {
		errs = append(errs, errNoMeasurementsSent)
		span.RecordError(err)
		span.SetStatus(codes.Error, errNoMeasurementsSent.Error())

		return scans, errors.Join(errs...)
	}

	payload := newPayload(
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

func (c *Collector) executeScans(ctx context.Context, previousScans collector.ScanMap, runtime *runtime.Info) (collector.ScanMap, error) {
	var errs []error

	currentScans := make(collector.ScanMap)

	for _, s := range c.scanners {
		scan, err := s.Scan(ctx, runtime)
		success := err == nil
		collector.RecordScan(success, string(s.ID()), *runtime)
		// if the scanner fails during execution, attempt to get the previous scan
		if err != nil {
			errs = append(errs, fmt.Errorf("scanner with ID(%s) failed during scanning: %w", s.ID(), err))

			previousScan, previousScanExists := previousScans[s.ID()]
			// if the previous scan also doesn't exist, nothing else we can do here
			if !previousScanExists {
				// since even a previous scan doesn't exist, we won't be able to convert any scan to an EDP measurement, so conversion is recorded as unsuccessful
				collector.RecordScanConversion(false, string(s.ID()), collector.EDPBackendName, *runtime)
				errs = append(errs, fmt.Errorf("no previous scan found for scanner with ID(%s)", s.ID()))

				continue
			}

			scan = previousScan
		}

		// store only successful scans in the scan map
		currentScans[s.ID()] = scan
	}

	return currentScans, errors.Join(errs...)
}

func (c *Collector) convertScansToEDPMeasurements(currentScans collector.ScanMap, previousScans collector.ScanMap, runtime *runtime.Info) (collector.ScanMap, []resource.EDPMeasurement, error) {
	var errs []error

	// we need to return convertableScans which are the scans that can be converted successfully to EDP measurements
	convertableScans := make(collector.ScanMap)
	EDPMeasurements := []resource.EDPMeasurement{}

	for id, scan := range currentScans {
		EDPMeasurement, err := scan.EDP()
		success := err == nil
		collector.RecordScanConversion(success, string(id), collector.EDPBackendName, *runtime)
		// if conversion to an EDP measurement fails, attempt to get the previous scan and convert it to EDP measurement
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to convert scan to an EDP measurement for scanner with ID(%s): %w", string(id), err))

			previousScan, previousScanExists := previousScans[id]
			// if the previous scan doesn't exist, nothing else we can do here
			if !previousScanExists {
				errs = append(errs, fmt.Errorf("no previous scan found for scanner with ID(%s)", string(id)))

				continue
			}

			EDPMeasurement, err = previousScan.EDP()
			// if conversion of previous scan to an EDP measurement also fails, nothing else we can do here
			if err != nil {
				errs = append(errs, fmt.Errorf("failed to convert previous scan to an EDP measurement for scanner with ID(%s): %w", string(id), err))

				continue
			}

			scan = previousScan
		}

		convertableScans[id] = scan
		EDPMeasurements = append(EDPMeasurements, EDPMeasurement)
	}

	return convertableScans, EDPMeasurements, errors.Join(errs...)
}

// sendPayload sends the payload to the EDP backend.
func (c *Collector) sendPayload(payload payload, subAccountID string) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload for subAccountID (%s): %w", subAccountID, err)
	}

	req, err := c.EDPClient.NewRequest(subAccountID)
	if err != nil {
		return fmt.Errorf("failed to create a new request for EDP for subAccountID (%s): %w", subAccountID, err)
	}

	resp, err := c.EDPClient.Send(req, payloadJSON)
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
