package broker_test

import (
	"context"
	"errors"
	"testing"
	"time"

	jsonhash "github.com/komkom/go-jsonhash"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"k8s.io/helm/pkg/proto/hapi/chart"
	rls "k8s.io/helm/pkg/proto/hapi/services"

	"github.com/kyma-project/helm-broker/internal"
	"github.com/kyma-project/helm-broker/internal/bind"
	"github.com/kyma-project/helm-broker/internal/broker"
	"github.com/kyma-project/helm-broker/internal/broker/automock"
	"github.com/kyma-project/helm-broker/internal/platform/logger/spy"
)

func newProvisionServiceTestSuite(t *testing.T) *provisionServiceTestSuite {
	return &provisionServiceTestSuite{t: t}
}

type provisionServiceTestSuite struct {
	t   *testing.T
	Exp expAll
}

func (ts *provisionServiceTestSuite) SetUp() {
	ts.Exp.Populate()
}

func (ts *provisionServiceTestSuite) FixAddon() internal.Addon {
	return *ts.Exp.NewAddon()
}

func (ts *provisionServiceTestSuite) FixChart() chart.Chart {
	return *ts.Exp.NewChart()
}

func (ts *provisionServiceTestSuite) FixInstance() internal.Instance {
	return *ts.Exp.NewInstance()
}

func (ts *provisionServiceTestSuite) FixInstanceCollection() []*internal.Instance {
	return ts.Exp.NewInstanceCollection()
}

func (ts *provisionServiceTestSuite) FixInstanceOperation() internal.InstanceOperation {
	return *ts.Exp.NewInstanceOperation(internal.OperationTypeCreate, internal.OperationStateInProgress)
}

func (ts *provisionServiceTestSuite) FixProvisionRequest() osb.ProvisionRequest {
	return osb.ProvisionRequest{
		InstanceID: string(ts.Exp.InstanceID),
		ServiceID:  string(ts.Exp.Service.ID),
		PlanID:     string(ts.Exp.ServicePlan.ID),
		Parameters: make(internal.ChartValues),
		Context: map[string]interface{}{
			"namespace": string(ts.Exp.Namespace),
		},
		AcceptsIncomplete: true,
	}
}

func TestProvisionServiceProvisionSuccessAsyncInstall(t *testing.T) {
	// GIVEN
	ts := newProvisionServiceTestSuite(t)
	ts.SetUp()

	isgMock := &automock.InstanceStateGetter{}
	defer isgMock.AssertExpectations(t)
	isgMock.On("IsProvisioned", ts.Exp.InstanceID).Return(false, nil).Once()
	isgMock.On("IsProvisioningInProgress", ts.Exp.InstanceID).Return(internal.OperationID(""), false, nil).Once()

	bgMock := &automock.AddonStorage{}
	defer bgMock.AssertExpectations(t)
	expAddon := ts.FixAddon()
	bgMock.On("GetByID", internal.ClusterWide, ts.Exp.Addon.ID).Return(&expAddon, nil).Once()

	cgMock := &automock.ChartGetter{}
	defer cgMock.AssertExpectations(t)
	expChart := ts.FixChart()
	cgMock.On("Get", internal.ClusterWide, ts.Exp.Chart.Name, ts.Exp.Chart.Version).Return(&expChart, nil).Once()

	iiMock := &automock.InstanceStorage{}
	defer iiMock.AssertExpectations(t)
	expInstance := ts.FixInstance()
	params := jsonhash.HashS(ts.FixProvisionRequest().Parameters)
	expInstance.ParamsHash = params
	expInstanceCollection := ts.FixInstanceCollection()
	iiMock.On("Insert", &expInstance).Return(nil).Once()
	iiMock.On("GetAll").Return(expInstanceCollection, nil)

	ioMock := &automock.OperationStorage{}
	defer ioMock.AssertExpectations(t)
	expInstOp := ts.FixInstanceOperation()
	expInstOp.ParamsHash = params
	ioMock.On("Insert", &expInstOp).Return(nil).Once()
	operationSucceeded := make(chan struct{})
	ioMock.On("UpdateStateDesc", ts.Exp.InstanceID, ts.Exp.OperationID, internal.OperationStateSucceeded, mock.Anything).Return(nil).Once().
		Run(func(mock.Arguments) { close(operationSucceeded) })

	hiMock := &automock.HelmClient{}
	defer hiMock.AssertExpectations(t)
	releaseResp := &rls.InstallReleaseResponse{}
	expChartOverrides := internal.ChartValues{
		"addonsRepositoryURL": expAddon.RepositoryURL,
	}
	hiMock.On("Install", &expChart, expChartOverrides, ts.Exp.ReleaseName, ts.Exp.Namespace).Return(releaseResp, nil).Once()

	renderedYAML := bind.RenderedBindYAML(`rendered-template`)
	rendererMock := &automock.BindTemplateRenderer{}
	defer rendererMock.AssertExpectations(t)
	rendererMock.On("Render", ts.Exp.AddonPlan.BindTemplate, releaseResp).Return(renderedYAML, nil)

	expCreds := internal.InstanceCredentials{
		"test-param": "test-value",
	}
	resolverMock := &automock.BindTemplateResolver{}
	defer resolverMock.AssertExpectations(t)
	resolverMock.On("Resolve", renderedYAML, ts.Exp.Namespace).Return(&bind.ResolveOutput{
		Credentials: expCreds,
	}, nil)

	expInsert := internal.InstanceBindData{InstanceID: ts.Exp.InstanceID, Credentials: expCreds}
	ibdMock := &automock.InstanceBindDataInserter{}
	defer ibdMock.AssertExpectations(t)
	ibdMock.On("Insert", &expInsert).Return(nil)

	oipFake := func() (internal.OperationID, error) {
		return ts.Exp.OperationID, nil
	}

	testHookCalled := make(chan struct{})

	svc := broker.NewProvisionService(bgMock, cgMock, iiMock, isgMock, ioMock, ioMock, ibdMock, rendererMock, resolverMock, hiMock, oipFake, spy.NewLogDummy()).
		WithTestHookOnAsyncCalled(func(opID internal.OperationID) {
			assert.Equal(t, ts.Exp.OperationID, opID)
			close(testHookCalled)
		})

	ctx := context.Background()
	osbCtx := *broker.NewOSBContext("", "v1")
	req := ts.FixProvisionRequest()

	// WHEN
	resp, err := svc.Provision(ctx, osbCtx, &req)

	// THEN
	assert.Nil(t, err)
	assert.True(t, resp.Async)
	assert.EqualValues(t, ts.Exp.OperationID, *resp.OperationKey)

	select {
	case <-operationSucceeded:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("timeout on operation succeeded")
	}

	select {
	case <-testHookCalled:
	default:
		t.Fatal("async test hook not called")
	}
}

func TestProvisionServiceProvisionFailureAsync(t *testing.T) {
	// GIVEN
	ts := newProvisionServiceTestSuite(t)
	ts.SetUp()

	isgMock := &automock.InstanceStateGetter{}
	defer isgMock.AssertExpectations(t)
	isgMock.On("IsProvisioned", ts.Exp.InstanceID).Return(false, nil).Once()
	isgMock.On("IsProvisioningInProgress", ts.Exp.InstanceID).Return(internal.OperationID(""), false, nil).Once()

	bgMock := &automock.AddonStorage{}
	defer bgMock.AssertExpectations(t)
	expAddon := ts.FixAddon()
	bgMock.On("GetByID", internal.ClusterWide, ts.Exp.Addon.ID).Return(&expAddon, nil).Once()
	iiMock := &automock.InstanceStorage{}
	defer iiMock.AssertExpectations(t)
	expInstance := ts.FixInstance()
	params := jsonhash.HashS(ts.FixProvisionRequest().Parameters)
	expInstance.ParamsHash = params
	expInstanceCollection := ts.FixInstanceCollection()
	iiMock.On("Insert", &expInstance).Return(nil).Once()
	iiMock.On("GetAll").Return(expInstanceCollection, nil)

	ioMock := &automock.OperationStorage{}
	defer ioMock.AssertExpectations(t)
	expInstOp := ts.FixInstanceOperation()
	expInstOp.ParamsHash = params
	ioMock.On("Insert", &expInstOp).Return(nil).Once()

	cgMock := &automock.ChartGetter{}
	defer cgMock.AssertExpectations(t)
	expChartError := errors.New("fake-chart-error")
	cgMock.On("Get", internal.ClusterWide, ts.Exp.Chart.Name, ts.Exp.Chart.Version).Return(nil, expChartError).Once()

	operationFailed := make(chan struct{})
	ioMock.On("UpdateStateDesc", ts.Exp.InstanceID, ts.Exp.OperationID, internal.OperationStateFailed, mock.Anything).Return(nil).Once().
		Run(func(mock.Arguments) { close(operationFailed) })

	hiMock := &automock.HelmClient{}
	defer hiMock.AssertExpectations(t)

	oipFake := func() (internal.OperationID, error) {
		return ts.Exp.OperationID, nil
	}

	testHookCalled := make(chan struct{})

	svc := broker.NewProvisionService(bgMock, cgMock, iiMock, isgMock, ioMock, ioMock, nil, nil, nil, hiMock, oipFake, spy.NewLogDummy()).
		WithTestHookOnAsyncCalled(func(opID internal.OperationID) {
			assert.Equal(t, ts.Exp.OperationID, opID)
			close(testHookCalled)
		})

	ctx := context.Background()
	osbCtx := *broker.NewOSBContext("", "v1")
	req := ts.FixProvisionRequest()

	// WHEN
	resp, err := svc.Provision(ctx, osbCtx, &req)

	// THEN
	assert.Nil(t, err)
	assert.True(t, resp.Async)
	assert.EqualValues(t, ts.Exp.OperationID, *resp.OperationKey)

	select {
	case <-operationFailed:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("timeout on operation failed")
	}

	select {
	case <-testHookCalled:
	default:
		t.Fatal("async test hook not called")
	}
}

func TestProvisionServiceProvisionSuccessRepeatedOnAlreadyFullyProvisionedInstance(t *testing.T) {
	// GIVEN
	ts := newProvisionServiceTestSuite(t)
	ts.SetUp()

	isgMock := &automock.InstanceStateGetter{}
	defer isgMock.AssertExpectations(t)
	isgMock.On("IsProvisioned", ts.Exp.InstanceID).Return(true, nil).Once()

	bgMock := &automock.AddonStorage{}
	defer bgMock.AssertExpectations(t)

	cgMock := &automock.ChartGetter{}
	defer cgMock.AssertExpectations(t)

	iiMock := &automock.InstanceStorage{}
	fixInstance := ts.FixInstance()
	fixInstance.ParamsHash = jsonhash.HashS(map[string]interface{}{})
	iiMock.On("Get", fixInstance.ID).Return(&fixInstance, nil)
	defer iiMock.AssertExpectations(t)

	ioMock := &automock.OperationStorage{}
	defer ioMock.AssertExpectations(t)

	hiMock := &automock.HelmClient{}
	defer hiMock.AssertExpectations(t)

	oipFake := func() (internal.OperationID, error) {
		t.Error("operation ID provider called when it should not be")
		return ts.Exp.OperationID, nil
	}

	testHookCalled := make(chan struct{})

	svc := broker.NewProvisionService(bgMock, cgMock, iiMock, isgMock, ioMock, ioMock, nil, nil, nil, hiMock, oipFake, spy.NewLogDummy()).
		WithTestHookOnAsyncCalled(func(internal.OperationID) { close(testHookCalled) })

	ctx := context.Background()
	osbCtx := *broker.NewOSBContext("", "v1")
	req := ts.FixProvisionRequest()

	// WHEN
	resp, err := svc.Provision(ctx, osbCtx, &req)

	// THEN
	assert.Nil(t, err)
	assert.False(t, resp.Async)
	assert.Nil(t, resp.OperationKey)

	select {
	case <-testHookCalled:
		t.Fatal("async test hook called")
	default:
	}
}

func TestProvisionServiceProvisionSuccessRepeatedOnProvisioningInProgress(t *testing.T) {
	// GIVEN
	ts := newProvisionServiceTestSuite(t)
	ts.SetUp()

	isgMock := &automock.InstanceStateGetter{}
	defer isgMock.AssertExpectations(t)
	isgMock.On("IsProvisioned", ts.Exp.InstanceID).Return(false, nil).Once()
	expOpID := internal.OperationID("exp-op-id")
	isgMock.On("IsProvisioningInProgress", ts.Exp.InstanceID).Return(expOpID, true, nil).Once()

	bgMock := &automock.AddonStorage{}
	defer bgMock.AssertExpectations(t)

	cgMock := &automock.ChartGetter{}
	defer cgMock.AssertExpectations(t)

	iiMock := &automock.InstanceStorage{}
	fixInstance := ts.FixInstance()
	fixInstance.ParamsHash = jsonhash.HashS(map[string]interface{}{})
	iiMock.On("Get", fixInstance.ID).Return(&fixInstance, nil)
	defer iiMock.AssertExpectations(t)

	ioMock := &automock.OperationStorage{}
	defer ioMock.AssertExpectations(t)

	hiMock := &automock.HelmClient{}
	defer hiMock.AssertExpectations(t)

	oipFake := func() (internal.OperationID, error) {
		t.Error("operation ID provider called when it should not be")
		return ts.Exp.OperationID, nil
	}

	testHookCalled := make(chan struct{})

	svc := broker.NewProvisionService(bgMock, cgMock, iiMock, isgMock, ioMock, ioMock, nil, nil, nil, hiMock, oipFake, spy.NewLogDummy()).
		WithTestHookOnAsyncCalled(func(internal.OperationID) { close(testHookCalled) })

	ctx := context.Background()
	osbCtx := *broker.NewOSBContext("", "v1")
	req := ts.FixProvisionRequest()

	// WHEN
	resp, err := svc.Provision(ctx, osbCtx, &req)

	// THEN
	assert.Nil(t, err)
	assert.True(t, resp.Async)
	assert.EqualValues(t, expOpID, *resp.OperationKey)

	select {
	case <-testHookCalled:
		t.Fatal("async test hook called")
	default:
	}
}
