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
package controllers

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/boxproject/appServerGo/utils"
	"github.com/boxproject/appServerGo/models"
	log "github.com/alecthomas/log4go"
	"encoding/json"
	"github.com/boxproject/appServerGo/models/verify"
	"github.com/satori/go.uuid"
	"net/http"
	"strconv"
	"github.com/boxproject/appServerGo/middleware/jwt"
	"net/url"
	"time"
)

func GenFlow(ctx *gin.Context) {
	//appid := ctx.PostForm("app_account_id")
	appid := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	flow := ctx.PostForm("flow")
	sign := ctx.PostForm("sign")
	log.Debug("新建审批流输入", gin.H{"flow": flow, "sign":sign, "token": ctx.PostForm("token")})
	if flow == "" || sign == "" {
		log.Error("GenFlow")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}
	// 获取创建者账号信息
	accInfo, _ := models.GetAccountInfoByAppID(appid)

	// 只有depth=0的节点才有权创建审批流
	log.Info("有权创建审批流,depth = ", accInfo.Depth)
	if accInfo.Depth != 0 {
		log.Error("GenFlow")
		utils.RetError(ctx, BUSINESS_ERROR_CODE+1)
		return
	}

	flowContent := models.FlowInfo{}
	json.Unmarshal([]byte(flow), &flowContent)
	// 校验flow的内容
	approvers := flowContent.ApprovalInfo[0].Approvers
	if flowContent.Period == "" || flowContent.Name == "" || flowContent.ApprovalInfo[0].Require == 0 || approvers[0].Account == "" || approvers[0].AppID == "" || approvers[0].PubKey == "" {
		log.Error("GenFlow")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}
	// 验证签名
	signPass, err := verify.SignInfo(flow, accInfo.Pubkey, sign)

	if err != nil {
		log.Error("创建审批流验证签名", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	log.Info("创建业务流验证签名", signPass)
	if signPass != true {
		log.Error("GenFlow")
		utils.RetError(ctx, ERROR_CODE+5)
		return
	}

	// 是否创建过相同业务流模板
	flowHash := utils.GenHashStr(flow)
	log.Info("新创建的业务流模板hash", flowHash)
	flowExists, err := models.FlowHashExist(flowHash)

	if err != nil {
		log.Error("创建审批流验证是否唯一", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	log.Info("创建审批流模板Hash是否存在", flowHash)

	if flowExists == true {
		log.Error("GenFlow")
		utils.RetError(ctx, BUSINESS_ERROR_CODE+2)
		return
	}

	// 获取该账号对应的注册申请信息
	regInfo, err := models.GetRegistrationByRegID(accInfo.RegID, 1)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("GenFlow")
			utils.RetError(ctx, ERROR_CODE + 3)
			return
		}
		log.Error("创建审批流获取captain信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 向代理服务器上报新增的审批流模板
	pass := utils.AddFlowToServer(flowContent.Name, appid, flow, sign, flowHash, regInfo.CaptainID)
	log.Info("向代理服务器上报新增审批流申请", pass)
	if pass != true {
		// 请求代理服务器错误
		log.Error("GenFlow")
		utils.RetError(ctx, ERROR_CODE+12)
		return
	}

	// 创建业务流模板
	flowID := uuid.Must(uuid.NewV4()).String()
	errorcode, err := models.GenBusinessFlow(flowID, flowHash, flowContent.Name, flow, sign, flowContent.SingleLimit, accInfo.ID, flowContent.FlowLimit, flowContent.Period)

	if err != nil {
		log.Error("创建审批流落库", err)
		utils.RetError(ctx, errorcode)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["GEN_FLOW"], "data": gin.H{"flow_id": flowID}})
}

func GetFlowInfo(ctx *gin.Context) {
	flowID := ctx.Query("flow_id")
	//appID := ctx.Query("app_account_id")
	appID := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	if flowID == "" {
		log.Error("GetFlowInfo")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}

	accInfo, _ := models.GetAccountInfoByAppID(appID)

	//if errorcode != 0 {
	//	utils.RetError(ctx, errorcode, errmsg)
	//	return
	//}

	// 查找某账号所属的根节点账号
	var mAccID int64

	if accInfo.Depth == 0 {
		mAccID = accInfo.ID
	} else {
		managerAccInfo, err := models.GetRootAccountByUnderlingAcc(accInfo.AccLft, accInfo.AccRgt)

		if err != nil {
			if err == sql.ErrNoRows {
				log.Error("GetFlowInfo")
				utils.RetError(ctx, ERROR_CODE+4)
				return
			}
			log.Error("获取审批流信息查找根节点", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
		mAccID = managerAccInfo.ID

	}

	flowInfo, err := models.GetFlowInfoByID(flowID, mAccID)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("GetFlowInfo")
			utils.RetError(ctx, ERROR_CODE+6)
			return
		}
		log.Error("获取审批流信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["FLOW_INFO"], "data": flowInfo})
}

func GetFlowList(ctx *gin.Context) {
	var mAccID int64
	appID := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	paramType := ctx.Query("type")
	keyWords := ctx.Query("key_words")
	limit := ctx.DefaultQuery("limit", "20")
	page := ctx.DefaultQuery("page", "1")
	currencyName := ctx.Query("currency")
	log.Debug("获取审批流列表用户输入", gin.H{"appID": appID, "paramType": paramType, "keywords": keyWords, "limit": limit, "page":page})
	p, _ := strconv.ParseInt(page, 10, 64)
	l, _ := strconv.ParseInt(limit, 10, 64)
	t, _ := strconv.ParseInt(paramType, 10, 64)

	// 获取账号信息
	accInfo, errorcode, errmsg := verify.ValidateUser(appID)

	if errorcode != 0 {
		log.Error("GetFlowList")
		utils.RetError(ctx, errorcode, errmsg)
		return
	}

	// 查找某账号所属的根节点账号

	if accInfo.Depth == 0 {
		mAccID = accInfo.ID
	} else {
		managerAccInfo, err := models.GetRootAccountByUnderlingAcc(accInfo.AccLft, accInfo.AccRgt)

		if err != nil {
			if err == sql.ErrNoRows {
				log.Error("GetFlowList")
				utils.RetError(ctx, ERROR_CODE+4)
				return
			}
			log.Error("获取审批流列表查找根节点", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
		mAccID = managerAccInfo.ID

	}

	// 将除了未审批的审批流外的其他审批流状态更新为2
	err := models.DisableFlows()

	if err != nil {
		log.Error("获取审批流列表更新审批流状态", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 从代理服务器获取已经通过审批的审批流
	updateOK, err := models.UpdateFlowStatusByRPC()
	//updateOK := true
	if err != nil {
		log.Error("获取审批流列表从代理服务器获取已通过的审批流", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if updateOK == false {
		log.Error("GetFlowList")
		utils.RetError(ctx, ERROR_CODE+12)
		return
	}

	// 更新审批流对应的额度信息
	err = models.UpdateFlowAmount(time.Now().Unix())

	if err != nil {
		log.Error("flowlist...")
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	result := models.FlowListInfo{}

	var currencyInfo models.CurrencyInfo
	var currency_id_s string
	if currencyName != "" {
		currencyInfo, err = models.GetCurrencyInfoByName(currencyName)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Error("GetFlowList")
				utils.RetError(ctx, CAPITAL_ERROR_CODE + 2)
				return
			}
			log.Error("flowList", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
		currency_id_s = strconv.FormatInt(currencyInfo.CurrencyID, 10)
	}

	if keyWords != "" {
		result, err = models.SearchFlowByName(keyWords, mAccID, p, l, t, currencyName, currency_id_s)

		if err != nil {
			log.Error("获取审批流列表搜索审批流", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	} else {
		result, err = models.GetFlowList(mAccID, p, l, t, currencyName, currency_id_s)

		if err != nil {
			log.Error("重新获取审批流列表", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	}
	//log.Debug("FLOW_LIST_INFO:%v",result)

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["FLOW_LIST"], "data": result})
}

// 新增部门
func AddBranch(ctx *gin.Context) {
	name := ctx.PostForm("name")
	//appid := ctx.PostForm("appid")
	appid := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	sign := ctx.PostForm("sign")
	log.Info("新增部门用户输入", gin.H{"name": name, "appid": appid, "sign": sign})

	if name == "" || sign == "" {
		log.Error("AddBranch")
		utils.RetError(ctx, ERROR_CODE + 1)
		return
	}

	// 校验账号
	accInfo, _ := models.GetAccountInfoByAppID(appid)

	// 部门是否存在
	branchExists, err := verify.BranchExists(name, accInfo.ID)

	if err != nil {
		log.Error("添加部门校验名称", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 部门已存在
	if branchExists == true {
		log.Error("AddBranch")
		utils.RetError(ctx, BUSINESS_ERROR_CODE + 3)
		return
	}

	// 检测用户账号
	//accInfo, err := models.GetAccountInfoByAppID(appid)

	// 是否有权新增部门
	if accInfo.Depth != 0 {
		log.Error("AddBranch")
		utils.RetError(ctx, ERROR_CODE + 7)
		return
	}

	// 校验签名信息
	signPass, err := verify.SignInfo(name, accInfo.Pubkey, sign)

	if err != nil {
		log.Error("添加部门校验签名", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if signPass != true {
		log.Error("AddBranch")
		utils.RetError(ctx, ERROR_CODE + 5)
		return
	}

	// 新增部门
	err = models.AddBranch(name, accInfo.ID)

	if err != nil {
		log.Error("添加部门落库", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["ADD_BRANCH"]})
}

// 删除或修改部门
func ModifyBranch(ctx *gin.Context) {
	//appid := ctx.PostForm("appid")
	appid := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	originBid := ctx.PostForm("bid")
	newIndex := ctx.PostForm("new_index")
	newName := ctx.PostForm("new_name")
	sign := ctx.PostForm("sign")
	if originBid == "" || sign == "" {
		log.Error("ModifyBranch")
		utils.RetError(ctx, ERROR_CODE + 1)
		return
	}

	// 获取账号信息
	accInfo, _ := models.GetAccountInfoByAppID(appid)
	// 对应的一级员工账号
	rootAccInfo, _ := models.GetRootAccountByUnderlingAcc(accInfo.AccLft, accInfo.AccRgt)
	// 是否有操作权限
	if accInfo.Depth != 0 {
		log.Error("ModifyBranch")
		utils.RetError(ctx, ERROR_CODE + 7)
		return
	}

	// 获取对应部门信息
	branchInfo, err := models.GetBranchInfoByID(originBid)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("ModifyBranch")
			utils.RetError(ctx, BUSINESS_ERROR_CODE + 5)
			return
		}
		log.Error("删除部门获取部门信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if newIndex == "" && newName == "" &&  originBid != "1"{
		// 删除部门
		// 验签
		signPass, err := verify.SignInfo(originBid, accInfo.Pubkey, sign)

		if err != nil {
			log.Error("删除部门", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		if signPass == false {
			log.Error("ModifyBranch")
			utils.RetError(ctx, ERROR_CODE + 5)
			return
		}

		err = models.DelBranch(originBid)

		if err != nil {
			log.Error("删除部门", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}

	} else if newName != ""{
		// 修改部门名称
		// 是否重名
		branchExist, err := verify.BranchExists(newName, accInfo.ID)

		if err != nil {
			log.Error("修改部门名称重名", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		if branchExist == true {
			log.Error("ModifyBranch")
			utils.RetError(ctx, BUSINESS_ERROR_CODE + 3)
			return
		}

		// 验签
		signPass, err := verify.SignInfo(newName, accInfo.Pubkey, sign)

		if err != nil {
			log.Error("修改部门名称", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		if signPass == false {
			log.Error("ModifyBranch")
			utils.RetError(ctx, ERROR_CODE + 5)
			return
		}

		err = models.ChangeBranchName(originBid, newName)

		if err != nil {
			log.Error("修改部门名称", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	} else if newIndex != "" {
		// 调整部门顺序
		log.Debug("调整部门顺序", gin.H{"bid":  originBid, "newIndex": newIndex})
		// 验签
		//signPass, err := verify.SignInfo(newIndex, accInfo.Pubkey, sign)
		//
		//if err != nil {
		//	log.Error("调整部门顺序", err)
		//	utils.RetError(ctx, ERROR_CODE)
		//	return
		//}
		//
		//if signPass == false {
		//	log.Error("ModifyBranch")
		//	utils.RetError(ctx, ERROR_CODE + 5)
		//	return
		//}

		ind, _ := strconv.ParseInt(newIndex, 10, 64)
		err = models.ChangeBranchIndex(ind, branchInfo)

		if err != nil {
			log.Error("调整部门顺序", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	}

	// 获取部门列表
	list, err := models.GetBranchList(rootAccInfo.ID, rootAccInfo.AccLft, rootAccInfo.AccRgt)

	if err != nil {
		log.Error("获取修改后的部门列表", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["DATA"], "data": gin.H{"list": list}})
}

// 获取部门列表
func BranchList(ctx *gin.Context) {
	appid := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	// 账号信息
	accInfo, err := models.GetAccountInfoByAppID(appid)

	if err != nil {
		log.Error("department...")
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 获取对应根节点账号信息
	rootAccInfo, err := models.GetRootAccountByUnderlingAcc(accInfo.AccLft, accInfo.AccRgt)

	if err != nil {
		log.Error("department rootacc...")
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 获取部门列表
	list, err := models.GetBranchList(rootAccInfo.ID, rootAccInfo.AccLft, rootAccInfo.AccRgt)

	if err != nil {
		log.Error("获取修改后的部门列表", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["BRANCH_LIST"], "data": gin.H{"list": list}})
}

// 获取部门详情
func GetBranchInfo(ctx *gin.Context) {
	//appid := ctx.Query("appid")
	appid := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	branchId := ctx.Query("bid")

	if branchId == "" {
		log.Error("GetBranchInfo")
		utils.RetError(ctx, ERROR_CODE + 1)
		return
	}


	// 账号信息
	accInfo, err := models.GetAccountInfoByAppID(appid)

	if err != nil {
		log.Error("department...")
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	// 获取对应根节点账号信息
	rootAccInfo, err := models.GetRootAccountByUnderlingAcc(accInfo.AccLft, accInfo.AccRgt)

	if err != nil {
		log.Error("department rootacc...")
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 获取部门信息
	branchInfo, err := models.GetBranchInfoByID(branchId)

	if err != nil {
		if err ==  sql.ErrNoRows {
			log.Error("GetBranchInfo")
			utils.RetError(ctx, BUSINESS_ERROR_CODE + 5)
			return
		}
		log.Error("GetBranchInfo")
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 获取对应部门的员工列表
	employees, err := models.GetAccountsListByBid(branchId, rootAccInfo.AccLft, rootAccInfo.AccRgt)

	if err != nil {
		log.Error("获取部门详情", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["BRANCH_INFO"], "data": gin.H{"Name": branchInfo.Name, "Employees": len(employees), "EmployeesList": employees}})
}

// 作废审批流
func DisuseBusinessFlow(ctx *gin.Context) {
	appid := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	flowID := ctx.PostForm("flow_id")
	sign := ctx.PostForm("sign")
	password := ctx.PostForm("password")
	if flowID == "" || sign == "" || password == "" {
		log.Error("DisuseBusinessFlow")
		utils.RetError(ctx, ERROR_CODE + 1)
		return
	}

	// 校验密码
	validatePwd, err := verify.SignPassword(appid, password)
	if err != nil {
		log.Error("申请提现验证密码", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 获取申请者账号信息
	accInfo, _ := models.GetAccountInfoByAppID(appid)

	if validatePwd == false {
		frozenAcc, attempts, err := models.AttemptFrozen(appid)
		if err != nil {
			log.Error("DisuseBusinessFlow")
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		if frozenAcc == true {
			// 账号被冻结
			var frozenTo = accInfo.FrozenTo
			if accInfo.FrozenTo == 0 {
				frozenTo = time.Now().Add(models.FROZEN_HOUR*time.Hour).Unix()
			}
			data := map[string]string{"frozenTo":strconv.FormatInt(frozenTo, 10)}
			log.Error("DisuseBusinessFlow")
			utils.RetError(ctx, ERROR_CODE + 18, data)
			return
		} else {
			log.Error("DisuseBusinessFlow")
			utils.RetError(ctx, ERROR_CODE + 16, map[string]string{"attempts": strconv.Itoa(attempts), "frozenFor": strconv.FormatInt(models.FROZEN_HOUR, 10)})
			return
		}
	}

	// 重置用户尝试密码次数
	err = models.ResetAttempts(appid)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.RetError(ctx, ERROR_CODE + 6)
			return
		}
		log.Error("作废审批流", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 获取审批流内容
	flowInfo, err := models.GetBusinessFlowInfoByFlowID(flowID, 1)

	if err != nil {
		log.Error("disallow", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 验证签名信息
	signPass, err := verify.SignInfo(flowInfo.Name, accInfo.Pubkey, sign)

	if err != nil {
		log.Error("作废审批流验签", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if signPass == false {
		utils.RetError(ctx, ERROR_CODE + 5)
		return
	}

	// 被审批通过的审批流才可以作废
	if flowInfo.Progress != models.FLOW_APPROVALED {
		log.Error("审批流尚未通过审批流", flowInfo.Progress)
		utils.RetError(ctx, BUSINESS_ERROR_CODE + 6)
		return
	}

	// 是否有权作废
	if flowInfo.FounderID != accInfo.ID {
		utils.RetError(ctx, ERROR_CODE + 7)
		return
	}

	// 向代理服务器申请作废审批流
	rpcParam := url.Values{"appid": {appid}, "sign": {sign}, "hash": {flowInfo.Hash}}
	log.Debug("向代理服务器申请作废审批流", rpcParam)
	err = utils.ApplyDisuseFlow(rpcParam.Encode())

	if err != nil {
		log.Error("disuse...请求代理服务器", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	// 作废审批流
	err = models.DisuseFlow(flowID)

	if err != nil {
		log.Error("作废审批流，更新审批流状态", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 作废对应的转账
	err = models.InvalidTx(flowInfo.ID)

	if err != nil {
		log.Error("作废审批流，更新转账状态", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["DATA"]})

}

// 审批流操作日志
func FlowOperationHistory(ctx *gin.Context) {
	flowID := ctx.Query("flow_id")

	if flowID == "" {
		log.Error("FlowOperationHistory")
		utils.RetError(ctx, ERROR_CODE + 1)
		return
	}

	// 获取对应审批流信息
	flowInfo, err := models.GetBusinessFlowInfoByFlowID(flowID, 1)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("FlowOperationHistory")
			utils.RetError(ctx, ERROR_CODE + 6)
			return
		}
		log.Error("FlowOperationHistory")
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 从代理服务器获取审批流操作日志
	data, err := utils.GetFlowOperationLogFromRPC(flowInfo.Hash)

	if err != nil {
		log.Error("FlowOperationHistory err: %v",err,utils.RetError(ctx, ERROR_CODE))
		log.Error("retError:%v",err,utils.RetError(ctx, ERROR_CODE))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["DATA"], "data": gin.H{"HashOperates": data.HashOperates}})
}
