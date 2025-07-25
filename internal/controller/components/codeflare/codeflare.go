package codeflare

import (
	"context"
	"errors"
	"fmt"

	operatorv1 "github.com/openshift/api/operator/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/opendatahub-io/opendatahub-operator/v2/api/common"
	componentApi "github.com/opendatahub-io/opendatahub-operator/v2/api/components/v1alpha1"
	dscv1 "github.com/opendatahub-io/opendatahub-operator/v2/api/datasciencecluster/v1"
	cr "github.com/opendatahub-io/opendatahub-operator/v2/internal/controller/components/registry"
	"github.com/opendatahub-io/opendatahub-operator/v2/internal/controller/status"
	"github.com/opendatahub-io/opendatahub-operator/v2/pkg/controller/conditions"
	"github.com/opendatahub-io/opendatahub-operator/v2/pkg/controller/types"
	odhdeploy "github.com/opendatahub-io/opendatahub-operator/v2/pkg/deploy"
	"github.com/opendatahub-io/opendatahub-operator/v2/pkg/metadata/annotations"
)

type componentHandler struct{}

func init() { //nolint:gochecknoinits
	cr.Add(&componentHandler{})
}

func (s *componentHandler) GetName() string {
	return componentApi.CodeFlareComponentName
}

func (s *componentHandler) NewCRObject(dsc *dscv1.DataScienceCluster) common.PlatformObject {
	return &componentApi.CodeFlare{
		TypeMeta: metav1.TypeMeta{
			Kind:       componentApi.CodeFlareKind,
			APIVersion: componentApi.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: componentApi.CodeFlareInstanceName,
			Annotations: map[string]string{
				annotations.ManagementStateAnnotation: string(dsc.Spec.Components.CodeFlare.ManagementState),
			},
		},
		Spec: componentApi.CodeFlareSpec{
			CodeFlareCommonSpec: dsc.Spec.Components.CodeFlare.CodeFlareCommonSpec,
		},
	}
}

func (s *componentHandler) Init(_ common.Platform) error {
	if err := odhdeploy.ApplyParams(paramsPath, imageParamMap); err != nil {
		return fmt.Errorf("failed to update images on path %s: %w", paramsPath, err)
	}

	return nil
}

func (s *componentHandler) IsEnabled(dsc *dscv1.DataScienceCluster) bool {
	return dsc.Spec.Components.CodeFlare.ManagementState == operatorv1.Managed
}

func (s *componentHandler) UpdateDSCStatus(ctx context.Context, rr *types.ReconciliationRequest) (metav1.ConditionStatus, error) {
	cs := metav1.ConditionUnknown

	c := componentApi.CodeFlare{}
	c.Name = componentApi.CodeFlareInstanceName

	if err := rr.Client.Get(ctx, client.ObjectKeyFromObject(&c), &c); err != nil && !k8serr.IsNotFound(err) {
		return cs, err
	}

	dsc, ok := rr.Instance.(*dscv1.DataScienceCluster)
	if !ok {
		return cs, errors.New("failed to convert to DataScienceCluster")
	}

	dsc.Status.InstalledComponents[LegacyComponentName] = false
	dsc.Status.Components.CodeFlare.ManagementState = dsc.Spec.Components.CodeFlare.ManagementState
	dsc.Status.Components.CodeFlare.CodeFlareCommonStatus = nil

	rr.Conditions.MarkFalse(ReadyConditionType)

	if s.IsEnabled(dsc) {
		dsc.Status.InstalledComponents[LegacyComponentName] = true
		dsc.Status.Components.CodeFlare.CodeFlareCommonStatus = c.Status.CodeFlareCommonStatus.DeepCopy()

		if rc := conditions.FindStatusCondition(c.GetStatus(), status.ConditionTypeReady); rc != nil {
			rr.Conditions.MarkFrom(ReadyConditionType, *rc)
			cs = rc.Status
		} else {
			cs = metav1.ConditionFalse
		}
	} else {
		rr.Conditions.MarkFalse(
			ReadyConditionType,
			conditions.WithReason(string(dsc.Spec.Components.CodeFlare.ManagementState)),
			conditions.WithMessage("Component ManagementState is set to %s", dsc.Spec.Components.CodeFlare.ManagementState),
			conditions.WithSeverity(common.ConditionSeverityInfo),
		)
	}

	return cs, nil
}
