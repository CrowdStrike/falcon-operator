package sensor

import (
	"context"
	"errors"
	"testing"

	"github.com/crowdstrike/falcon-operator/internal/apitest"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client/sensor_update_policies"
	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetPreferredImage(t *testing.T) {
	ctx := context.Background()

	runner := func(t apitest.Test) {
		m := &mockFalcon{Mock: *t.GetMock()}
		images := ImageRepository{
			api:  m,
			tags: m,
		}

		image, err := images.GetPreferredImage(
			ctx,
			t.GetInput(0).(falcon.SensorType),
			t.GetStringPointerInput(1),
			t.GetStringPointerInput(2),
		)
		t.AssertExpectations(image, err)
	}

	noError := error(nil)
	noUpdatePolicyRequested := (*string)(nil)
	noVersionRequested := (*string)(nil)

	const policyDisabled = false
	const policyEnabled = true

	apitest.NewTest("latestVersion").
		WithInputs(falcon.SidecarSensor, noVersionRequested, noUpdatePolicyRequested).
		ExpectOutputs("someImageTag", noError).
		WithMockCall(newLastContainerTagCall(ctx, falcon.SidecarSensor, noVersionRequested, "someImageTag", noError)).
		Run(t, runner)

	apitest.NewTest("latestNodeSensorVersion").
		WithInputs(falcon.NodeSensor, noVersionRequested, noUpdatePolicyRequested).
		ExpectOutputs("someNodeImageTag", noError).
		WithMockCall(newLastNodeTagCall(ctx, noVersionRequested, "someNodeImageTag", noError)).
		Run(t, runner)

	apitest.NewTest("specificVersion").
		WithInputs(falcon.SidecarSensor, stringPointer("someSpecificVersion"), noUpdatePolicyRequested).
		ExpectOutputs("imageByVersion", noError).
		WithMockCall(newLastContainerTagCall(ctx, falcon.SidecarSensor, stringPointer("someSpecificVersion"), "imageByVersion", noError)).
		Run(t, runner)

	apitest.NewTest("versionByPolicy").
		WithInputs(falcon.SidecarSensor, noVersionRequested, stringPointer("somePolicyName")).
		ExpectOutputs("imageByPolicy", noError).
		WithMockCall(newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError)).
		WithMockCall(newGetSensorUpdatePoliciesCall("somePolicyID", "1.2.3", policyEnabled, noError)).
		WithMockCall(newLastContainerTagCall(ctx, falcon.SidecarSensor, stringPointer("1.2"), "imageByPolicy", noError)).
		Run(t, runner)

	apitest.NewTest("querySensorUpdatePoliciesFails").
		WithInputs(falcon.SidecarSensor, noVersionRequested, stringPointer("somePolicyName")).
		ExpectOutputs("", assert.AnError).
		WithMockCall(newQuerySensorUpdatePoliciesCall("somePolicyName", "", assert.AnError)).
		Run(t, runner)

	apitest.NewTest("getSensorUpdatePoliciesFails").
		WithInputs(falcon.SidecarSensor, noVersionRequested, stringPointer("somePolicyName")).
		ExpectOutputs("", assert.AnError).
		WithMockCall(newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError)).
		WithMockCall(newGetSensorUpdatePoliciesCall("somePolicyID", "", policyDisabled, assert.AnError)).
		Run(t, runner)

	apitest.NewTest("policyNameNotFound").
		WithInputs(falcon.SidecarSensor, noVersionRequested, stringPointer("somePolicyName")).
		ExpectOutputs("", errors.New("update-policy somePolicyName not found")).
		WithMockCall(newQuerySensorUpdatePoliciesCall("somePolicyName", "", noError)).
		Run(t, runner)

	apitest.NewTest("policyIDNotFound").
		WithInputs(falcon.SidecarSensor, noVersionRequested, stringPointer("somePolicyName")).
		ExpectOutputs("", errors.New("update-policy with ID somePolicyID not found")).
		WithMockCall(newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError)).
		WithMockCall(newGetSensorUpdatePoliciesCall("somePolicyID", "", policyDisabled, noError)).
		Run(t, runner)

	apitest.NewTest("policyDisabled").
		WithInputs(falcon.SidecarSensor, noVersionRequested, stringPointer("somePolicyName")).
		ExpectOutputs("", errors.New("update-policy with ID somePolicyID is disabled")).
		WithMockCall(newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError)).
		WithMockCall(newGetSensorUpdatePoliciesCall("somePolicyID", "1.2.3", policyDisabled, noError)).
		Run(t, runner)

	apitest.NewTest("invalidSensorVersion").
		WithInputs(falcon.SidecarSensor, noVersionRequested, stringPointer("somePolicyName")).
		ExpectOutputs("", errors.New("update-policy with ID somePolicyID has an invalid sensor version")).
		WithMockCall(newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError)).
		WithMockCall(newGetSensorUpdatePoliciesCall("somePolicyID", "1.2", policyEnabled, noError)).
		Run(t, runner)

	apitest.NewTest("lastContainerTagFails").
		WithInputs(falcon.SidecarSensor, noVersionRequested, noUpdatePolicyRequested).
		ExpectOutputs("", assert.AnError).
		WithMockCall(newLastContainerTagCall(ctx, falcon.SidecarSensor, noVersionRequested, "", assert.AnError)).
		Run(t, runner)

	apitest.NewTest("lastNodeTagFails").
		WithInputs(falcon.NodeSensor, noVersionRequested, noUpdatePolicyRequested).
		ExpectOutputs("", assert.AnError).
		WithMockCall(newLastNodeTagCall(ctx, noVersionRequested, "", assert.AnError)).
		Run(t, runner)
}

type mockFalcon struct {
	mock.Mock
}

func (m *mockFalcon) GetSensorUpdatePoliciesV2(params *sensor_update_policies.GetSensorUpdatePoliciesV2Params, opts ...sensor_update_policies.ClientOption) (*sensor_update_policies.GetSensorUpdatePoliciesV2OK, error) {
	args := m.Called(params, opts)
	return args.Get(0).(*sensor_update_policies.GetSensorUpdatePoliciesV2OK), args.Error(1)
}

func (m *mockFalcon) LastContainerTag(ctx context.Context, sensorType falcon.SensorType, versionRequested *string) (string, error) {
	args := m.Called(ctx, sensorType, versionRequested)
	return args.String(0), args.Error(1)
}

func (m *mockFalcon) LastNodeTag(ctx context.Context, versionRequested *string) (string, error) {
	args := m.Called(ctx, versionRequested)
	return args.String(0), args.Error(1)
}

func (m *mockFalcon) QuerySensorUpdatePolicies(params *sensor_update_policies.QuerySensorUpdatePoliciesParams, opts ...sensor_update_policies.ClientOption) (*sensor_update_policies.QuerySensorUpdatePoliciesOK, error) {
	args := m.Called(params, opts)
	return args.Get(0).(*sensor_update_policies.QuerySensorUpdatePoliciesOK), args.Error(1)
}

func newGetSensorUpdatePoliciesCall(policyID string, expectedVersion string, expectedStatus bool, expectedError error) *mock.Mock {
	params := sensor_update_policies.NewGetSensorUpdatePoliciesV2Params().WithIds([]string{policyID})

	payload := &models.SensorUpdateRespV2{}
	if expectedVersion != "" {
		payload.Resources = []*models.SensorUpdatePolicyV2{
			{
				Enabled: &expectedStatus,
				Settings: &models.SensorUpdateSettingsRespV2{
					SensorVersion: stringPointer(expectedVersion),
				},
			},
		}
	}

	m := &mock.Mock{}
	m.On("GetSensorUpdatePoliciesV2", params, []sensor_update_policies.ClientOption(nil)).
		Return(&sensor_update_policies.GetSensorUpdatePoliciesV2OK{Payload: payload}, expectedError)
	return m
}

func newLastContainerTagCall(ctx context.Context, sensorType falcon.SensorType, versionRequested *string, expectedImage string, expectedError error) *mock.Mock {
	m := &mock.Mock{}
	m.On("LastContainerTag", ctx, sensorType, versionRequested).Return(expectedImage, expectedError)
	return m
}

func newLastNodeTagCall(ctx context.Context, versionRequested *string, expectedImage string, expectedError error) *mock.Mock {
	m := &mock.Mock{}
	m.On("LastNodeTag", ctx, versionRequested).Return(expectedImage, expectedError)
	return m
}

func newQuerySensorUpdatePoliciesCall(updatePolicyRequested string, expectedPolicyID string, expectedError error) *mock.Mock {
	filter := falconFilter{}.
		addClause("platform_name", "Linux").
		addClause("name.raw", updatePolicyRequested).
		encode()

	params := sensor_update_policies.NewQuerySensorUpdatePoliciesParams().WithFilter(&filter)

	payload := &models.MsaQueryResponse{}
	if expectedPolicyID != "" {
		payload.Resources = []string{expectedPolicyID}
	}

	m := &mock.Mock{}
	m.On("QuerySensorUpdatePolicies", params, []sensor_update_policies.ClientOption(nil)).
		Return(&sensor_update_policies.QuerySensorUpdatePoliciesOK{Payload: payload}, expectedError)
	return m
}

func stringPointer(s string) *string {
	return &s
}
