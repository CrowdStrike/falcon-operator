package apitest

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type Test[T any] struct {
	expectedOutputs []any
	goTest          *testing.T
	inputs          []any
	m               *mock.Mock
	mockCalls       []*mock.Call
	name            string
	runnerArgs      T
}

func NewTest[T any](name string, runnerArgs T) *Test[T] {
	test := Test[T]{
		name:       name,
		runnerArgs: runnerArgs,
	}
	return &test
}

func (test Test[T]) AssertExpectations(outputs ...any) {
	test.m.AssertExpectations(test.goTest)

	for i, expectedValue := range test.expectedOutputs {
		assert.Equal(test.goTest, expectedValue, outputs[i], fmt.Sprintf("wrong value in output %d", i))
	}
}

func (test *Test[T]) ExpectOutputs(outputs ...any) *Test[T] {
	test.expectedOutputs = outputs
	return test
}

func (test Test[T]) GetInput(index int) any {
	return test.inputs[index]
}

func (test Test[T]) GetStringPointerInput(index int) *string {
	return test.GetInput(index).(*string)
}

func (test Test[T]) GetMock() *mock.Mock {
	return test.m
}

func (test Test[T]) Run(goTest *testing.T, runner func(Test[T], T)) {
	test.m = &mock.Mock{}
	for _, call := range test.mockCalls {
		call.Parent = test.m
	}
	test.m.ExpectedCalls = test.mockCalls

	goTest.Run(test.name, func(goTest *testing.T) {
		test.goTest = goTest
		runner(test, test.runnerArgs)
	})
}

func (test *Test[T]) WithMockCall(call *mock.Mock) *Test[T] {
	test.mockCalls = append(test.mockCalls, call.ExpectedCalls...)
	return test
}

func (test *Test[T]) WithInputs(inputs ...any) *Test[T] {
	test.inputs = inputs
	return test
}
