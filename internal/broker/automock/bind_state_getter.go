// Code generated by mockery v1.0.0. DO NOT EDIT.

package automock

import internal "github.com/kyma-project/helm-broker/internal"
import mock "github.com/stretchr/testify/mock"

// bindStateGetter is an autogenerated mock type for the bindStateGetter type
type bindStateGetter struct {
	mock.Mock
}

// IsBinded provides a mock function with given fields: _a0, _a1
func (_m *bindStateGetter) IsBinded(_a0 internal.InstanceID, _a1 internal.BindingID) (bool, error) {
	ret := _m.Called(_a0, _a1)

	var r0 bool
	if rf, ok := ret.Get(0).(func(internal.InstanceID, internal.BindingID) bool); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(internal.InstanceID, internal.BindingID) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// IsBindingInProgress provides a mock function with given fields: _a0, _a1
func (_m *bindStateGetter) IsBindingInProgress(_a0 internal.InstanceID, _a1 internal.BindingID) (internal.OperationID, bool, error) {
	ret := _m.Called(_a0, _a1)

	var r0 internal.OperationID
	if rf, ok := ret.Get(0).(func(internal.InstanceID, internal.BindingID) internal.OperationID); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(internal.OperationID)
	}

	var r1 bool
	if rf, ok := ret.Get(1).(func(internal.InstanceID, internal.BindingID) bool); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Get(1).(bool)
	}

	var r2 error
	if rf, ok := ret.Get(2).(func(internal.InstanceID, internal.BindingID) error); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Error(2)
	}

	return r0, r1, r2
}