package process

import (
	"strings"
	"time"

	kebruntime "github.com/kyma-project/kyma-environment-broker/common/runtime"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"

	kmccache "github.com/kyma-project/kyma-metrics-collector/pkg/cache"
	log "github.com/kyma-project/kyma-metrics-collector/pkg/logger"
)

// pollKEBForRuntimes polls KEB for runtimes information.
func (p *Process) pollKEBForRuntimes() {
	kebReq, err := p.KEBClient.NewRequest()
	if err != nil {
		p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
			Fatal("create a new request for KEB")
	}

	for {
		runtimesPage, err := p.KEBClient.GetAllRuntimes(kebReq)
		if err != nil {
			p.namedLogger().With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).
				Error("get runtimes from KEB")
			p.namedLogger().Infof("waiting to poll KEB again after %v....", p.KEBClient.Config.PollWaitDuration)
			time.Sleep(p.KEBClient.Config.PollWaitDuration)

			continue
		}

		p.namedLogger().Debugf("num of runtimes are: %d", runtimesPage.Count)
		p.populateCacheAndQueue(runtimesPage)
		p.namedLogger().Debugf("length of the cache after KEB is done populating: %d", p.Cache.ItemCount())
		p.namedLogger().Infof("waiting to poll KEB again after %v....", p.KEBClient.Config.PollWaitDuration)
		recordItemsInCache(float64(p.Cache.ItemCount()))
		time.Sleep(p.KEBClient.Config.PollWaitDuration)
	}
}

// populateCacheAndQueue populates Cache and Queue with new runtimes and deletes the runtimes which should not be tracked.
func (p *Process) populateCacheAndQueue(runtimes *kebruntime.RuntimesPage) {
	// clear the gauge to fill it with the new data
	kebFetchedClusters.Reset()

	validSubAccounts := make(map[string]bool)

	for _, runtime := range runtimes.Data {
		if runtime.SubAccountID == "" {
			continue
		}

		validSubAccounts[runtime.SubAccountID] = true
		recordObj, isFoundInCache := p.Cache.Get(runtime.SubAccountID)

		// Get provisioning and deprovisioning states if available otherwise return empty string for logging.
		provisioning := getOrDefault(runtime.Status.Provisioning, "")
		deprovisioning := getOrDefault(runtime.Status.Deprovisioning, "")
		p.namedLoggerWithRuntime(runtime).
			With(log.KeyRuntimeState, runtime.Status.State).
			With(log.KeyProvisioningStatus, provisioning).
			With(log.KeyDeprovisioningStatus, deprovisioning).
			Debug("Runtime state")

		if isRuntimeTrackable(runtime) {
			newRecord := kmccache.Record{
				SubAccountID:    runtime.SubAccountID,
				RuntimeID:       runtime.RuntimeID,
				InstanceID:      runtime.InstanceID,
				GlobalAccountID: runtime.GlobalAccountID,
				ShootName:       runtime.ShootName,
				ProviderType:    strings.ToLower(runtime.Provider),
				KubeConfig:      "",
				Metric:          nil,
			}

			// record kebFetchedClusters metric for trackable cluster
			recordKEBFetchedClusters(
				trackableTrue,
				runtime.ShootName,
				runtime.InstanceID,
				runtime.RuntimeID,
				runtime.SubAccountID,
				runtime.GlobalAccountID)

			// Cluster is trackable but does not exist in the cache
			if !isFoundInCache {
				err := p.Cache.Add(runtime.SubAccountID, newRecord, cache.NoExpiration)
				if err != nil {
					p.namedLoggerWithRecord(&newRecord).With(log.KeyResult, log.ValueFail).With(log.KeyError, err.Error()).Error("Failed to add subAccountID to cache. Skipping queueing it")
					continue
				}

				p.Queue.Add(runtime.SubAccountID)
				p.namedLoggerWithRecord(&newRecord).With(log.KeyResult, log.ValueSuccess).Debug("Queued and added to cache")

				continue
			}

			// Cluster is trackable and exists in the cache
			if record, ok := recordObj.(kmccache.Record); ok {
				if record.ShootName == runtime.ShootName {
					continue
				}
				// The shootname has changed hence the record in the cache is not valid anymore
				// No need to queue as the subAccountID already exists in queue
				p.Cache.Set(runtime.SubAccountID, newRecord, cache.NoExpiration)
				p.namedLoggerWithRecord(&record).Debug("Resetted the values in cache for subAccount")

				// delete metrics for old shoot name.
				if success := deleteMetrics(record); !success {
					p.namedLoggerWithRecord(&record).Info("prometheus metrics were not successfully removed for subAccount")
				}
			}

			continue
		}

		// record kebFetchedClusters metric for not trackable clusters
		recordKEBFetchedClusters(
			trackableFalse,
			runtime.ShootName,
			runtime.InstanceID,
			runtime.RuntimeID,
			runtime.SubAccountID,
			runtime.GlobalAccountID)

		if isFoundInCache {
			// Cluster is not trackable but is found in cache should be deleted
			p.Cache.Delete(runtime.SubAccountID)
			p.Queue.Done(runtime.SubAccountID)
			p.namedLogger().With(log.KeySubAccountID, runtime.SubAccountID).
				With(log.KeyRuntimeID, runtime.RuntimeID).Debug("Deleted subAccount from cache")
			// delete metrics for old shoot name.
			if record, ok := recordObj.(kmccache.Record); ok {
				if success := deleteMetrics(record); !success {
					p.namedLoggerWithRecord(&record).Info("prometheus metrics were not successfully removed for subAccount")
				}
			}

			continue
		}

		p.namedLogger().With(log.KeySubAccountID, runtime.SubAccountID).
			With(log.KeyRuntimeID, runtime.RuntimeID).Debug("Ignoring SubAccount as it is not trackable")
	}

	// Cleaning up subAccounts from the cache which are not returned by KEB anymore
	for sAccID, recordObj := range p.Cache.Items() {
		if _, ok := validSubAccounts[sAccID]; !ok {
			record, ok := recordObj.Object.(kmccache.Record)

			p.Cache.Delete(sAccID)

			if !ok {
				p.namedLoggerWithRecord(&record).
					Error("bad item from cache, could not cast to a record obj")
			} else {
				p.namedLoggerWithRecord(&record).
					Info("SubAccount is not trackable anymore, deleting it from cache")
			}
			// delete metrics for old shoot name.
			if success := deleteMetrics(record); !success {
				p.namedLoggerWithRecord(&record).
					Info("prometheus metrics were not successfully removed for subAccount")
			}
		}
	}
}

// getOrDefault returns the runtime state or a default value if runtimeStatus is nil.
func getOrDefault(runtimeStatus *kebruntime.Operation, defaultValue string) string {
	if runtimeStatus != nil {
		return runtimeStatus.State
	}

	return defaultValue
}

func isRuntimeTrackable(runtime kebruntime.RuntimeDTO) bool {
	if runtime.Status.State == kebruntime.StateDeprovisioning {
		return false
	}

	return isTrackableState(runtime.Status.State) || isProvisionedStatus(runtime)
}

// isProvisionedStatus returns true if the runtime is successfully provisioned, otherwise returns false.
func isProvisionedStatus(runtime kebruntime.RuntimeDTO) bool {
	if runtime.Status.Provisioning != nil &&
		runtime.Status.Provisioning.State == string(kebruntime.StateSucceeded) &&
		runtime.Status.Deprovisioning == nil {
		return true
	}

	return false
}

// isTrackableState returns true if the runtime state is trackable, otherwise returns false.
func isTrackableState(state kebruntime.State) bool {
	//nolint:exhaustive // we only care about these states
	switch state {
	case kebruntime.StateSucceeded, kebruntime.StateError, kebruntime.StateUpgrading, kebruntime.StateUpdating:
		return true
	}

	return false
}

func (p *Process) namedLogger() *zap.SugaredLogger {
	return p.Logger.With("component", "kmc")
}

func (p *Process) namedLoggerWithRecord(record *kmccache.Record) *zap.SugaredLogger {
	logger := p.Logger.With("component", "kmc")

	if record == nil {
		return logger
	}

	return logger.With(log.KeyRuntimeID, record.RuntimeID).With(log.KeyShoot, record.ShootName).With(log.KeySubAccountID, record.SubAccountID).With(log.KeyGlobalAccountID, record.GlobalAccountID)
}

func (p *Process) namedLoggerWithRuntime(runtime kebruntime.RuntimeDTO) *zap.SugaredLogger {
	return p.Logger.With("component", "kmc").With(log.KeyRuntimeID, runtime.RuntimeID).With(log.KeyShoot, runtime.ShootName).With(log.KeySubAccountID, runtime.SubAccountID).With(log.KeyGlobalAccountID, runtime.GlobalAccountID)
}
