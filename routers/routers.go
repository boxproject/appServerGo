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
package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/boxproject/appServerGo/config"
	"github.com/boxproject/appServerGo/controllers"
	"github.com/ekyoung/gin-nice-recovery"
	"github.com/boxproject/appServerGo/middleware"
	JWT "github.com/boxproject/appServerGo/middleware/jwt"
)

func InitRouter() *gin.Engine {
	cfg := config.GetConfig()
	router := gin.Default() //获得路由实例
	// 处理未被捕捉的错误
	router.Use(nice.Recovery(middleware.RecoveryHandler))
	//router.Use(controllers.ValidateUser())
	app := router.Group("/api/" + cfg.Server.APIVersion)
	// 1-下属注册提交扫码后的信息
	app.POST("/registrations", controllers.ApplyForAccount)
	// 3-下属获取注册申请审批结果
	app.GET("/registrations/approval/result", controllers.RegistrationApprovalInfo)
	// 5-员工APP反馈上级审批结果出错
	app.POST("/registrations/approval/cancel", controllers.CancelApprovalRegistration)
	// 6-用户登录
	app.POST("/accounts/login", controllers.Login)
	// 代理服务器上报转账结果
		// a1-临时结果
	app.POST("/capital/withdraw/id", controllers.WithdrawResultOfID)
		// a2-最终结果
	app.POST("/capital/withdraw", controllers.WithdrawResult)
	// a3-代理服务器上报充值记录
	app.POST("/capital/deposit", controllers.DepositSuccess)
	// a4-代理服务器上报私钥APP审批注册结果
	app.POST("/registrations/admin/approval", controllers.AdminApprovalRegistration)
	// 代理服务器通知新增币种，代币
	app.POST("/capital/curency/add", controllers.AddCurrency)
	// a5-代理服务器获取交易流水
	app.GET("/history/trade", controllers.TradeHistoryForAgent)
	// a6-代理服务器获取账户资金信息
	app.GET("/capital/assets", controllers.AssetForAgent)
	// 校验token
	app.Use(JWT.JWTAuth())
	// 2-上级APP获取待审核的注册信息
	app.GET("/registrations/pending", controllers.RegistrationInfo)
	// 4-上级APP审批下级的注册申请
	app.POST("/registrations/approval", controllers.ApprovalRegistration)
	// 7-修改密码
	app.POST("/accounts/passwords/modify", controllers.ChangePassword)
	// 8-根节点获取指定非直属下属的公钥信息
	app.GET("/employee/pubkeys/info", controllers.GetEmployeePubKeyInfo)
	// 9-上级管理员获取下属员工账号详情
	app.GET("/accounts/info", controllers.GetEmployeeAccountsInfo)
	// 10-删除/替换员工账号
	app.POST("/employee/account/change", controllers.ChangeEmployeeAccount)
	// 11-获取余额
	app.GET("/capital/balance", controllers.GetBalanceList)
	//app.Use(controllers.ValidateUser())
	// 12-提交转账申请
	app.POST("/transfer/application", controllers.ApplyTransfer)
	// 13-获取转账记录列表(待审批/已审批、作为发起者/作为审批者)
	app.GET("/transfer/records/list", controllers.GetTransferRecordsList)
	// 14-获取指定的转账记录详情
	app.GET("/transfer/records", controllers.GetTransInfoByOrderNumber)
	// 15-提交审批意见
	app.POST("/transfer/approval", controllers.ApprovalTransfer)
	// 16-获取业务流模板列表
	app.GET("/business/flows/list", controllers.GetFlowList)
	// 17-获取业务流模板详情
	app.GET("/business/flow/info", controllers.GetFlowInfo)
	// 18-根节点获取非直属下属的公钥信息列表
	app.GET("/employee/pubkeys/list", controllers.GetEmployeePubKeyInfoList)
	// 19-上级管理员获取下属员工账号列表
	app.GET("/accounts/list", controllers.GetEmployeeAccountsList)
	// 20-创建业务流模板
	app.POST("/business/flow", controllers.GenFlow)
	// 21-获取币种列表
	app.GET("/capital/currency/list", controllers.GetCurrencyList)
	// 22-获取交易记录列表
	app.GET("/capital/trade/history/list", controllers.GetTradeHistoryList)
	// 23-添加部门
	app.POST("/branch/add", controllers.AddBranch)
	// 24-删除或修改部门
	app.POST("/branch/change", controllers.ModifyBranch)
	// 25-获取部门列表
	app.GET("/branch/list", controllers.BranchList)
	// 26-修改账号所属部门
	app.POST("/branch/select", controllers.SelectBranch)
	// 27-获取部门详情
	app.GET("/branch/info", controllers.GetBranchInfo)
	// 28-获取账号详情
	app.GET("/accounts/detail", controllers.GetAccountDetail)
	// 29-撤回转账申请
	app.POST("/transfer/application/cancel", controllers.CancelTransferApply)
	// 30-作废审批流
	app.POST("/business/flow/disuse", controllers.DisuseBusinessFlow)
	// 31-转账操作日志
	app.GET("/history/transfer/operation", controllers.TxOperationHistory)
	// 32-审批流操作日志
	app.GET("/history/flow/operation", controllers.FlowOperationHistory)
	return router
}
