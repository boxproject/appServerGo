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
	"github.com/gin-gonic/gin"
	"database/sql"
	log "github.com/alecthomas/log4go"
	"github.com/boxproject/appServerGo/models"
	"net/http"
	"github.com/boxproject/appServerGo/utils"
	"strings"
	"github.com/satori/go.uuid"
)

// 代理服务器通知临时提现结果
func WithdrawResultOfID(ctx *gin.Context) {
	wdhash := ctx.PostForm("wd_hash")
	txid := ctx.PostForm("tx_id")
	log.Info("代理服务器通知临时提现结果:", gin.H{"wd_hash": wdhash, "tx_id": txid})
	if wdhash != "" && txid != "" {
		// 获取该笔转账记录详情
		txInfo, err := models.GetTransferInfoByWdHash(wdhash)
		if err != nil && err != sql.ErrNoRows {
			log.Error("代理通知提现结果", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
		if err != sql.ErrNoRows && txInfo.Arrived != 2 {
			// 插入本地数据库 tb_transfer_history,set progress = 1
			err = models.AddTempTransferArrivedInfo(wdhash, txid, 1)
			if err != nil {
				log.Error("代理服务器通知最终提现结果插入记录错误", err)
				utils.RetError(ctx, ERROR_CODE)
				return
			}
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["NOTICE"]})
}

// 代理服务器通知最终提现信息
func WithdrawResult(ctx *gin.Context) {
	wdhash := ctx.PostForm("wd_hash")
	txid := ctx.PostForm("tx_id")
	if wdhash != "" && txid != "" {
		log.Info("代理服务器通知最终提现结果:", gin.H{"wd_hash": wdhash, "tx_id": txid})
		// 获取该笔转账记录详情
		txInfo, err := models.GetTransferInfoByWdHash(wdhash)
		if err != nil && err != sql.ErrNoRows {
			log.Error("WithdrawResult")
			utils.RetError(ctx, ERROR_CODE)
			return
		}
		if err != sql.ErrNoRows && txInfo.Arrived == 1 {
			// 插入本地数据库 tb_transfer_history,set progress = 2
			err = models.AddTransferArrivedInfo(txInfo, wdhash, txid, 2)
			if err != nil {
				log.Error("代理服务器通知最终提现结果插入记录错误", err)
				utils.RetError(ctx, ERROR_CODE)
				return
			}
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["NOTICE"]})
}

// 代理服务器上报充值记录
func DepositSuccess(ctx *gin.Context) {
	fromAddr := ctx.PostForm("from")
	toAddr := ctx.PostForm("to")
	amount := ctx.PostForm("amount")
	txID := ctx.PostForm("tx_id")
	currencyID := ctx.PostForm("category")
	log.Info("代理服务器通知充值结果", gin.H{"fromAddr": fromAddr, "toAddr": toAddr, "amount": amount, "txID": txID, "currencyID": currencyID})
	if fromAddr == "" || toAddr == "" || amount == "" || txID == "" || currencyID == "" {
		log.Error("DepositSuccess")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}
	// 获取币种信息
	currencyInfo, _ := models.GetCurrencyInfoByID(currencyID)
	fromArry := strings.Split(fromAddr, ",")
	orderNum := uuid.Must(uuid.NewV4()).String()
	// 记录充值记录并更新余额
	err := models.DepositHistory(orderNum, toAddr, amount, txID, currencyInfo.CurrencyID, fromArry)

	if err != nil {
		log.Error("代理服务器上报充值记录落库", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["NOTICE"]})
}

// 添加代币
func AddCurrency(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["NOTICE"]})
}
