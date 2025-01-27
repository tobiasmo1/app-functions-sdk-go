//
// Copyright (c) 2019 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package appsdk

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	"github.com/edgexfoundry/app-functions-sdk-go/appcontext"
	"github.com/edgexfoundry/app-functions-sdk-go/internal"
	"github.com/edgexfoundry/app-functions-sdk-go/internal/common"
	"github.com/edgexfoundry/app-functions-sdk-go/internal/runtime"
	triggerHttp "github.com/edgexfoundry/app-functions-sdk-go/internal/trigger/http"
	"github.com/edgexfoundry/app-functions-sdk-go/internal/trigger/messagebus"
	"github.com/edgexfoundry/app-functions-sdk-go/internal/webserver"
	"github.com/edgexfoundry/go-mod-core-contracts/clients"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/clients/types"
	"github.com/edgexfoundry/go-mod-core-contracts/models"
)

var lc logger.LoggingClient
var params types.EndpointParams

func init() {
	lc = logger.NewClient("app_functions_sdk_go", false, "./test.log", "DEBUG")
	params = types.EndpointParams{
		ServiceKey:  clients.SupportSchedulerServiceKey,
		Path:        clients.ApiIntervalActionRoute,
		UseRegistry: false,
		Url:         "http://test" + clients.ApiIntervalActionRoute,
		Interval:    int(clients.ClientMonitorDefault / time.Millisecond),
	}
}

func IsInstanceOf(objectPtr, typePtr interface{}) bool {
	return reflect.TypeOf(objectPtr) == reflect.TypeOf(typePtr)
}
func TestAddRoute(t *testing.T) {
	router := mux.NewRouter()
	ws := webserver.NewWebServer(&common.ConfigurationStruct{}, lc, router)

	sdk := AppFunctionsSDK{
		webserver: ws,
	}
	sdk.AddRoute("/test", func(http.ResponseWriter, *http.Request) {}, "GET")
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		assert.Equal(t, "/test", path)
		return nil
	})

}

func TestSetupHTTPTrigger(t *testing.T) {
	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Binding: common.BindingInfo{
				Type: "htTp",
			},
		},
	}
	runtime := &runtime.GolangRuntime{}
	runtime.Initialize(nil)
	runtime.SetTransforms(sdk.transforms)
	trigger := sdk.setupTrigger(sdk.config, runtime)
	result := IsInstanceOf(trigger, (*triggerHttp.Trigger)(nil))
	assert.True(t, result, "Expected Instance of HTTP Trigger")
}
func TestSetupMessageBusTrigger(t *testing.T) {
	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Binding: common.BindingInfo{
				Type: "meSsaGebus",
			},
		},
	}
	runtime := &runtime.GolangRuntime{}
	runtime.Initialize(nil)
	runtime.SetTransforms(sdk.transforms)
	trigger := sdk.setupTrigger(sdk.config, runtime)
	result := IsInstanceOf(trigger, (*messagebus.Trigger)(nil))
	assert.True(t, result, "Expected Instance of Message Bus Trigger")
}
func TestSetFunctionsPipelineNoTransforms(t *testing.T) {
	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Binding: common.BindingInfo{
				Type: "meSsaGebus",
			},
		},
	}
	err := sdk.SetFunctionsPipeline()
	assert.NotNil(t, err, "There should be an error")
	assert.Equal(t, err.Error(), "No transforms provided to pipeline")
}
func TestSetFunctionsPipelineOneTransform(t *testing.T) {
	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		runtime:       &runtime.GolangRuntime{},
		config: common.ConfigurationStruct{
			Binding: common.BindingInfo{
				Type: "meSsaGebus",
			},
		},
	}
	function := func(edgexcontext *appcontext.Context, params ...interface{}) (bool, interface{}) {
		return true, nil
	}

	sdk.runtime.Initialize(nil)
	err := sdk.SetFunctionsPipeline(function)
	assert.Nil(t, err, "There should be no error")
	assert.Equal(t, 1, len(sdk.transforms))
}
func TestApplicationSettings(t *testing.T) {
	expectedSettingKey := "ApplicationName"
	expectedSettingValue := "simple-filter-xml"

	sdk := AppFunctionsSDK{}

	sdk.configDir = "../examples/simple-filter-xml/res"

	config, err := readConfigurationFromFile(sdk.configProfile, sdk.configDir)
	assert.NoError(t, err, "failed to load configuration from TOML file")

	err = sdk.initializeConfiguration(config)

	assert.NoError(t, err, "failed to initialize configuration")

	appSettings := sdk.ApplicationSettings()
	if !assert.NotNil(t, appSettings, "returned application settings is nil") {
		t.Fatal()
	}

	actual, ok := appSettings[expectedSettingKey]
	if !assert.True(t, ok, "expected application setting key not fond") {
		t.Fatal()
	}

	assert.Equal(t, expectedSettingValue, actual, "actual application setting value not as expected")

}

func TestApplicationSettingsNil(t *testing.T) {
	sdk := AppFunctionsSDK{}

	sdk.configDir = "../examples/simple-filter-xml-post/res"

	config, err := readConfigurationFromFile(sdk.configProfile, sdk.configDir)
	assert.NoError(t, err, "failed to load configuration from TOML file")

	err = sdk.initializeConfiguration(config)
	assert.NoError(t, err, "failed to initialize configuration")

	appSettings := sdk.ApplicationSettings()
	if !assert.Nil(t, appSettings, "returned application settings expected to be nil") {
		t.Fatal()
	}
}

func TestLoadConfigurablePipelineFunctionNotFound(t *testing.T) {
	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Writable: common.WritableInfo{
				Pipeline: common.PipelineInfo{
					ExecutionOrder: "Bogus",
					Functions:      make(map[string]common.PipelineFunction),
				},
			},
		},
	}

	appFunctions, err := sdk.LoadConfigurablePipeline()
	assert.Error(t, err, "expected error for function not found in config")
	assert.Equal(t, err.Error(), "Function Bogus configuration not found in Pipeline.Functions section")
	assert.Nil(t, appFunctions, "expected app functions list to be nil")
}

func TestLoadConfigurablePipelineNotABuiltInSdkFunction(t *testing.T) {
	functions := make(map[string]common.PipelineFunction)
	functions["Bogus"] = common.PipelineFunction{}

	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Writable: common.WritableInfo{
				Pipeline: common.PipelineInfo{
					ExecutionOrder: "Bogus",
					Functions:      functions,
				},
			},
		},
	}

	appFunctions, err := sdk.LoadConfigurablePipeline()
	assert.Error(t, err, "expected error")
	assert.Equal(t, err.Error(), "Function Bogus is not a built in SDK function")
	assert.Nil(t, appFunctions, "expected app functions list to be nil")
}

func TestLoadConfigurablePipelineAddressableConfig(t *testing.T) {
	functionName := "MQTTSend"
	functions := make(map[string]common.PipelineFunction)
	functions[functionName] = common.PipelineFunction{
		Parameters: map[string]string{"qos": "0", "autoreconnect": "false"},
		Addressable: models.Addressable{
			Address:   "localhost",
			Port:      1883,
			Protocol:  "tcp",
			Publisher: "MyApp",
			Topic:     "sampleTopic",
		},
	}

	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Writable: common.WritableInfo{
				Pipeline: common.PipelineInfo{
					ExecutionOrder: functionName,
					Functions:      functions,
				},
			},
		},
	}

	appFunctions, err := sdk.LoadConfigurablePipeline()
	assert.NoError(t, err, "")
	assert.NotNil(t, appFunctions, "expected app functions list to be set")
}

func TestLoadConfigurablePipelineNumFunctions(t *testing.T) {
	functions := make(map[string]common.PipelineFunction)
	functions["FilterByDeviceName"] = common.PipelineFunction{
		Parameters: map[string]string{"FilterValues": "Random-Float-Device, Random-Integer-Device"},
	}
	functions["TransformToXML"] = common.PipelineFunction{}
	functions["SetOutputData"] = common.PipelineFunction{}

	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Writable: common.WritableInfo{
				Pipeline: common.PipelineInfo{
					ExecutionOrder: "FilterByDeviceName, TransformToXML, SetOutputData",
					Functions:      functions,
				},
			},
		},
	}

	appFunctions, err := sdk.LoadConfigurablePipeline()
	assert.NoError(t, err, "")
	assert.NotNil(t, appFunctions, "expected app functions list to be set")
	assert.Equal(t, 3, len(appFunctions))
}

func TestUseTargetTypeOfByteArrayTrue(t *testing.T) {
	functions := make(map[string]common.PipelineFunction)
	functions["CompressWithGZIP"] = common.PipelineFunction{}
	functions["SetOutputData"] = common.PipelineFunction{}

	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Writable: common.WritableInfo{
				Pipeline: common.PipelineInfo{
					ExecutionOrder:           "CompressWithGZIP, SetOutputData",
					UseTargetTypeOfByteArray: true,
					Functions:                functions,
				},
			},
		},
	}

	_, err := sdk.LoadConfigurablePipeline()
	assert.NoError(t, err, "")
	assert.NotNil(t, sdk.TargetType)
	assert.Equal(t, reflect.Ptr, reflect.TypeOf(sdk.TargetType).Kind())
	assert.Equal(t, reflect.TypeOf([]byte{}).Kind(), reflect.TypeOf(sdk.TargetType).Elem().Kind())
}

func TestUseTargetTypeOfByteArrayFalse(t *testing.T) {
	functions := make(map[string]common.PipelineFunction)
	functions["CompressWithGZIP"] = common.PipelineFunction{}
	functions["SetOutputData"] = common.PipelineFunction{}

	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Writable: common.WritableInfo{
				Pipeline: common.PipelineInfo{
					ExecutionOrder:           "CompressWithGZIP, SetOutputData",
					UseTargetTypeOfByteArray: false,
					Functions:                functions,
				},
			},
		},
	}

	_, err := sdk.LoadConfigurablePipeline()
	assert.NoError(t, err, "")
	assert.Nil(t, sdk.TargetType)
}

func TestSetLoggingTargetLocal(t *testing.T) {
	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Logging: common.LoggingInfo{
				EnableRemote: false,
				File:         "./myfile",
			},
		},
	}
	result, err := sdk.setLoggingTarget()
	assert.Nil(t, err, "Should be no error")
	assert.Equal(t, "./myfile", result, "File path is incorrect")
}

func TestSetLoggingTargetRemote(t *testing.T) {
	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Clients: map[string]common.ClientInfo{
				"Logging": common.ClientInfo{
					Protocol: "http",
					Host:     "logs",
					Port:     8080,
				},
			},
			Logging: common.LoggingInfo{
				EnableRemote: true,
			},
		},
	}
	result, err := sdk.setLoggingTarget()
	assert.Nil(t, err, "Should be no error")
	assert.Equal(t, "http://logs:8080/api/v1/logs", result, "File path is incorrect")
}
func TestInitializeClientsAll(t *testing.T) {
	clients := make(map[string]common.ClientInfo)
	clients[common.CoreDataClientName] = common.ClientInfo{}
	clients[common.NotificationsClientName] = common.ClientInfo{}
	clients[common.CoreCommandClientName] = common.ClientInfo{}

	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Clients: clients,
		},
	}

	sdk.initializeClients()

	assert.NotNil(t, sdk.edgexClients.EventClient)
	assert.NotNil(t, sdk.edgexClients.CommandClient)
	assert.NotNil(t, sdk.edgexClients.ValueDescriptorClient)
	assert.NotNil(t, sdk.edgexClients.NotificationsClient)
}

func TestInitializeClientsJustData(t *testing.T) {
	clients := make(map[string]common.ClientInfo)
	clients[common.CoreDataClientName] = common.ClientInfo{}

	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Clients: clients,
		},
	}

	sdk.initializeClients()

	assert.NotNil(t, sdk.edgexClients.EventClient)
	assert.NotNil(t, sdk.edgexClients.ValueDescriptorClient)

	assert.Nil(t, sdk.edgexClients.CommandClient)
	assert.Nil(t, sdk.edgexClients.NotificationsClient)
}

func TestValidateVersionMatch(t *testing.T) {
	clients := make(map[string]common.ClientInfo)
	clients[common.CoreDataClientName] = common.ClientInfo{
		Protocol: "http",
		Host:     "localhost",
		Port:     0, // Will be replaced by local test webserver's port
	}

	sdk := AppFunctionsSDK{
		LoggingClient: lc,
		config: common.ConfigurationStruct{
			Clients: clients,
		},
	}

	tests := []struct {
		Name             string
		CoreVersion      string
		SdkVersion       string
		skipVersionCheck bool
		ExpectFailure    bool
	}{
		{"Compatible Versions", "1.1.0", "v1.0.0", false, false},
		{"Dev Compatible Versions", "2.0.0", "v2.0.0-dev.11", false, false},
		{"Un-compatible Versions", "2.0.0", "v1.0.0", false, true},
		{"Skip Version Check", "2.0.0", "v1.0.0", true, false},
		{"Running in Debugger", "1.0.0", "v0.0.0", false, false},
		{"SDK Beta Version", "1.0.0", "v0.2.0", false, false},
		{"SDK Version malformed", "1.0.0", "", false, true},
		{"Core version malformed", "12", "v1.0.0", false, true},
		{"Core version JSON bad", "", "v1.0.0", false, true},
		{"Core version JSON empty", "{}", "v1.0.0", false, true},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {

			internal.SDKVersion = test.SdkVersion
			sdk.skipVersionCheck = test.skipVersionCheck

			handler := func(w http.ResponseWriter, r *http.Request) {
				var versionJson string
				if test.CoreVersion == "{}" {
					versionJson = "{}"
				} else if test.CoreVersion == "" {
					versionJson = ""
				} else {
					versionJson = fmt.Sprintf(`{"version" : "%s"}`, test.CoreVersion)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(versionJson))
			}

			// create test server with handler
			testServer := httptest.NewServer(http.HandlerFunc(handler))
			defer testServer.Close()

			testServerUrl, _ := url.Parse(testServer.URL)
			port, _ := strconv.Atoi(testServerUrl.Port())
			coreService := sdk.config.Clients[common.CoreDataClientName]
			coreService.Port = port
			sdk.config.Clients[common.CoreDataClientName] = coreService

			result := sdk.validateVersionMatch()
			assert.Equal(t, test.ExpectFailure, !result)
		})
	}
}

type mockEventEndpoint struct {
}

func (e mockEventEndpoint) Monitor(params types.EndpointParams, ch chan string) {
	switch params.ServiceKey {
	case clients.SupportSchedulerServiceKey:
		url := fmt.Sprintf("http://%s:%v%s", "localhost", 48080, params.Path)
		ch <- url
		break
	default:
		ch <- ""
	}
}
