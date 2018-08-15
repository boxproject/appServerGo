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
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"encoding/json"
	"net/http"
	"strconv"
	logger "github.com/alecthomas/log4go"
	"strings"
	"crypto/sha256"
	"encoding/hex"
	"crypto/tls"
	"math/big"
	"github.com/go-errors/errors"
	"fmt"
)

func ReadJsonFile(ctx *gin.Context) map[string]string {
	lang := ctx.GetHeader(" content-language")
	if lang == "" {
		lang = "zh-Hans"
	}
	var err error
	language := make(map[string]map[string]string)
	if _, okHave := language[lang]; okHave == false {
		language[lang], err = GetJsonMap("./lang/" + lang + ".json")
		if err != nil {
			logger.Error("GetJsonMap: ", err.Error())
			return nil
		}
	}
	return language[lang]
}

func RetError(ctx *gin.Context, code int, args ...interface{} )error {
	language := ReadJsonFile(ctx)
	errorcode := strconv.Itoa(code)
	msg := language[errorcode]


	logger.Error("[ERROR]: %v", gin.H{
		"code":  code,
		"cause": language[errorcode]})
	if len(args)>0 {
		ctx.JSON(http.StatusOK, gin.H{"code": code, "message": msg, "data": args[0]})
	} else {
		ctx.JSON(http.StatusOK, gin.H{"code": code, "message": msg})
	}

	return errors.New(fmt.Sprintf("[ERROR]: %v",gin.H{"code":  code, "cause": language[errorcode]}))

}

func GetJsonMap(filename string) (map[string]string, error) {
	var retJsonMap = map[string]string{}
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		logger.Error("ReadFile: ", err.Error())
		return nil, err
	}

	if err := json.Unmarshal(bytes, &retJsonMap); err != nil {
		logger.Error("Unmarshal: ", err.Error())
		return nil, err
	}
	return retJsonMap, nil
}

// 发起https请求
func HttpRequest(method, urls string, jsonStr string) (r []byte, err error) {
	//跳过证书验证
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	payload := strings.NewReader(jsonStr)
	req, _ := http.NewRequest(method, urls, payload)

	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded;param=value")
	}
	c := &http.Client{Transport: tr}
	res, err := c.Do(req)

	if err != nil {
		logger.Error("请求代理服务器", err)
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	return body, nil
}

// 生成sha256哈希字符串
func GenHashStr(msg string) string {
	h := sha256.New()
	h.Write([]byte(msg))
	bs := h.Sum(nil)
	transHash := "0x" + hex.EncodeToString(bs)
	return transHash
}


// 单位换算 amount/(10^factor)
func UnitConversion(amount float64, factor int64, base int64) (float64, error) {
	decimal := new(big.Int).Exp(big.NewInt(10), big.NewInt(factor), big.NewInt(0)).String()
	decimal_f, err := strconv.ParseFloat(decimal, 64)

	if err != nil {
		return 0, err
	}

	result := DivFloat64(amount, decimal_f)

	return result, nil
}

// 单位反换算 amount*(10^factor)
func UnitReConversion(amount float64, factor int64, base int64) (float64, error) {
	decimal := new(big.Int).Exp(big.NewInt(10), big.NewInt(factor), big.NewInt(0)).String()
	decimal_f, err := strconv.ParseFloat(decimal, 64)

	if err != nil {
		return 0, err
	}

	result := MulFloat64(amount, decimal_f)

	return result, nil
}



