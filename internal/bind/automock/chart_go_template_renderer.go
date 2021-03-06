// Code generated by mockery v1.0.0
package automock

import chart "k8s.io/helm/pkg/proto/hapi/chart"
import chartutil "k8s.io/helm/pkg/chartutil"
import mock "github.com/stretchr/testify/mock"

// ChartGoTemplateRenderer is an autogenerated mock type for the ChartGoTemplateRenderer type
type ChartGoTemplateRenderer struct {
	mock.Mock
}

// Render provides a mock function with given fields: _a0, _a1
func (_m *ChartGoTemplateRenderer) Render(_a0 *chart.Chart, _a1 chartutil.Values) (map[string]string, error) {
	ret := _m.Called(_a0, _a1)

	var r0 map[string]string
	if rf, ok := ret.Get(0).(func(*chart.Chart, chartutil.Values) map[string]string); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*chart.Chart, chartutil.Values) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
