// Copyright 2017 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package stdioLogger provides an implementation of Mixer's logger aspect that
// writes logs (serialized as JSON) to a standard stream (stdout | stderr).
package l7collector

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"istio.io/mixer/adapter/l7collector/config"
	"istio.io/mixer/pkg/adapter"

	"github.com/golang/glog"
	"net/http"
	"time"
	"os"
	"strings"
	"bytes"
	"io/ioutil"
	"encoding/json"
	"fmt"
)

const(
	GW_API_URL = "http://169.46.62.146:9000/"
	TENANT_ID_ENV = "TENANT_ID"
	CRN_ENV = "RESOURCE_GROUP_CRN"
	URI = "api/v1/tenant/{tenantId}/resourceGroup/{CRN}/collection/{collectionId}"
	REQUEST_TIMEOUT = 10 * time.Second
	DATAGW_SERVICE_PORT_ENV = "DATA_GATEWAY_SERVICE_PORT"
	DATAGW_SERVICE_HOST_ENV = "DATA_GATEWAY_SERVICE_HOST"
)

var collectionIDs = map[string]string {
	"internal": "internal",
	"ingress": "ingress",
	"external": "external",
}

type (
	builder struct{ adapter.DefaultBuilder }
	collector  struct {
		env adapter.Env
		omitEmpty   bool
	}
	zapBuilderFn func(outputPaths ...string) (*zap.Logger, error)
)


var (
	zapConfig = zapcore.EncoderConfig{
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}
)

// Register records the builders exposed by this adapter.
func Register(r adapter.Registrar) {
	b := builder{adapter.NewDefaultBuilder(
		"l7collector",
		"Collects L7 data form istio and sends it to a specified API",
		&config.Params{},
	)}

	r.RegisterApplicationLogsBuilder(b)
	r.RegisterAccessLogsBuilder(b)
}

func (builder) NewApplicationLogsAspect(env adapter.Env, cfg adapter.Config) (adapter.ApplicationLogsAspect, error) {
	return l7collector(cfg, newZapLogger, env)
}

func (builder) NewAccessLogsAspect(env adapter.Env, cfg adapter.Config) (adapter.AccessLogsAspect, error) {
	return l7collector(cfg, newZapLogger, env)
}

func l7collector(cfg adapter.Config, buildZap zapBuilderFn, env adapter.Env) (*collector, error) {
	c := cfg.(*config.Params)
	/*
	if err != nil {
		return nil, fmt.Errorf("could not build l7collector: %v", err)
	}
*/
	return &collector{env: env, omitEmpty:c.OmitEmptyFields},nil
}

func (l *collector) Log(entries []adapter.LogEntry) error {
	return nil
}

func (l *collector) LogAccess(entries []adapter.LogEntry) error {
	memos,memoCollectionIDs,err := l.get_current_memos(entries)
	if err != nil{
		glog.Warningf("request resulted in an error when creating memo: %v",err.Error())
		return err
	}
	for i, memo := range memos{
		err = dataGWCollectPost(memoCollectionIDs[i],memo)
		if err != nil{
			glog.Errorf(err.Error())
		}
	}
	return nil
}

func (l *collector) get_current_memos(entries []adapter.LogEntry) ([][]byte, []string, error) {
	memos := make([][]byte,0)
	memoCollectionIDs := make([]string,0)
	includeEmpty := !l.omitEmpty
	for _, entry := range entries {
		memo := make([]map[string]interface{},1)
		memo[0] = make(map[string]interface{})
		if includeEmpty || entry.LogName != "" {
			memo[0]["logName"] = entry.LogName
		}
		if includeEmpty || entry.Timestamp != "" {
			memo[0]["timeStamp"] = entry.Timestamp
		}
		if includeEmpty || entry.Severity != adapter.Default {
			memo[0]["severity"] = entry.Severity.String()
		}
		if includeEmpty || len(entry.Labels) > 0 {
			memo[0]["labels"] = entry.Labels
			if _, exists := entry.Labels["source.name"] ; !exists{
				memoCollectionIDs = append(memoCollectionIDs, collectionIDs["ingress"])
			} else if _, exists := entry.Labels["target.name"] ; !exists {
				memoCollectionIDs = append(memoCollectionIDs, collectionIDs["external"])
			} else {
				memoCollectionIDs = append(memoCollectionIDs, collectionIDs["internal"])
			}
		}
		if includeEmpty || len(entry.TextPayload) > 0 {
			memo[0]["textPayload"] = entry.TextPayload
		}
		if includeEmpty || len(entry.StructPayload) > 0 {
			memo[0]["structPayload"] = entry.StructPayload
		}
		byt, _ := json.Marshal(memo)
		memos = append(memos, byt)
	}

	return memos,memoCollectionIDs,nil
}


/*
func (l *collector) get_current_collectionID(entries []adapter.LogEntry) (string,error) {
	includeEmpty := !l.omitEmpty
	for _, entry := range entries {
		if includeEmpty || len(entry.Labels) > 0 {
			//
		}
	}
	byt, _ := json.Marshal(memo)
	return string(byt),nil
}
*/

func (l *collector) Close() error { return nil }

func newZapLogger(outputPaths ...string) (*zap.Logger, error) {
	prodConfig := zap.NewProductionConfig()
	prodConfig.DisableCaller = true
	prodConfig.DisableStacktrace = true
	prodConfig.EncoderConfig = zapConfig
	prodConfig.OutputPaths = outputPaths
	zapLogger, err := prodConfig.Build()
	return zapLogger, err
}

func createDataGWURI(tenantID string, crn string, collectionID string) string{
	s := URI
	s = strings.Replace(s,"{tenantId}", tenantID, 1)
	s = strings.Replace(s,"{CRN}", crn, 1)
	s = strings.Replace(s,"{collectionId}", collectionID, 1)
	return s
}

func dataGWCollectGet(collectionID string) error{
	tenantID := os.Getenv(TENANT_ID_ENV)
	if tenantID == ""{
		return fmt.Errorf("l7Collector Error: %v env var undefined",TENANT_ID_ENV)
	}
	crn := os.Getenv(CRN_ENV)
	if crn == ""{
		return fmt.Errorf("l7Collector Error: %v env var undefined",CRN_ENV)
	}
	host := os.Getenv(DATAGW_SERVICE_HOST_ENV)
	if host == ""{
		return fmt.Errorf("l7Collector Error: %v env var undefined",DATAGW_SERVICE_HOST_ENV)
	}
	port := os.Getenv(DATAGW_SERVICE_PORT_ENV)
	if port == ""{
		return fmt.Errorf("l7Collector Error: %v env var undefined",DATAGW_SERVICE_PORT_ENV)
	}

	client := &http.Client{
		Timeout: REQUEST_TIMEOUT,
	}

	url := "http://"+host+":"+port+"/"+createDataGWURI(tenantID,crn,collectionID)

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	glog.Infof("response Status:", resp.Status)
	glog.Infof("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	glog.Infof("response Body:", string(body))
	return nil
}

func dataGWCollectPost(collectionID string, memo []byte) error{

	tenantID := os.Getenv(TENANT_ID_ENV)
	if tenantID == ""{
		return fmt.Errorf("l7Collector Error: %v env var undefined",TENANT_ID_ENV)
	}
	crn := os.Getenv(CRN_ENV)
	if crn == ""{
		return fmt.Errorf("l7Collector Error: %v env var undefined",CRN_ENV)
	}
	host := os.Getenv(DATAGW_SERVICE_HOST_ENV)
	if host == ""{
		return fmt.Errorf("l7Collector Error: %v env var undefined",DATAGW_SERVICE_HOST_ENV)
	}
	port := os.Getenv(DATAGW_SERVICE_PORT_ENV)
	if port == ""{
		return fmt.Errorf("l7Collector Error: %v env var undefined",DATAGW_SERVICE_PORT_ENV)
	}

	url := "http://"+host+":"+port+"/"+createDataGWURI(tenantID,crn,collectionID)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(memo))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: REQUEST_TIMEOUT,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	glog.Infof("response Status:", resp.Status)
	glog.Infof("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	glog.Infof("response Body:", string(body))

	return nil
}
