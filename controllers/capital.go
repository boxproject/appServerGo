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
	"github.com/boxproject/appServerGo/models/verify"
	log "github.com/alecthomas/log4go"
	"encoding/json"
	"strconv"
	"github.com/satori/go.uuid"
	"net/http"
	"net/url"
	"math"
	"github.com/boxproject/appServerGo/middleware/jwt"
	"time"
	"strings"
)

// 提交转账申请
func ApplyTransfer(ctx *gin.Context) {
	appid := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	content := ctx.PostForm("apply_info")
	flowID := ctx.PostForm("flow_id")
	signature := ctx.PostForm("sign")
	password := ctx.PostForm("password")

	log.Debug("申请转账用户输入", gin.H{"appid": appid, "apply_info": content, "flow_id": flowID, "sign": signature})
	var order_num string

	if content == "" || flowID == "" || signature == "" || password == "" {
		log.Error("ApplyTransfer")
		utils.RetError(ctx, ERROR_CODE+1)
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
	applyerAccInfo, _ := models.GetAccountInfoByAppID(appid)

	if validatePwd == false {
		frozenAcc, attempts, err := models.AttemptFrozen(appid)
		if err != nil {
			log.Error("ApplyTransfer")
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		if frozenAcc == true {
			// 账号被冻结
			var frozenTo = applyerAccInfo.FrozenTo
			if applyerAccInfo.FrozenTo == 0 {
				frozenTo = time.Now().Add(models.FROZEN_HOUR*time.Hour).Unix()
			}
			data := map[string]string{"frozenTo":strconv.FormatInt(frozenTo, 10)}
			log.Error("ApplyTransfer")
			utils.RetError(ctx, ERROR_CODE + 18, data)
			return
		} else {
			log.Error("ApplyTransfer")
			utils.RetError(ctx, ERROR_CODE + 16, map[string]string{"attempts": strconv.Itoa(attempts), "frozenFor": strconv.FormatInt(models.FROZEN_HOUR, 10)})
			return
		}
	}

	// 重置用户尝试密码次数
	err = models.ResetAttempts(appid)


	// 验证签名信息
	signPass, err := verify.SignInfo(content, applyerAccInfo.Pubkey, signature)
	log.Info("提交转账申请验签", signPass)
	if err != nil {
		log.Error("申请转账验签错误", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if signPass == false {
		log.Error("ApplyTransfer")
		utils.RetError(ctx, ERROR_CODE+5)
		return
	}

	// 解析转账内容
	var transferContent = models.TransferContent{}
	err = json.Unmarshal([]byte(content), &transferContent)
	if err != nil {
		log.Error("申请转账解析转账内容", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if transferContent.Amount == "" || transferContent.ApplyTimestamp == "" || transferContent.Currency == "" || transferContent.Miner == "" || transferContent.ToAddr == "" || transferContent.TxInfo == "" {
		log.Error("ApplyTransfer")
		utils.RetError(ctx, CAPITAL_ERROR_CODE+1)
		return
	}

	// 获取币种信息
	currencyInfo, err := models.GetCurrencyInfoByName(transferContent.Currency)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("ApplyTransfer")
			utils.RetError(ctx, CAPITAL_ERROR_CODE + 11)
			return
		}
		log.Error("申请转账获取币种信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	// 转账金额单位换算
	amount_f, err := strconv.ParseFloat(transferContent.Amount, 64)
	factor, err := strconv.ParseInt(currencyInfo.Factor, 10, 64)
	amount_u, err := utils.UnitReConversion(amount_f, factor, 10)
	// 转账金额不能小于精度
	leagelAmount := utils.CompareFloat64(amount_u, math.Ceil(amount_u))

	if leagelAmount != 0 {
		log.Error("校验转账金额精度", amount_u)
		utils.RetError(ctx, CAPITAL_ERROR_CODE + 1)
		return
	}

	// 转账地址 from != to
	correctAddr, err := verify.FromIsTo(transferContent.ToAddr)

	if err != nil {
		log.Error("ApplyTransfer")
		log.Error("申请转账from=to")
		return
	}

	if correctAddr == true {
		utils.RetError(ctx, CAPITAL_ERROR_CODE+1)
		return
	}

	//balance_f, err := strconv.ParseFloat(currencyInfo.Balance, 64)

	if err != nil {
		log.Error("申请转账余额转换", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	//// 是否余额不足
	//if utils.GreaterThanFloat64(amount_f, balance_f) {
	//	log.Error("申请转账余额不足")
	//	utils.RetError(ctx, CAPITAL_ERROR_CODE+9)
	//	return
	//}

	// 获取对应的审批流
	flowInfo, err := models.GetBusinessFlowInfoByFlowID(flowID, 1)
	log.Debug("flowInfo:%v", flowInfo)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("ApplyTransfer")
			utils.RetError(ctx, ERROR_CODE+6)
			return
		}
		log.Error("申请转账获取对应审批流", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 获取审批流对应币种的限额
	var flow_limit string
	if flowInfo.SingleLimit != ""{
		flow_limit = flowInfo.SingleLimit
	} else if flowInfo.FlowLimit != nil {
		for i:=0; i<len(flowInfo.FlowLimit);i++ {
			currencyName := strings.Replace(flowInfo.FlowLimit[i].CurrencyName, " ", "", -1)
			if currencyName == currencyInfo.Currency {
				flow_limit = flowInfo.FlowLimit[i].Limit
				break
			}
		}
		// 审批流不支持对应币种
		if flow_limit == "" {
			log.Error("ApplyTransfer")
			utils.RetError(ctx, CAPITAL_ERROR_CODE + 11)
			return
		}
	}

	// 检测是否超过单笔上限
	singleLimit, _ := strconv.ParseFloat(flow_limit, 64)
	log.Debug("flow_limit", flow_limit)
	if utils.GreaterThanFloat64(amount_f, singleLimit) && singleLimit != 0 {
		log.Error("singleLimit")
		utils.RetError(ctx, CAPITAL_ERROR_CODE+10)
		return
	}

	// 获取剩余额度
	amountLeft, err := models.TransferLimitByCurrencyID(currencyInfo.CurrencyID, flowInfo.ID)
	if err != nil {
		log.Error("申请转账获取额度", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	log.Debug("amountLeftInfo:%v",amountLeft)

	if amountLeft.AmountLeft == "-1" {
		// 额度未使用
		amount_left := utils.SubFloat64(singleLimit, amount_f)
		// 初始化额度
		period, _ := strconv.Atoi(flowInfo.Period)
		err = models.InitTxAmount(currencyInfo.CurrencyID, time.Now().Add(time.Duration(period)*time.Hour).Unix(), flowInfo.ID, amount_left)
		if err != nil {
			log.Error("初始化额度")
			utils.RetError(ctx, ERROR_CODE)
			return
		}

	} else {
		log.Debug("flowInfo.Period:%v",flowInfo.Period)
		if flowInfo.Period != "0"  {
			log.Debug("amountLeft.FrozenTo:%v,time Now:%v",amountLeft.FrozenTo,time.Now().Unix())
			// 更新额度时间
			if (amountLeft.FrozenTo != 0 && amountLeft.FrozenTo <= time.Now().Unix()){
				models.ResetTxAmount(currencyInfo.CurrencyID, flowInfo.ID, singleLimit)
			}

			// 重新获取额度信息
			new_amountLeft, err := models.TransferLimitByCurrencyID(currencyInfo.CurrencyID, flowInfo.ID)
			// 额度被冻结
			if new_amountLeft.Frozen == true {
				log.Error("申请转账额度已用完")
				utils.RetError(ctx, CAPITAL_ERROR_CODE + 12, map[string]string{"frozenTo": strconv.FormatInt(amountLeft.FrozenTo, 10)})
				return
			}

			// 判断剩余额度
			amt_lft, _ := strconv.ParseFloat(new_amountLeft.AmountLeft, 64)

			if utils.LessThanFloat64(amt_lft, amount_f) {
				log.Error("申请转账额度不足", amt_lft)
				utils.RetError(ctx, CAPITAL_ERROR_CODE+10, map[string]string{"amountLeft": new_amountLeft.AmountLeft, "frozenTo": strconv.FormatInt(new_amountLeft.FrozenTo, 10)})
				return
			}

			// 更新额度
			recent_amt_lft, _ := strconv.ParseFloat(new_amountLeft.AmountLeft, 64)
			err = models.UpdateTxAmount(currencyInfo.CurrencyID, flowInfo.ID, utils.SubFloat64(recent_amt_lft, amount_f), flowInfo.Period)

			if err != nil {
				log.Error("申请转账更新额度", err)
				utils.RetError(ctx, ERROR_CODE)
				return
			}
		}
	}


	// 查询审批流上链状态
	flow_on_chain_status, err := utils.BusinessFlowStatus(flowInfo.Hash)

	if err != nil {
		log.Error("申请提现，查询对应审批流状态", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	log.Info("提交转账申请获取审批流上链状态", gin.H{"flowHash": flowInfo.Hash, "status": flow_on_chain_status})
	// 更新审批流状态
	err = models.UpdateFlowStatus(flowInfo.ID, flow_on_chain_status)

	if err != nil {
		log.Error("申请提现更新审批流状态", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if flow_on_chain_status == 3 {
		// 提交转账申请
		order_num = uuid.Must(uuid.NewV4()).String()
		// 提交转账申请
		err = models.ApplyTransfer(order_num, content, signature, flowInfo, transferContent, applyerAccInfo, currencyInfo)

		if err != nil {
			log.Error("申请转账落库", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	}  else {
		log.Error("ApplyTransfer")
		utils.RetError(ctx, ERROR_CODE+6)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["APPLY_TRANSFER"], "data": gin.H{"order_number": order_num}})
}

// 获取转账列表
func GetTransferRecordsList(ctx *gin.Context) {
	//claims := ctx.MustGet("claims").(*jwt.CustomClaims)
	appid := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	tx_type := ctx.DefaultQuery("type", "0")
	progress := ctx.DefaultQuery("progress", "0")
	page := ctx.DefaultQuery("page", "1")
	limit := ctx.DefaultQuery("limit", "20")
	// 获取账号信息
	accountInfo, _ := models.GetAccountInfoByAppID(appid)

	p, _ := strconv.ParseInt(progress, 10, 64)
	
	log.Debug("GetTransferRecordsList,appid:%v,accountInfo:%v,type:%v,progress:%v,page:%v,limit:%v",appid,accountInfo,tx_type,progress,page,limit)
	// 申请者获取列表
	var result models.TransferRecordInfo
	var err error
	if tx_type == "0" {
		result, err = models.GetTransferRecordsListByApplyerID(accountInfo.ID, p, page, limit)
		if err != nil {
			log.Error("作为申请者获取转账列表", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	} else {
		result, err = models.GetTransferRecordListByCaptainID(accountInfo.ID, p, page, limit)
		if err != nil {
			log.Error("作为审批者获取转账列表", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	}
	log.Debug("GetTransferRecordsList:%v",result)

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["TRANSFER_LIST"], "data": result})
}

// 根据订单号获取指定转账记录
func GetTransInfoByOrderNumber(ctx *gin.Context) {
	ordernum := ctx.Query("order_number")
	if ordernum == "" {
		log.Error("GetTransInfoByOrderNumber")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}

	// 获取转账信息
	flowID, txInfo, err := models.GetTransferInfo(ordernum, models.TXTYPE_ORDERNUM)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("GetTransInfoByOrderNumber")
			utils.RetError(ctx, CAPITAL_ERROR_CODE+5)
			return
		}
		log.Error("根据订单号获取指定转账信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 获取该笔转账对应的审批流信息
	flowInfo, err := models.GetBusinessFlowInfoByFlowID(flowID, 0)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("GetTransInfoByOrderNumber")
			utils.RetError(ctx, ERROR_CODE+6)
			return
		}
		log.Error("获取指定转账对应的审批流信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	txInfo.SingleLimit = flowInfo.SingleLimit
	// 获取各级人员对该订单的审批情况
	txInfo.ApprovalInfo, err = models.GetApprovalInfoByTxID(flowInfo, txInfo.TxID)
	if err != nil {
		log.Error("获取指定转账的审批情况", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["TRANSFER_INFO"], "data": txInfo})
}

// 获取资产信息
func GetBalanceList(ctx *gin.Context) {
	appid := ctx.MustGet("claims").(*jwt.CustomClaims).AppID

	page := ctx.DefaultQuery("page", "1")
	limit := ctx.DefaultQuery("limit", "20")
	p, _ := strconv.ParseInt(page, 10, 64)
	l, _ := strconv.ParseInt(limit, 10, 64)

	accInfo, _ := models.GetAccountInfoByAppID(appid)

	if accInfo.Depth != 0 {
		log.Error("GetBalanceList")
		utils.RetError(ctx, ERROR_CODE+7)
		return
	}

	// 更新币种列表
	_, err := models.GetCurrencyList("")

	if err != nil {
		log.Error("获取余额更新币种列表", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	asset, err := models.GetAssets(p, l)

	if err != nil {
		log.Error("获取资产信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["GET_BALANCE"], "data": asset})
}

// 获取币种列表
func GetCurrencyList(ctx *gin.Context) {
	keywords := ctx.Query("key_words")

	// 获取可用的币种列表
	result, err := models.GetCurrencyList(keywords)

	if err != nil {
		log.Error("rpc获取可用币种列表", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["CURRENCY_LIST"], "data": gin.H{"currency_list": result}})
}

func GetTradeHistoryList(ctx *gin.Context) {
	currency := ctx.Query("currency")
	page := ctx.DefaultQuery("page", "1")
	limit := ctx.DefaultQuery("limit", "20")
	p, _ := strconv.ParseInt(page, 10, 64)
	l, _ := strconv.ParseInt(limit, 10, 64)
	if currency == "" {
		log.Error("GetTradeHistoryList")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}

	// 获取币种信息
	currencyInfo, err := models.GetCurrencyInfoByName(currency)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("GetTradeHistoryList")
			utils.RetError(ctx, CAPITAL_ERROR_CODE+2)
			return
		}
		log.Error("充提现获取币种列表", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	data, err := models.GetTradeHistoryListByAppID(currencyInfo.Currency, currencyInfo.CurrencyID, p, l)
	if err != nil {
		log.Error("GetTradeHistoryList")
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["TTRADE_LIST"], "data": data})
}

// 审批转账
type applyTx struct {
	WDHash string		`json:"wdhash"`
	Category int64		`json:"category"`
	Amount float64		`json:"amount"`
	Fee float64			`json:"fee"`
	To string 			`json:"recaddress"`
	Apply []models.TxApprovalInfo	`json:"apply"`
	Sign []models.TxSignInfo		`json:"applysign"`
}
func ApprovalTransfer(ctx *gin.Context) {
	orderNum := ctx.PostForm("order_number")
	//appID := ctx.PostForm("app_account_id")
	appID := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	progress := ctx.PostForm("progress")
	signature := ctx.PostForm("sign")
	pwd := ctx.PostForm("password")
	reason := ctx.PostForm("reason")
	if orderNum == "" || progress == "" || signature == "" || pwd == "" {
		log.Error("ApprovalTransfer")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}

	p, _ := strconv.Atoi(progress)

	if p == models.TX_REJECTED {
		// 被驳回，需要填写原因
		if reason == "" {
			log.Error("驳回转账申请")
			utils.RetError(ctx, ERROR_CODE + 1)
			return
		}
	}

	// 校验密码
	validatepwd, err := verify.SignPassword(appID, pwd)

	if err != nil {
		log.Error("审批转账验证密码", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 账户信息
	accInfo, _ := models.GetAccountInfoByAppID(appID)
	if validatepwd == false {
		frozenAcc, attempts, err := models.AttemptFrozen(appID)
		if err != nil {
			log.Error("ApprovalTransfer")
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		if frozenAcc == true {
			// 账号被冻结
			var frozenTo = accInfo.FrozenTo
			if accInfo.FrozenTo == 0 {
				frozenTo = time.Now().Add(models.FROZEN_HOUR*time.Hour).Unix()
			}
			data := map[string]string{"frozenTo": strconv.FormatInt(frozenTo, 10)}
			log.Error("ApprovalTransfer")
			utils.RetError(ctx, ERROR_CODE + 18, data)
			return
		} else {
			log.Error("ApprovalTransfer")
			utils.RetError(ctx, ERROR_CODE + 16, map[string]string{"attempts": strconv.Itoa(attempts), "frozenFor": strconv.FormatInt(models.FROZEN_HOUR, 10)})
			return
		}
	}
	// 重置用户尝试密码次数
	models.ResetAttempts(appID)

	// 获取订单信息
	flowID, txInfo, err := models.GetTransferInfo(orderNum, models.TXTYPE_ORDERNUM)

	if err != nil {
		log.Error("审批转账获取订单信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if flowID == "" {
		log.Error("ApprovalTransfer")
		utils.RetError(ctx, CAPITAL_ERROR_CODE+5)
		return
	}

	//检测交易状态
	if txInfo.Progress == models.TX_CANCEL {
		log.Error("ApprovalTransfer")
		utils.RetError(ctx, CAPITAL_ERROR_CODE+14)
		return
	}

	// 获取币种信息
	applyContent := models.TransferContent{}
	json.Unmarshal([]byte(txInfo.ApplyInfo), &applyContent)
	currencyInfo, err := models.GetCurrencyInfoByName(applyContent.Currency)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("ApprovalTransfer")
			utils.RetError(ctx, CAPITAL_ERROR_CODE+8)
			return
		}
		log.Error("审批转账获取币种信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 验证是否有审批权限
	approverComments, err := models.GetTxInfoByApprover(appID, txInfo.TxID)

	if err != nil {
		log.Error("审批提现验证是否有审批权限", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if approverComments == -1 {
		log.Error("ApprovalTransfer")
		utils.RetError(ctx, CAPITAL_ERROR_CODE+3)
		return
	}

	if approverComments != 0 {
		log.Error("ApprovalTransfer")
		utils.RetError(ctx, CAPITAL_ERROR_CODE+6)
		return
	}

	// 验证签名
	signPass, err := verify.SignInfo(txInfo.ApplyInfo, accInfo.Pubkey, signature)

	if err != nil {
		log.Error("申请转账验签", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if signPass == false {
		log.Error("ApprovalTransfer")
		utils.RetError(ctx, ERROR_CODE+5)
		return
	}

	// 获取订单对应的审批流模板内容
	flowInfo, err := models.GetBusinessFlowInfoByFlowID(flowID, 0)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("Debug", err)
			utils.RetError(ctx, ERROR_CODE+6)
			return
		}
		log.Error("审批转账获取订单对应的审批流", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 获取审批流上链状态
	flowStatu, err := utils.BusinessFlowStatus(flowInfo.Hash)

	if err != nil {
		log.Error("审批转账订单对应审批流上链状态", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	log.Info("审批转账_获取审批流上链状态", flowStatu)

	err = models.UpdateFlowStatus(flowInfo.ID, flowStatu)

	if err != nil {
		log.Error("更新审批流上链状态", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if flowStatu != models.FLOW_APPROVALED {
		// 审批流哈希未上链，转账失败
		txProgress, _ := strconv.ParseInt(progress, 10, 64)
		err = models.UpdateTxProgress(txInfo.TxID, txProgress)

		if err != nil {
			log.Error("审批转账更新订单审批进度", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
		log.Error("ApprovalTransfer")
		utils.RetError(ctx, ERROR_CODE+6)
		return
	}

	// 提交审批意见
	err = models.ApprovalTransfer(txInfo.TxID, accInfo.ID, p, signature, reason)

	if err != nil {
		log.Error("审批转账提交审批意见", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 获取审批者在审批流中的位置
	location := models.GetManagerLocation(flowInfo.ApprovalInfo, appID)
	log.Debug("审批者在审批流中的位置", location)

	// 获取订单审批进度
	txProgress, err := models.GetTxProgress(flowInfo.ApprovalInfo, txInfo.TxID)
	if err != nil {
		log.Error("审批转账获取订单审批进度", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	if txProgress == models.TX_DOING && location.Level+1 < len(flowInfo.ApprovalInfo) {
		// 将未提交审批意见的其余审批者对该订单的审批意见progress置为-1
		err = models.InitManagerComments(flowInfo.ApprovalInfo, txInfo.TxID, location)
		if err != nil {
			log.Error("通知上级待审批", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	}
	// 获取订单审批最新进度
	newTxProgress, err := models.GetTxProgress(flowInfo.ApprovalInfo, txInfo.TxID)
	if err != nil {
		log.Error("审批转账获取订单最新审批进度", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 获取各级审批人员签名信息
	approversSign := models.GetTxApproversSign(flowInfo.ApprovalInfo, txInfo.TxID)


	log.Info("审批后的订单进度", gin.H{"progress": newTxProgress, "trans_id": txInfo.TxID})


	// 审批通过，转账
	if newTxProgress == models.TX_APPROVALED{
		wdHash := utils.GenHashStr(txInfo.ApplyInfo)
		factor, _ := strconv.ParseInt(currencyInfo.Factor, 10, 64)
		amount, _ := strconv.ParseFloat(applyContent.Amount, 64)
		miner, _ := strconv.ParseFloat(applyContent.Miner, 64)
		log.Debug("转账content", applyContent)
		// 转换单位
		amount_converted, _ := utils.UnitReConversion(amount, factor, 10)
		miner_converted, _ := utils.UnitReConversion(miner, factor, 10)
		amount_str, _ := json.Marshal(amount_converted)
		miner_str, _ := json.Marshal(miner_converted)
		applysign, _ := json.Marshal(approversSign)


		log.Debug("转账amount", amount_converted)
		log.Debug("转账miner", miner_converted)

		// 订单全部审批通过, 上链
		rpcParam := url.Values{"hash": {flowInfo.Hash}, "wdhash": {wdHash}, "category": {strconv.FormatInt(currencyInfo.CurrencyID, 10)}, "amount": {string(amount_str)}, "fee": {string(miner_str)}, "recaddress": {applyContent.ToAddr}, "apply": {txInfo.ApplyInfo}, "applysign": {string(applysign)}}
		log.Debug("向代理服务器申请提现", rpcParam)
		err = utils.ApplyTx(rpcParam.Encode())

		if err != nil {
			// 提交转账失败, 更改订单状态
			err = models.TransferFailed(txInfo.TxID)

			if err != nil {
				log.Error("ApprovalTransfer")
				utils.RetError(ctx, CAPITAL_ERROR_CODE+7)
				return
			}
		}
	}


	// 审批拒绝，释放额度
	if newTxProgress == models.TX_REJECTED {
		// 获取额度信息
		amountLeft, err := models.TransferLimitByCurrencyID(currencyInfo.CurrencyID, flowID)

		if err != nil {

		}
		tx_amount_f, _ := strconv.ParseFloat(txInfo.Amount, 10)
		amount_lft_f, _ := strconv.ParseFloat(amountLeft.AmountLeft, 10)
		// 释放额度
		new_amountlft := utils.AddFloat64(amount_lft_f, tx_amount_f)
		// 释放额度

		err = models.ReleaseAmount(flowInfo.Period, txInfo.CurrencyID, flowID, new_amountlft)
	}
	// 更新订单审批进度
	log.Info("订单最终审批进度", gin.H{"trans_id": txInfo.TxID, "progress": newTxProgress})
	err = models.UpdateTxProgress(txInfo.TxID, int64(newTxProgress))

	if err != nil {
		log.Error("审批转账更新订单最终审批进度", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}


	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["APPROVAL_TX"]})
}

// 撤销转账申请
func CancelTransferApply(ctx *gin.Context) {
	appid := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	transID := ctx.PostForm("order_number")
	reason := ctx.PostForm("reason")
	pwd := ctx.PostForm("password")

	if transID == "" || reason == "" || pwd == "" {
		log.Error("CancelTransferApply")
		utils.RetError(ctx, ERROR_CODE + 1)
		return
	}
	// 校验密码
	validatepwd, err := verify.SignPassword(appid, pwd)

	if err != nil {
		log.Error("审批转账验证密码", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 账户信息
	accInfo, err := models.GetAccountInfoByAppID(appid)

	if err != nil {
		log.Error("获取撤销者账号信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	if validatepwd == false {
		frozenAcc, attempts, err := models.AttemptFrozen(appid)
		if err != nil {
			log.Error("CancelTransferApply")
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		if frozenAcc == true {
			// 账号被冻结
			var frozenTo = accInfo.FrozenTo
			if accInfo.FrozenTo == 0 {
				frozenTo = time.Now().Add(models.FROZEN_HOUR*time.Hour).Unix()
			}
			data := map[string]string{"frozenTo": strconv.FormatInt(frozenTo, 10)}
			log.Error("CancelTransferApply")
			utils.RetError(ctx, ERROR_CODE + 18, data)
			return
		} else {
			log.Error("CancelTransferApply")
			utils.RetError(ctx, ERROR_CODE + 16, map[string]string{"attempts": strconv.Itoa(attempts), "frozenFor": strconv.FormatInt(models.FROZEN_HOUR, 10)})
			return
		}
	}
	// 重置用户尝试密码次数
	models.ResetAttempts(appid)

	// 获取转账申请详情
	flow_id, transInfo, err := models.GetTransferInfo(transID, models.TXTYPE_ORDERNUM)

	if err != nil {
		log.Error("撤销转账申请", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 审批流信息
	flowInfo, err := models.GetBusinessFlowInfoByFlowID(flow_id, 0)

	if err != nil {
		log.Error("CancelTransferApply get flowInfo err %v", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 是否有权撤回
	if accInfo.AppID != transInfo.ApplyerAppID {
		log.Error("是否有权撤回转账")
		utils.RetError(ctx, ERROR_CODE + 7)
		return
	}

	// 已经审批完成
	if transInfo.Progress == models.TX_APPROVALED || transInfo.Progress == models.TX_REJECTED {
		log.Debug("撤销转账申请", transInfo.Progress)
		utils.RetError(ctx, CAPITAL_ERROR_CODE + 13)
		return
	}
	// 撤销转账申请
	err = models.CancelTxApply(transInfo.TxID, accInfo.ID, reason)

	if err != nil {
		log.Error("撤销转账申请，更新订单状态", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 更新订单状态

	err = models.UpdateTxProgress(transInfo.TxID, models.TX_CANCEL)

	// 获取剩余额度信息
	amountLeft, err := models.TransferLimitByCurrencyID(transInfo.CurrencyID, flow_id)

	amount_lft_f, _ := strconv.ParseFloat(amountLeft.AmountLeft, 10)
	tx_amount_f, _ := strconv.ParseFloat(transInfo.Amount, 10)
	total_amount_f, _ := strconv.ParseFloat(amountLeft.Amount, 10)
	if err != nil {
		log.Error("撤销转账，获取剩余额度", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	// 更新后的额度
	new_amountlft := utils.AddFloat64(amount_lft_f, tx_amount_f)
	// 释放额度
	if utils.LessThanFloat64(total_amount_f, new_amountlft) || utils.EqualFloat64(total_amount_f, new_amountlft) {
		new_amountlft = total_amount_f
	}
	err = models.ReleaseAmount(flowInfo.Period, transInfo.CurrencyID, flow_id, new_amountlft)

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["TX_CANCEL"]})
}

// 审批转账操作日志
func TxOperationHistory(ctx *gin.Context) {
	//appid := ctx.MustGet("claims").(*jwt.CustomClaims).AppID
	transNum := ctx.Query("order_number")

	if transNum == "" {
		log.Error("TxOperationHistory")
		utils.RetError(ctx, ERROR_CODE + 1)
		return
	}

	data, err := models.GetTxOperationHistoryByNum(transNum)

	if err != nil {
		log.Error("审批转账日志", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["DATA"], "data": data})
}

// 代理服务器获取账户余额
func AssetForAgent(ctx *gin.Context) {
	//appid := ctx.Query("appid")
	page := ctx.DefaultQuery("page", "1")
	limit := ctx.DefaultQuery("limit", "20")
	p, _ := strconv.ParseInt(page, 10, 64)
	l, _ := strconv.ParseInt(limit, 10, 64)
	//log.Debug("AssetForAgent...appid = %v", appid)
	//if appid == "" {
	//	log.Error("AssetForAgent")
	//	utils.RetError(ctx, ERROR_CODE)
	//	return
	//}
	//isAdminAcc, err := verify.AdminAcc(appid)


	//if err != nil {
	//	log.Error("代理服务器获取资产信息", err)
	//	utils.RetError(ctx, ERROR_CODE)
	//	return
	//}
	//
	//if isAdminAcc == false {
	//	log.Error("AssetForAgent")
	//	utils.RetError(ctx, ERROR_CODE+7)
	//	return
	//}

	// 更新币种列表
	_, err := models.GetCurrencyList("")

	if err != nil {
		log.Error("私钥APP获取余额更新币种列表", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	asset, err := models.GetAssets(p, l)

	if err != nil {
		log.Error("获取资产信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["GET_BALANCE"], "data": asset})
}

// 代理服务器获取交易流水
func TradeHistoryForAgent(ctx *gin.Context) {
	appid := ctx.Query("appid")
	currency := ctx.Query("currency")
	page := ctx.DefaultQuery("page", "1")
	limit := ctx.DefaultQuery("limit", "20")
	p, _ := strconv.ParseInt(page, 10, 64)
	l, _ := strconv.ParseInt(limit, 10, 64)

	if currency == "" || appid == "" {
		log.Error("TradeHistoryForAgent")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}

	// 校验账号是否是私钥app
	isAdminAcc, err := verify.AdminAcc(appid)

	if err != nil {
		log.Error("私钥APP获取交易流水", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if isAdminAcc == false {
		log.Error("TradeHistoryForAgent")
		utils.RetError(ctx, ERROR_CODE + 7)
		return
	}

	// 获取币种信息
	currencyInfo, err := models.GetCurrencyInfoByName(currency)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("TradeHistoryForAgent")
			utils.RetError(ctx, CAPITAL_ERROR_CODE+2)
			return
		}
		log.Error("充提现获取币种列表", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	data, err := models.GetTradeHistoryListByAppID(currencyInfo.Currency, currencyInfo.CurrencyID, p, l)
	if data.List == nil {
		data.List = make([]models.TradeInfo, 0)
	}
	if err != nil {
		log.Error("TradeHistoryForAgent")
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["TTRADE_LIST"], "data": data})
}
