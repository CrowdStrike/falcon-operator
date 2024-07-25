package apitest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type Test struct {
	expectedOutputs []any
	goTest          *testing.T
	inputs          []any
	m               *mock.Mock
	mockCalls       []*mock.Call
	name            string
}

func NewTest(name string) *Test {
	test := Test{
		name: name,
	}
	return &test
}

func (test Test) AssertExpectations(outputs ...any) {
	test.m.AssertExpectations(test.goTest)

	for i, expectedValue := range test.expectedOutputs {
		assert.Equal(test.goTest, expectedValue, outputs[i], fmt.Sprintf("wrong value in output %d", i))
	}
}

func (test *Test) ExpectOutputs(outputs ...any) *Test {
	test.expectedOutputs = outputs
	return test
}

func (test Test) GetInput(index int) any {
	return test.inputs[index]
}

func (test Test) GetStringPointerInput(index int) *string {
	return test.GetInput(index).(*string)
}

func (test Test) GetMock() *mock.Mock {
	return test.m
}

func (test Test) Run(goTest *testing.T, runner func(Test)) {
	test.m = &mock.Mock{}
	for _, call := range test.mockCalls {
		call.Parent = test.m
	}
	test.m.ExpectedCalls = test.mockCalls

	goTest.Run(test.name, func(goTest *testing.T) {
		test.goTest = goTest
		runner(test)
	})
}

func (test *Test) WithMockCall(call *mock.Mock) *Test {
	test.mockCalls = append(test.mockCalls, call.ExpectedCalls...)
	return test
}

func (test *Test) WithInputs(inputs ...any) *Test {
	test.inputs = inputs
	return test
}
