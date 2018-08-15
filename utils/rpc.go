// Copyright 2018. box.la authors.
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
package utils

import (
	"net/http"
	"crypto/tls"
	"net/url"
	"github.com/boxproject/appServerGo/config"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/alecthomas/log4go"
)

var Client *http.Client

func init() {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	Client = &http.Client{Transport: tr}
}

// 向代理服务器请求注册
func RegToServer(regid, msg, applicantID, captainID, applicantAccount string, status int) (err error) {
	data := url.Values{"regid": {regid}, "msg": {msg}, "applyerid": {applicantID}, "captainid": {captainID}, "applyeraccount": {applicantAccount}, "status": {string(status)}}
	cfg := config.GetConfig()
	proxy_info := cfg.Agent
	cRsp := &RPCRsp{}
	log.Info("向代理服务器注册", data.Encode())
	body, err := HttpRequest("POST", proxy_info.Server+proxy_info.Registration, data.Encode())

	if err = json.Unmarshal(body, cRsp); err != nil {
		return
	}

	log.Info("请求代理服务器注册，返回值", cRsp)

	if cRsp.RspNo != "0" {
		err = errors.New(fmt.Sprintf("RPC ERROR Code; %s", cRsp.RspNo))
		return
	}
	return
}

// 获取业务流结构上链状态
func BusinessFlowStatus(flowHash string) (status int, err error) {
	cfg := config.GetConfig()
	status = 2
	proxy_info := cfg.Agent
	body, err := HttpRequest("GET", proxy_info.Server+proxy_info.FlowStatus+"?hash="+flowHash, "")

	// 解析RPC返回的数据
	cRsp := &ResHashStatus{}
	if err = json.Unmarshal(body, cRsp); err != nil {
		return
	}
	if cRsp.RspNo != "0" {
		err = errors.New(fmt.Sprintf("RPC ERROR Code; %s", cRsp.RspNo))
		return
	} else {
		if cRsp.ApprovalInfo.Status == "7" {
			status = 3
			err  = nil
		}
		if cRsp.ApprovalInfo.Status == "0" || cRsp.ApprovalInfo.Status == "1" || cRsp.ApprovalInfo.Status == "3" || cRsp.ApprovalInfo.Status == "4" || cRsp.ApprovalInfo.Status == "6" {
			status = 1
			err = nil
		}
	}
	return
}

// 向代理服务器申请创建审批流
func AddFlowToServer(flowName, appid, flow, sign, flowHash, captainID string) bool {
	data := url.Values{"name": {flowName}, "appid": {appid}, "flow": {flow}, "sign": {sign}, "hash": {flowHash}, "captainid": {captainID}}
	cfg :=  config.GetConfig()
	proxy_info := cfg.Agent
	cRsp := &RPCRsp{}
	body, err := HttpRequest("POST", proxy_info.Server+proxy_info.AddFlow, data.Encode())
	if err = json.Unmarshal(body, cRsp); err != nil {
		return false
	}

	if cRsp.RspNo != "0" {
		err = errors.New(fmt.Sprintf("RPC ERROR Code; %s", cRsp.RspNo))
		return false
	}
	return true
}

// 向代理服务器发起转账申请
func ApplyTx(param string) error {
	cfg := config.GetConfig()
	url := cfg.Agent.Server + cfg.Agent.ApplyTx
	data, err := HttpRequest("POST", url, param)
	cRsp := &RPCRsp{}
	if err != nil {
		return err
	}

	if err = json.Unmarshal(data, cRsp); err != nil {
		return err
	}

	if cRsp.RspNo != "0" {
		return errors.New("申请转账失败")
	}
	return nil
}

// 向代理服务器申请作废审批流
func ApplyDisuseFlow(param string) error {
	cfg := config.GetConfig()
	url := cfg.Agent.Server + cfg.Agent.DisallowFlow
	data, err := HttpRequest("POST", url, param)

	cRsp := &RPCRsp{}
	if err != nil {
		return err
	}

	if err = json.Unmarshal(data, cRsp); err != nil {
		return err
	}

	if cRsp.RspNo != "0" {
		return errors.New("作废审批流失败")
	}
	return nil
}

// 向代理服务器获取审批流操作日志

func GetFlowOperationLogFromRPC(flow_hash string) (ResFlowOpLog, error) {
	cfg := config.GetConfig()
	url := cfg.Agent.Server + cfg.Agent.FlowOpLog+"?hash="+flow_hash
	log.Debug("GetFlowOperationLogFromRPC URL:[%v]",url)
	body, err := HttpRequest("GET", url, "")
	// 解析RPC返回的数据
	cRsp := ResFlowOpLog{}
	if err = json.Unmarshal(body, &cRsp); err != nil {
		return ResFlowOpLog{}, err
	}
	if cRsp.RspNo != "0" {
		err = errors.New(fmt.Sprintf("RPC ERROR Code: %s", cRsp.RspNo))
		return ResFlowOpLog{}, err
	}

	return cRsp, nil
}
