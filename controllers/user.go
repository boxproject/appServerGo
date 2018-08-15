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
	"github.com/boxproject/appServerGo/utils"
	"database/sql"
	"net/http"
	log "github.com/alecthomas/log4go"
	"github.com/satori/go.uuid"
	"encoding/json"
	"strconv"
	"github.com/boxproject/appServerGo/models/verify"
	"github.com/boxproject/appServerGo/models"
	//"github.com/dgrijalva/jwt-go"
	//"time"
	JWT "github.com/boxproject/appServerGo/middleware/jwt"
	"time"
	"github.com/dgrijalva/jwt-go"
)

var db *sql.DB
//var l

// 申请注册账号
func ApplyForAccount(ctx *gin.Context) {
	msg := ctx.PostForm("msg")
	applicantID := ctx.PostForm("applyer_id")
	captainID := ctx.PostForm("captain_id")
	applicantAccount := ctx.PostForm("applyer_account")
	password := ctx.PostForm("password")
	log.Info("用户提交注册信息_param: %+v \n", gin.H{"msg": msg, "applicantID": applicantID, "captainID": captainID, "applicantAccount": applicantAccount, "password": password})

	if msg == "" || applicantID == "" || captainID == "" || applicantAccount == "" || password == "" {
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}

	// 检测用户名是否存在
	accExist, err := verify.AccExist(applicantAccount)
	log.Info("员工注册账号重名", accExist)
	if err != nil {
		log.Error("注册验证账号重名", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if accExist {
		log.Error("ApplyForAccount")
		utils.RetError(ctx, ERROR_CODE+10)
		return
	}

	// 检测appid是否存在
	employeeAccInfo, err := models.GetAccountInfoByAppID(applicantID)

	if err != nil && err != sql.ErrNoRows {
		log.Error("注册检测appid重复", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if employeeAccInfo.AppID != "" && employeeAccInfo.Departured == false {
		// 账号存在
		log.Error("ApplyForAccount")
		utils.RetError(ctx, ERROR_CODE+10)
		return
	} else if employeeAccInfo.AppID != "" && employeeAccInfo.Departured == true {
		// 已离职
		log.Error("ApplyForAccount")
		utils.RetError(ctx, ERROR_CODE+11)
		return
	}

	// 是否提交过注册申请
	has_applyed, err := verify.HasApplyedRegistration(applicantID, captainID)

	if err != nil {
		log.Error("注册是否提交过相同注册申请", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 已提交过申请
	if has_applyed {
		log.Error("ApplyForAccount")
		utils.RetError(ctx, ERROR_CODE+2)
		return
	}

	//是否是向私钥APP注册申请
	isAdminAcc, err := verify.AdminAcc(captainID)

	log.Info("员工扫私钥", isAdminAcc)

	if err != nil {
		log.Error("isAdminAcc...")
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 生成注册ID regid
	regid := uuid.Must(uuid.NewV4()).String()

	// 向代理服务器提交注册申请
	if isAdminAcc == true {
		err = utils.RegToServer(regid, msg, applicantID, captainID, applicantAccount, 0)
		if err != nil {
			log.Error("apply for agent...")
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	} else {
		// 校验上级账号信息
		_, errcode, _ := verify.ValidateUser(captainID)

		if errcode != 0 {
			log.Error("verify admin...")
			utils.RetError(ctx, errcode)
			return
		}
	}

	// 处理密码
	hashpassword := utils.GenHashStr(password)
	// 将申请记录存入tb_registration_history中
	err = models.AddRegistration(regid, applicantID, captainID, msg, applicantAccount, hashpassword)
	if err != nil {
		log.Error("hashpassrord...")
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["GEN_ACCOUNT"], "data": gin.H{"reg_id": regid}})
}

// 获取指定上司所涉及的注册申请
func RegistrationInfo(ctx *gin.Context) {
	claims := ctx.MustGet("claims").(*JWT.CustomClaims)
	captainID := claims.AppID
	log.Info("captain_id", captainID)

	// 获取对应的注册信息
	reginfo_list, err := models.Registration(captainID)
	if err != nil {
		log.Error("获取注册信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	result := reginfo_list
	// 删除多余的注册信息
	if len(reginfo_list) > 5 {
		result = reginfo_list[:5]
		min_date_time := reginfo_list[len(reginfo_list)-1].ApplyAt
		max_date_time := reginfo_list[5].ApplyAt
		// 默认最多返回5条记录
		err = models.DelRegistrationInfoByDateTime(min_date_time, max_date_time)
		if err != nil {
			log.Error("删除多余注册信息", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["GET_REGISTRATION"], "data": result})
}

// 下属获取注册申请审批结果
func RegistrationApprovalInfo(ctx *gin.Context) {
	regid := ctx.Query("reg_id")

	if regid == "" {
		log.Error("RegistrationApprovalInfo")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}
	_, err := models.GetRegistrationByRegID(regid, -1)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("RegistrationApprovalInfo")
			// 未找到对应的注册申请
			utils.RetError(ctx, ERROR_CODE+3)
			return
		} else {
			log.Error("下属获取注册信息校验注册信息", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	}

	// 获取账户对应的注册信息
	reg_info_acc, err := models.GetRegistrationByRegIDWithAcc(regid)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("RegistrationApprovalInfo")
			utils.RetError(ctx, ERROR_CODE+3)
			return
		}
		log.Error("下属获取注册信息详情", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 生成token
	var j *JWT.JWT = &JWT.JWT{
		[]byte(JWT.GetSignKey()),
	}
	claims := JWT.CustomClaims{reg_info_acc.ApplicantID, reg_info_acc.ApplicantAccount, jwt.StandardClaims{ExpiresAt:time.Now().Add((models.TOKEN_EXP)*time.Hour).Unix()}}
	token, err := j.CreateToken(claims)
	log.Debug("注册成功", reg_info_acc.ApplicantID, token )
	if err != nil {
		log.Error("用户注册成功生成token", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["GET_REGISTRATION"], "data": reg_info_acc, "token": token})
}

// 上级管理员审批下级员工的注册申请
func ApprovalRegistration(ctx *gin.Context) {
	//captainID := ctx.MustGet("claims").(*JWT.CustomClaims).AppID
	regid := ctx.PostForm("reg_id")
	consent := ctx.PostForm("consent")
	applyerPubKey := ctx.PostForm("applyer_pub_key")
	cipherText := ctx.PostForm("cipher_text")
	enPubKey := ctx.PostForm("en_pub_key")

	log.Info("上级审批注册: ", gin.H{"reg_id": regid, "consent": consent})
	if regid == "" || consent == "" {
		log.Error("ApprovalRegistration")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}

	// 获取注册信息
	reginfo, err := models.GetRegistrationByRegID(regid, 0)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("ApprovalRegistration")
			utils.RetError(ctx, ERROR_CODE+3)
			return
		}
		log.Error("审批注册获取对应注册信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 审批通过
	if consent == "2" {
		log.Info("上级审批注册申请_审批通过: ", gin.H{"applyerPubKey": applyerPubKey, "cipherText": cipherText, "enPubKey": enPubKey})
		if applyerPubKey == "" || cipherText == "" || enPubKey == "" {
			log.Error("ApprovalRegistration")
			utils.RetError(ctx, ERROR_CODE+1)
			return
		}

		is_uploaded := 1
		// 获取直属上级账号信息
		captainAccountInfo, err := models.GetAccountInfoByAppID(reginfo.CaptainID)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Error("ApprovalRegistration")
				// 账号不存在
				utils.RetError(ctx, ERROR_CODE+4)
				return
			}
			log.Error("审批注册获取直属上级账号信息", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		// 已离职
		if captainAccountInfo.Departured {
			log.Error("ApprovalRegistration")
			utils.RetError(ctx, ERROR_CODE+14)
			return
		}

		// 验证签名值
		signPass, err := verify.SignInfo(applyerPubKey, captainAccountInfo.Pubkey, enPubKey)

		log.Info("审批注册验证签名: ", signPass)

		if err != nil {
			log.Error("审批注册验签", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		if signPass == false {
			log.Error("ApprovalRegistration")
			utils.RetError(ctx, ERROR_CODE+5)
			return
		}

		eAccInfo := captainAccountInfo
		eAccInfo.Account = reginfo.ApplicantAccount
		eAccInfo.AppID = reginfo.ApplicantID
		eAccInfo.Pubkey = applyerPubKey
		eAccInfo.CipherText = cipherText
		eAccInfo.EnPubkey = enPubKey
		eAccInfo.RegID = regid
		eAccInfo.Depth = eAccInfo.Depth+1
		err = models.GenAccount(eAccInfo, is_uploaded)
		if err != nil {
			log.Error("审批注册创建账号", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	}

	// 记录上级审批结果
	log.Info("上级审批注册_out", gin.H{"regid": regid, "consent": consent})

	err = models.CaptainApprovalRegInfo(regid, consent)
	if err != nil {
		log.Error("审批注册记录上级审批结果", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["APPROVAL_REGISTRATION"]})
}

// 员工APP反馈上级审批结果出错
func CancelApprovalRegistration(ctx *gin.Context) {
	regid := ctx.PostForm("reg_id")
	applyerid := ctx.PostForm("applyer_id")
	signature := ctx.PostForm("sign")

	if regid == "" || applyerid == "" || signature == "" {
		log.Error("CancelApprovalRegistration")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}

	// 获取员工账号信息
	//employeeAccountInfo, err := models.GetAccountInfoByAppID(applyerid)
	employeeAccountInfo, errorcode, errmsg :=verify.ValidateUser(applyerid)

	if errorcode != 0 {
		log.Error("CancelApprovalRegistration")
		utils.RetError(ctx, errorcode, errmsg)
		return
	}

	// 获取注册信息
	reginfo, err := models.GetRegistrationByRegID(regid, 1)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("CancelApprovalRegistration")
			utils.RetError(ctx, ERROR_CODE+3)
			return
		}
		log.Error("反馈审批注册出错获取对应注册信息", err)
		utils.RetError(ctx, ERROR_CODE)

		return
	}

	// 注册信息不符
	if reginfo.ApplicantID != applyerid {
		log.Error("CancelApprovalRegistration")
		utils.RetError(ctx, ERROR_CODE+7)
		return
	}

	// 验证签名
	signPass, err := verify.SignInfo(regid, employeeAccountInfo.Pubkey, signature)
	if err != nil {
		log.Error("反馈审批注册出错验签", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	log.Info("员工反馈注册审批出错_验签", signPass)

	if signPass == false {
		log.Error("CancelApprovalRegistration")
		utils.RetError(ctx, ERROR_CODE+5)
		return
	}

	// 回滚信息
	var underEmployeeInfo []models.AccountInfo
	underEmployeeInfo = append(underEmployeeInfo, employeeAccountInfo)
	err = models.ChangeEmployee(applyerid, underEmployeeInfo)

	if err != nil {
		log.Error("反馈审批注册出错回滚注册信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 如果是根节点账号
	if employeeAccountInfo.Depth == 0 {
		err = utils.RegToServer(regid, reginfo.Msg, applyerid, reginfo.CaptainID, employeeAccountInfo.Account, 1)
		if err != nil {
			log.Error("CancelApprovalRegistration")
			utils.RetError(ctx, ERROR_CODE+9)
			return
		}
	}

	models.CaptainApprovalRegInfo(regid, "1")

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["NOTICE"]})
}

// 获取下属员工账号列表
func GetEmployeeAccountsList(ctx *gin.Context) {
	//appid := ctx.Query("app_account_id")
	//managerAppid := ctx.MustGet("claims").(*JWT.CustomClaims).AppID
	appid := ctx.Query("app_account_id")
	page := ctx.DefaultQuery("page", "1")
	limit := ctx.DefaultQuery("limit", "20")
	p, _ := strconv.ParseInt(page, 10, 64)
	l, _ := strconv.ParseInt(limit, 10, 64)
	kw := ctx.Query("key_words")
	// 获取该上级账号信息
	managerAccInfo, _ := models.GetAccountInfoByAppID(appid)

	var result models.EmployeeList
	var err error
	// 如果是搜索
	if kw != "" {
		result, err = models.SearchAccountInfoByAccount(kw, p, l)
		if err != nil {
			log.Error("GetEmployeeAccountsList")
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	} else {
		result, err = models.GetEmployeeAccInfoByCaptainID(managerAccInfo.Depth+1, managerAccInfo.AccLft, managerAccInfo.AccRgt, p, l)
		if err != nil {
			log.Error("GetEmployeeAccountsList")
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["ACCOUNTS_LIST"], "data": result})
}

// 获取下属账号详情
func GetEmployeeAccountsInfo(ctx *gin.Context) {
	mAppid := ctx.MustGet("claims").(*JWT.CustomClaims).AppID
	eAppid := ctx.Query("employee_account_id")
	if mAppid == "" || eAppid == "" {
		log.Error("GetEmployeeAccountsInfo")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}
	// 获取上级账号信息
	managerAccInfo, err := models.GetAccountInfoByAppID(mAppid)

	if err != nil {
		log.Error("GetEmployeeAccountsInfo")
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 下级账号信息
	employeeAccInfo, errorcode, errmsg := verify.ValidateUser(eAppid)

	if errorcode != 0 {
		log.Error("GetEmployeeAccountsInfo")
		utils.RetError(ctx, errorcode, errmsg)
		return
	}

	// 是否有权获取
	if employeeAccInfo.Depth <= managerAccInfo.Depth {
		log.Error("GetEmployeeAccountsInfo")
		utils.RetError(ctx, ERROR_CODE+7)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["EMPLOYEE_ACCOUNT_INFO"], "data": gin.H{"app_account_id": employeeAccInfo.AppID, "cipher_text": employeeAccInfo.CipherText}})
}

// 代理服务器上报私钥APP审批注册结果
func AdminApprovalRegistration(ctx *gin.Context) {
	regid := ctx.PostForm("regid")
	statu := ctx.PostForm("status")
	// 获取注册信息
	regInfo, err := models.GetRegistrationByRegID(regid, 0)
	if err == sql.ErrNoRows {
		log.Error("AdminApprovalRegistration")
		utils.RetError(ctx, ERROR_CODE+3)
		return
	}

	// 审批通过
	if statu == "2" {
		cipherText := ctx.PostForm("ciphertext")
		pubKey := ctx.PostForm("pubkey")
		if cipherText == "" {
			log.Error("AdminApprovalRegistration")
			utils.RetError(ctx, ERROR_CODE+1)
			return
		}
		// 生成账号
		accInfo := models.AccountInfo{}
		accInfo.Account = regInfo.ApplicantAccount
		accInfo.AppID = regInfo.ApplicantID
		accInfo.Pubkey = pubKey
		accInfo.CipherText = cipherText
		accInfo.AccRgt = 0
		accInfo.RegID = regid
		accInfo.Depth = 0
		err = models.GenAccount(accInfo, 1)

		if err != nil {
			log.Error("私钥app审批注册创建账号", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}
	}
	// 记录上级审批结果
	err = models.UpdateCaptainApprovalInfo(regid, statu)

	if err != nil {
		log.Error("私钥app审批注册记录审批结果", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["APPROVAL_REGISTRATION"]})
}

// 删除或替换员工账号
func ChangeEmployeeAccount(ctx *gin.Context) {
	mAppid := ctx.MustGet("claims").(*JWT.CustomClaims).AppID
	eAppid := ctx.PostForm("employee_account_id")
	sign := ctx.PostForm("sign")
	cipherText := ctx.PostForm("cipher_texts")
	rAppid := ctx.PostForm("replacer_account_id")
	log.Info("删除/替换员工账号", gin.H{"employee_account_id": eAppid, "manager_account_id": mAppid, "sign": sign, "cipher_texts": cipherText})
	if eAppid == "" || mAppid == "" || sign == "" {
		log.Error("ChangeEmployeeAccount")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}
	// 获取上级账号信息
	managerAccInfo, _ := models.GetAccountInfoByAppID(mAppid)

	// 获取被删除、替换者账号信息
	employeeAccInfo, errorcode, errmsg := verify.ValidateUser(eAppid)

	if errorcode != 0 {
		if errorcode == 1011 {
			errorcode = 1013
		}
		log.Error("ChangeEmployeeAccount")
		utils.RetError(ctx, errorcode, errmsg)
		return
	}

	// 是否有权删除/替换
	if managerAccInfo.Depth >= employeeAccInfo.Depth {
		log.Error("ChangeEmployeeAccount")
		utils.RetError(ctx, ERROR_CODE+7)
		return
	}
	// 验证签名
	signPass, err := verify.SignInfo(eAppid, managerAccInfo.Pubkey, sign)
	log.Info("删除替换下级账号验签", signPass)

	if err != nil {
		log.Error("删除账号验签", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if signPass == false {
		log.Error("ChangeEmployeeAccount")
		utils.RetError(ctx, ERROR_CODE+5)
		return
	}

	// 获取被删除或被替换者直属下级账号信息
	eeAccInfo, err := models.GetUnderlingInfoByManagerAccountID(employeeAccInfo.Depth+1, employeeAccInfo.AccLft, employeeAccInfo.AccRgt)
	if err != nil {
		log.Error("删除账号获取被删除者下属账号信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 更新摘要信息
	cipherTexts := []models.AccountInfo{}
	json.Unmarshal([]byte(cipherText), &cipherTexts)
	data := models.ChangeCipherInfo(eeAccInfo, cipherTexts)
	// 删除,更新摘要信息
	err = models.ChangeEmployee(eAppid, data)
	// 更新替换后的上下级关系
	if rAppid != "" {
		// 替换
		rAccInfo, err := models.GetAccountInfoByAppID(rAppid)

		if err != nil {
			if err == sql.ErrNoRows {
				log.Error("ChangeEmployeeAccount")
				utils.RetError(ctx, ERROR_CODE+8)
				return
			}
			log.Error("删除账号更新摘要信息", err)
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		if rAccInfo.Departured {
			log.Error("ChangeEmployeeAccount")
			utils.RetError(ctx, ERROR_CODE+13)
			return
		}

		// 同级才可以替换
		if rAccInfo.Depth != employeeAccInfo.Depth {
			log.Error("ChangeEmployeeAccount")
			utils.RetError(ctx, ERROR_CODE+15)
			return
		}

		if len(eeAccInfo) > 0 {
			for _, r := range eeAccInfo {
				models.ReplaceEmployee(r.AppID, rAccInfo.AppID)
			}
		}
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["DATA"]})
}

// 获取指定下属的公钥信息
func GetEmployeePubKeyInfo(ctx *gin.Context) {
	mAppid := ctx.MustGet("claims").(*JWT.CustomClaims).AppID
	eAppid := ctx.Query("employee_account_id")
	if eAppid == "" {
		log.Error("GetEmployeePubKeyInfo")
		utils.RetError(ctx, ERROR_CODE+1)
		return
	}
	// 上级账号信息
	managerAccInfo, err := models.GetAccountInfoByAppID(mAppid)

	if err != nil {
		log.Error("GetEmployeePubKeyInfo")
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 是否有权限获取
	if managerAccInfo.Depth != 0 {
		log.Error("GetEmployeePubKeyInfo")
		utils.RetError(ctx, ERROR_CODE+7)
		return
	}

	// 下级账号信息
	_, errcode, _ := verify.ValidateUser(eAppid)

	if errcode != 0 {
		log.Error("GetEmployeePubKeyInfo")
		utils.RetError(ctx, errcode)
		return
	}

	// 获取下属公钥信息
	data, err := models.GetEmployeeEnPubKeyInfo(eAppid)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("获取下属公钥信息", err)
			utils.RetError(ctx, ERROR_CODE+4)
			return
		}
		log.Error("获取指定下属公钥信息", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 更新状态，标记公钥已上传根节点
	result := []models.AccountInfo{}
	type empPubksInfo struct {
		models.AccountInfo
		Applyer string `json:"applyer"`
		ApplyerAcc string `json:"applyer_account"`
		Captain string `json:"captain"`
	}

	result = append(result, data)
	models.UpdateAccountsPubkeyUploadInfo(result)

	data_r := empPubksInfo{}
	data_r.Applyer = data.AppID
	data_r.ApplyerAcc = data.Account
	data_r.Pubkey = data.Pubkey
	data_r.Captain = data.ManagerAppID
	data_r.EnPubkey = data.EnPubkey
	data_r.CipherText = data.CipherText
	data_r.ApplyAt = data.ApplyAt
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["EMPLOYEE_PUBKEY_INFO"], "data": data_r})
}

func GetEmployeePubKeyInfoList(ctx *gin.Context) {
	appid := ctx.MustGet("claims").(*JWT.CustomClaims).AppID

	// 获取账号信息
	accInfo, _ := models.GetAccountInfoByAppID(appid)

	if accInfo.Depth != 0 {
		log.Error("GetEmployeePubKeyInfoList")
		utils.RetError(ctx, ERROR_CODE+7)
		return
	}

	// 获取未被上传的下属公钥信息列表
	list, err := models.GetEmployeeEnPubKeyInfoList(appid)

	if err != nil {
		log.Error("获取下属公钥列表未被上传的公钥", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	err = models.UpdateAccountsPubkeyUploadInfo(list)

	if err != nil {
		log.Error("下属公钥列表更新", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["EMPLOYEE_PUB_KEY"], "data": list})
}

// 选择部门
func SelectBranch(ctx *gin.Context) {
	bid := ctx.PostForm("bid")

	if bid == "" {
		log.Error("SelectBranch")
		utils.RetError(ctx, ERROR_CODE + 1)
		return
	}

	// 获取部门信息
	_, err := models.GetBranchInfoByID(bid)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Error("SelectBranch")
			utils.RetError(ctx, BUSINESS_ERROR_CODE + 5)
			return
		}
		log.Error("SelectBranch")
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 修改账号所属部门信息
	models.ChangeAccBranch(ctx.PostForm("appid"), bid)

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["DATA"]})
}

// 获取账号详情
func GetAccountDetail(ctx *gin.Context){
	//appid := ctx.Query("appid")
	appid := ctx.MustGet("claims").(*JWT.CustomClaims).AppID
	sign := ctx.Query("sign")

	if sign == "" {
		log.Error("GetAccountDetail")
		utils.RetError(ctx, ERROR_CODE + 1)
		return
	}

	accountInfo, _ := models.GetAccountInfoByAppID(appid)

	//if errorcode != 0 {
	//	utils.RetError(ctx, errorcode, errmsg)
	//	return
	//}

	// 验证签名信息
	signPass, err := verify.SignInfo(appid, accountInfo.Pubkey, sign)

	if err != nil {
		log.Error("获取账号详情验签", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if signPass == false {
		log.Error("GetAccountDetail")
		utils.RetError(ctx, ERROR_CODE + 5)
		return
	}

	result := make(map[string]interface{})

	result["Department"] = gin.H{"ID": accountInfo.BranchID, "Name": accountInfo.BranchName}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["DATA"], "data":result})
}

// 登录
func Login(ctx *gin.Context) {
	appid := ctx.PostForm("appid")
	password := ctx.PostForm("password")

	// 参数不能为空
	if appid == "" || password == "" {
		log.Error("Login")
		utils.RetError(ctx, ERROR_CODE + 1)
		return
	}

	// 校验账号
	accInfo, errorcode, errmsg := verify.ValidateUser(appid)

	if errorcode != 0  {
		log.Error("Login")
		utils.RetError(ctx, errorcode, errmsg)
		return
	}

	// 校验密码
	validePassword, err := verify.SignPassword(appid, password)
	if err != nil {
		log.Error("登录验证密码", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	if validePassword == false {
		frozenAcc, attempts, err := models.AttemptFrozen(appid)
		if err != nil {
			log.Error("validate password...")
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		if frozenAcc == true {
			// 账号被冻结
			data := map[string]string{"frozenTo":strconv.FormatInt(time.Now().Add(models.FROZEN_HOUR*time.Hour).Unix(), 10)}
			log.Error("Login")
			utils.RetError(ctx, ERROR_CODE + 18, data)
			return
		} else {
			log.Error("Login")
			utils.RetError(ctx, ERROR_CODE + 16, map[string]string{"attempts": strconv.Itoa(attempts), "frozenFor": strconv.FormatInt(models.FROZEN_HOUR, 10)})
			return
		}
	}

	// 重置用户尝试密码次数
	err = models.ResetAttempts(appid)

	if err != nil {
		log.Error("登录成功重置尝试次数", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	// 生成token
	var j *JWT.JWT = &JWT.JWT{
		[]byte(JWT.GetSignKey()),
	}
	claims := JWT.CustomClaims{appid, accInfo.Account, jwt.StandardClaims{ExpiresAt:time.Now().Add((models.TOKEN_EXP)*time.Hour).Unix()}}
	token, err := j.CreateToken(claims)
	log.Debug("用户登录成功", appid, token)
	if err != nil {
		log.Error("Login")
		log.Error("登录生成TOKEN", err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["LOGIN"], "data":gin.H{"token":token}})

}

// 修改密码
func ChangePassword(ctx *gin.Context) {
	appid := ctx.MustGet("claims").(*JWT.CustomClaims).AppID
	oldpwd := ctx.PostForm("oldpwd")
	newpwd := ctx.PostForm("newpwd")

	if oldpwd == "" || newpwd == "" {
		log.Error("ChangePassword")
		utils.RetError(ctx, ERROR_CODE + 1)
		return
	}

	// 校验密码
	validePassword, err := verify.SignPassword(appid, oldpwd)

	if err != nil {
		log.Error("修改密码验证旧密码", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}

	accInfo, _ := models.GetAccountInfoByAppID(appid)

	if validePassword == false {
		frozenAcc, attempts, err := models.AttemptFrozen(appid)
		if err != nil {
			log.Error("ChangePassword")
			utils.RetError(ctx, ERROR_CODE)
			return
		}

		if frozenAcc == true {
			frozenTo := accInfo.FrozenTo
			if accInfo.FrozenTo == 0 {
				frozenTo = time.Now().Add(models.FROZEN_HOUR*time.Hour).Unix()
			}
			// 账号被冻结
			data := map[string]string{"frozenTo":strconv.FormatInt(frozenTo, 10)}
			log.Error("ChangePassword")
			utils.RetError(ctx, ERROR_CODE + 18, data)
			return
		} else {
			log.Error("ChangePassword")
			utils.RetError(ctx, ERROR_CODE + 16, map[string]string{"attempts": strconv.Itoa(attempts), "frozenFor": strconv.FormatInt(models.FROZEN_HOUR, 10)})
			return
		}
	}

	// 修改密码
	err = models.ModifyPwd(appid, newpwd)

	if err != nil {
		log.Error("修改密码写入新密码", err)
		utils.RetError(ctx, ERROR_CODE)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 0, "message": utils.ReadJsonFile(ctx)["DATA"]})
}
