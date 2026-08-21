package application

import (
	"context"
	"fmt"
	"github.com/example/iot-device-management/internal/platform/apperr"
	clockpkg "github.com/example/iot-device-management/internal/platform/clock"
	"github.com/example/iot-device-management/internal/platform/id"
	"github.com/example/iot-device-management/internal/twin/domain"
	"reflect"
	"sort"
	"strings"
)

type Service struct {
	repo  domain.Repository
	clock clockpkg.Clock
}

func NewService(repo domain.Repository, clock clockpkg.Clock) *Service {
	if clock == nil {
		clock = clockpkg.SystemClock{}
	}
	return &Service{repo: repo, clock: clock}
}

func (s *Service) Get(ctx context.Context, deviceID string) (domain.TwinDocument, error) {
	if deviceID == "" {
		return domain.TwinDocument{}, apperr.E(apperr.KindInvalid, "twin.Get", "device id is required", nil)
	}
	document, err := s.repo.Get(ctx, deviceID)
	if err != nil {
		if apperr.IsKind(err, apperr.KindNotFound) {
			return domain.TwinDocument{
				DeviceID:      deviceID,
				DesiredState:  map[string]any{},
				ReportedState: map[string]any{},
				Version:       0,
				UpdatedAt:     s.clock.Now(),
			}, nil
		}
		return domain.TwinDocument{}, apperr.E(apperr.KindInternal, "twin.Get", "load twin", err)
	}
	return document, nil
}

func (s *Service) UpdateDesired(ctx context.Context, deviceID string, patch map[string]any) (domain.MergeResult, error) {
	if deviceID == "" {
		return domain.MergeResult{}, apperr.E(apperr.KindInvalid, "twin.UpdateDesired", "device id is required", nil)
	}
	if patch == nil {
		patch = map[string]any{}
	}
	document, err := s.Get(ctx, deviceID)
	if err != nil {
		return domain.MergeResult{}, err
	}
	document.DesiredState = mergeMaps(document.DesiredState, patch)
	document.Version++
	document.UpdatedAt = s.clock.Now()
	if err := s.repo.Save(ctx, document); err != nil {
		return domain.MergeResult{}, apperr.E(apperr.KindInternal, "twin.UpdateDesired", "save desired state", err)
	}
	diff := s.ComputeDiff(document.DesiredState, document.ReportedState)
	return s.mergeResult(document, diff), nil
}

func (s *Service) UpdateReported(ctx context.Context, deviceID string, patch map[string]any) (domain.MergeResult, error) {
	if deviceID == "" {
		return domain.MergeResult{}, apperr.E(apperr.KindInvalid, "twin.UpdateReported", "device id is required", nil)
	}
	if patch == nil {
		patch = map[string]any{}
	}
	document, err := s.Get(ctx, deviceID)
	if err != nil {
		return domain.MergeResult{}, err
	}
	// Reported state is an observation from the device. It never replaces the
	// authoritative desired state; it is merged into the reported map only.
	document.ReportedState = mergeMaps(document.ReportedState, patch)
	document.UpdatedAt = s.clock.Now()
	document.Version++
	if err := s.repo.Save(ctx, document); err != nil {
		return domain.MergeResult{}, apperr.E(apperr.KindInternal, "twin.UpdateReported", "save reported state", err)
	}
	diff := s.ComputeDiff(document.DesiredState, document.ReportedState)
	return s.mergeResult(document, diff), nil
}

func (s *Service) ComputeDiff(desired, reported map[string]any) []domain.StateDiff {
	keys := map[string]struct{}{}
	for key := range desired {
		keys[key] = struct{}{}
	}
	for key := range reported {
		keys[key] = struct{}{}
	}
	sorted := make([]string, 0, len(keys))
	for key := range keys {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)
	diffs := make([]domain.StateDiff, 0)
	for _, key := range sorted {
		desiredValue, desiredOK := desired[key]
		reportedValue, reportedOK := reported[key]
		if !desiredOK || !reportedOK {
			if desiredOK || reportedOK {
				diffs = append(diffs, domain.StateDiff{Property: key, Path: key, Desired: desiredValue, Reported: reportedValue})
			}
			continue
		}
		appendDiff(&diffs, key, key, desiredValue, reportedValue)
	}
	return diffs
}

func (s *Service) Merge(ctx context.Context, deviceID string) (domain.MergeResult, error) {
	document, err := s.Get(ctx, deviceID)
	if err != nil {
		return domain.MergeResult{}, err
	}
	diff := s.ComputeDiff(document.DesiredState, document.ReportedState)
	return s.mergeResult(document, diff), nil
}

func (s *Service) Delete(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return apperr.E(apperr.KindInvalid, "twin.Delete", "device id is required", nil)
	}
	if err := s.repo.Delete(ctx, deviceID); err != nil && !apperr.IsKind(err, apperr.KindNotFound) {
		return apperr.E(apperr.KindInternal, "twin.Delete", "delete twin", err)
	}
	return nil
}

func (s *Service) mergeResult(document domain.TwinDocument, diff []domain.StateDiff) domain.MergeResult {
	return domain.MergeResult{
		DeviceID:      document.DeviceID,
		DesiredState:  domain.TwinDocument{DesiredState: document.DesiredState}.Clone().DesiredState,
		ReportedState: domain.TwinDocument{ReportedState: document.ReportedState}.Clone().ReportedState,
		Version:       document.Version,
		Diff:          diff,
		UpdatedAt:     document.UpdatedAt,
	}
}

func mergeMaps(base, patch map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(patch))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range patch {
		if baseValue, exists := out[key]; exists {
			baseMap, baseIsMap := baseValue.(map[string]any)
			patchMap, patchIsMap := value.(map[string]any)
			if baseIsMap && patchIsMap {
				out[key] = mergeMaps(baseMap, patchMap)
				continue
			}
		}
		out[key] = value
	}
	return out
}

func appendDiff(diffs *[]domain.StateDiff, property, path string, desired, reported any) {
	if reflect.DeepEqual(desired, reported) {
		return
	}
	desiredMap, desiredOK := desired.(map[string]any)
	reportedMap, reportedOK := reported.(map[string]any)
	if desiredOK && reportedOK {
		keys := map[string]struct{}{}
		for key := range desiredMap {
			keys[key] = struct{}{}
		}
		for key := range reportedMap {
			keys[key] = struct{}{}
		}
		sorted := make([]string, 0, len(keys))
		for key := range keys {
			sorted = append(sorted, key)
		}
		sort.Strings(sorted)
		for _, key := range sorted {
			appendDiff(diffs, key, path+"."+key, desiredMap[key], reportedMap[key])
		}
		return
	}
	*diffs = append(*diffs, domain.StateDiff{
		Property: property,
		Path:     path,
		Desired:  desired,
		Reported: reported,
	})
}

func (s *Service) EnsureInitialized(ctx context.Context, deviceID string) error {
	document, err := s.Get(ctx, deviceID)
	if err != nil {
		return err
	}
	if document.DeviceID == "" {
		document.DeviceID = deviceID
	}
	if document.DesiredState == nil {
		document.DesiredState = map[string]any{}
	}
	if document.ReportedState == nil {
		document.ReportedState = map[string]any{}
	}
	document.UpdatedAt = s.clock.Now()
	return s.repo.Save(ctx, document)
}

func (s *Service) OperationID(prefix string) string {
	return id.New(prefix)
}

func (s *Service) ValidatePatch(patch map[string]any, maxDepth int) error {
	if maxDepth <= 0 {
		maxDepth = 10
	}
	var walk func(value any, depth int) error
	walk = func(value any, depth int) error {
		if depth > maxDepth {
			return fmt.Errorf("patch exceeds maximum depth %d", maxDepth)
		}
		switch typed := value.(type) {
		case map[string]any:
			for key, child := range typed {
				if strings.TrimSpace(key) == "" {
					return fmt.Errorf("empty key is not allowed")
				}
				if err := walk(child, depth+1); err != nil {
					return err
				}
			}
		case []any:
			for _, child := range typed {
				if err := walk(child, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return walk(patch, 1)
}
