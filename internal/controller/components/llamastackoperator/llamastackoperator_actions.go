package llamastackoperator

import (
	"context"
	"fmt"

	componentApi "github.com/opendatahub-io/opendatahub-operator/v2/api/components/v1alpha1"
	odhtypes "github.com/opendatahub-io/opendatahub-operator/v2/pkg/controller/types"
	odhdeploy "github.com/opendatahub-io/opendatahub-operator/v2/pkg/deploy"
)

func initialize(ctx context.Context, rr *odhtypes.ReconciliationRequest) error {
	rr.Manifests = append(rr.Manifests, manifestPath(rr.Release.Name))
	return nil
}

func devFlags(ctx context.Context, rr *odhtypes.ReconciliationRequest) error {
	llamastackoperator, ok := rr.Instance.(*componentApi.LlamaStackOperator)
	if !ok {
		return fmt.Errorf("resource instance %v is not a componentApi.LlamaStackOperator)", rr.Instance)
	}

	if llamastackoperator.Spec.DevFlags == nil {
		return nil
	}
	if len(llamastackoperator.Spec.DevFlags.Manifests) != 0 {
		manifestConfig := llamastackoperator.Spec.DevFlags.Manifests[0]
		if err := odhdeploy.DownloadManifests(ctx, ComponentName, manifestConfig); err != nil {
			return err
		}
		if manifestConfig.SourcePath != "" {
			rr.Manifests[0].Path = odhdeploy.DefaultManifestPath
			rr.Manifests[0].ContextDir = ComponentName
			rr.Manifests[0].SourcePath = manifestConfig.SourcePath
		}
	}
	return nil
}
