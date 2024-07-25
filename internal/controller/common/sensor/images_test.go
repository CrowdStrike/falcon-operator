package sensor

import (
	"context"
	"errors"
	"testing"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client/sensor_update_policies"
	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetPreferredImage(t *testing.T) {
	const policyDisabled = false
	const policyEnabled = true

	noUpdatePolicyRequested := (*string)(nil)
	noVersionRequested := (*string)(nil)
	noQuerySensorUpdatePoliciesCall := (*mock.Mock)(nil)
	noGetSensorUpdatePoliciesCall := (*mock.Mock)(nil)
	noLastContainerTagCall := (*mock.Mock)(nil)
	noLastNodeTagCall := (*mock.Mock)(nil)
	noError := error(nil)

	ctx := context.Background()

	tests := []struct {
		name                          string
		sensorType                    falcon.SensorType
		versionRequested              *string
		updatePolicyRequested         *string
		querySensorUpdatePoliciesCall *mock.Mock
		getSensorUpdatePoliciesCall   *mock.Mock
		lastContainerTagCall          *mock.Mock
		lastNodeTagCall               *mock.Mock
		expectedImage                 string
		expectedError                 error
	}{
		{
			"latestVersion",
			falcon.SidecarSensor,
			noVersionRequested,
			noUpdatePolicyRequested,
			noQuerySensorUpdatePoliciesCall,
			noGetSensorUpdatePoliciesCall,
			newLastContainerTagCall(ctx, falcon.SidecarSensor, noVersionRequested, "someImageTag", noError),
			noLastNodeTagCall,
			"someImageTag",
			noError,
		},

		{
			"latestNodeSensorVersion",
			falcon.NodeSensor,
			noVersionRequested,
			noUpdatePolicyRequested,
			noQuerySensorUpdatePoliciesCall,
			noGetSensorUpdatePoliciesCall,
			noLastContainerTagCall,
			newLastNodeTagCall(ctx, noVersionRequested, "someNodeImageTag", noError),
			"someNodeImageTag",
			noError,
		},

		{
			"specificVersion",
			falcon.SidecarSensor,
			stringPointer("someSpecificVersion"),
			noUpdatePolicyRequested,
			noQuerySensorUpdatePoliciesCall,
			noGetSensorUpdatePoliciesCall,
			newLastContainerTagCall(ctx, falcon.SidecarSensor, stringPointer("someSpecificVersion"), "imageByVersion", noError),
			noLastNodeTagCall,
			"imageByVersion",
			noError,
		},

		{
			"versionByPolicy",
			falcon.SidecarSensor,
			noVersionRequested,
			stringPointer("somePolicyName"),
			newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError),
			newGetSensorUpdatePoliciesCall("somePolicyID", "1.2.3", policyEnabled, noError),
			newLastContainerTagCall(ctx, falcon.SidecarSensor, stringPointer("1.2"), "imageByPolicy", noError),
			noLastNodeTagCall,
			"imageByPolicy",
			noError,
		},

		{
			"querySensorUpdatePoliciesFails",
			falcon.SidecarSensor,
			noVersionRequested,
			stringPointer("somePolicyName"),
			newQuerySensorUpdatePoliciesCall("somePolicyName", "", assert.AnError),
			noGetSensorUpdatePoliciesCall,
			noLastContainerTagCall,
			noLastNodeTagCall,
			"",
			assert.AnError,
		},

		{
			"getSensorUpdatePoliciesFails",
			falcon.SidecarSensor,
			noVersionRequested,
			stringPointer("somePolicyName"),
			newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError),
			newGetSensorUpdatePoliciesCall("somePolicyID", "", policyDisabled, assert.AnError),
			noLastContainerTagCall,
			noLastNodeTagCall,
			"",
			assert.AnError,
		},

		{
			"policyNameNotFound",
			falcon.SidecarSensor,
			noVersionRequested,
			stringPointer("somePolicyName"),
			newQuerySensorUpdatePoliciesCall("somePolicyName", "", noError),
			noGetSensorUpdatePoliciesCall,
			noLastContainerTagCall,
			noLastNodeTagCall,
			"",
			errors.New("update-policy somePolicyName not found"),
		},

		{
			"policyIDNotFound",
			falcon.SidecarSensor,
			noVersionRequested,
			stringPointer("somePolicyName"),
			newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError),
			newGetSensorUpdatePoliciesCall("somePolicyID", "", policyDisabled, noError),
			noLastContainerTagCall,
			noLastNodeTagCall,
			"",
			errors.New("update-policy with ID somePolicyID not found"),
		},

		{
			"policyDisabled",
			falcon.SidecarSensor,
			noVersionRequested,
			stringPointer("somePolicyName"),
			newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError),
			newGetSensorUpdatePoliciesCall("somePolicyID", "1.2.3", policyDisabled, noError),
			noLastContainerTagCall,
			noLastNodeTagCall,
			"",
			errors.New("update-policy with ID somePolicyID is disabled"),
		},

		{
			"invalidSensorVersion",
			falcon.SidecarSensor,
			noVersionRequested,
			stringPointer("somePolicyName"),
			newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError),
			newGetSensorUpdatePoliciesCall("somePolicyID", "1.2", policyEnabled, noError),
			noLastContainerTagCall,
			noLastNodeTagCall,
			"",
			errors.New("update-policy with ID somePolicyID has an invalid sensor version"),
		},

		{
			"lastContainerTagFails",
			falcon.SidecarSensor,
			noVersionRequested,
			stringPointer("somePolicyName"),
			newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError),
			newGetSensorUpdatePoliciesCall("somePolicyID", "1.2.3", policyEnabled, noError),
			newLastContainerTagCall(ctx, falcon.SidecarSensor, stringPointer("1.2"), "", assert.AnError),
			noLastNodeTagCall,
			"",
			assert.AnError,
		},

		{
			"lastNodeTagFails",
			falcon.NodeSensor,
			noVersionRequested,
			stringPointer("somePolicyName"),
			newQuerySensorUpdatePoliciesCall("somePolicyName", "somePolicyID", noError),
			newGetSensorUpdatePoliciesCall("somePolicyID", "1.2.3", policyEnabled, noError),
			noLastContainerTagCall,
			newLastNodeTagCall(ctx, stringPointer("1.2"), "", assert.AnError),
			"",
			assert.AnError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := &mockFalcon{}
			m.Mock.ExpectedCalls = combineCalls(
				&m.Mock,
				test.querySensorUpdatePoliciesCall,
				test.getSensorUpdatePoliciesCall,
				test.lastContainerTagCall,
				test.lastNodeTagCall,
			)

			images := ImageRepository{
				api:  m,
				tags: m,
			}
			image, err := images.GetPreferredImage(ctx, test.sensorType, test.versionRequested, test.updatePolicyRequested)
			assert.Equal(t, test.expectedImage, image)
			assert.Equal(t, test.expectedError, err)
			m.AssertExpectations(t)
		})
	}
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

func combineCalls(m *mock.Mock, calls ...*mock.Mock) []*mock.Call {
	var combined []*mock.Call

	for _, call := range calls {
		if call != nil {
			for _, expectation := range call.ExpectedCalls {
				expectation.Parent = m
				combined = append(combined, expectation)
			}
		}
	}

	return combined
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
