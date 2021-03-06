// Code generated by mockery v1.0.0
package automock

import mock "github.com/stretchr/testify/mock"

// BrokerSyncer is an autogenerated mock type for the BrokerSyncer type
type BrokerSyncer struct {
	mock.Mock
}

// SetNamespace provides a mock function with given fields: namespace
func (_m *BrokerSyncer) SetNamespace(namespace string) {
	_m.Called(namespace)
}

// Sync provides a mock function with given fields:
func (_m *BrokerSyncer) Sync() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
