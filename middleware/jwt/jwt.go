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
package jwt

import (
	"errors"
	"time"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/boxproject/appServerGo/utils"
	log "github.com/alecthomas/log4go"
	"github.com/boxproject/appServerGo/models"
	"github.com/boxproject/appServerGo/models/verify"
)
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string
		if c.Request.Method == "GET" {
			token = c.Query("token")
		} else {
			token = c.PostForm("token")
		}
		if token == "" {
			log.Error("JWTAuth")
			c.Abort()
			utils.RetError(c, 1020)
			return
		}

		log.Debug("token", token)

		j := NewJWT()
		claims, err := j.ParseToken(token)

		if claims == nil || claims.AppID == "" {
			log.Error("JWTAuth")
			c.Abort()
			utils.RetError(c, 1020)
			return
		}

		if err != nil{
			if err == TokenExpired {
				if token, err = j.RefreshToken(token); err == nil {
					c.Next()
					return
				}
			}
			c.Abort()
			log.Error("token", err.Error())
			utils.RetError(c, 1020)
			return
		}


		appid := claims.AppID
		// 校验账号
		_, errorcode, errmsg := verify.ValidateUser(appid)

		if errorcode != 0 {
			log.Error("JWTAuth")
			c.Abort()
			utils.RetError(c, errorcode, errmsg)
			return
		}

		c.Set("claims", claims)
	}
}
type JWT struct {
	SigningKey []byte
}
var (
	TokenExpired error = errors.New("Token is expired")
	TokenNotValidYet error = errors.New("Token not active yet")
	TokenMalformed error = errors.New("That's not even a token")
	TokenInvalid error = errors.New("Couldn't handle this token:")
	SignKey string = "box.la"
)
type CustomClaims struct {
	AppID string `json:"appid"`
	Account string `json:"account"`
	jwt.StandardClaims
}
func NewJWT() *JWT {
	return &JWT{
		[]byte(GetSignKey()),
	}
}
func GetSignKey() string {
	return SignKey
}
func SetSignKey(key string) string {
	SignKey = key
	return SignKey
}
func (j *JWT) CreateToken(claims CustomClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.SigningKey)
}
func (j *JWT) ParseToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	})

	if err != nil || token.Valid == false {
		if ve, ok := err.(*jwt.ValidationError); ok {
			if ve.Errors&jwt.ValidationErrorMalformed != 0 {
				return nil, TokenMalformed
			} else if ve.Errors&jwt.ValidationErrorExpired != 0 {
				// Token is expired
				return nil, TokenExpired
			} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
				return nil, TokenNotValidYet
			} else {
				return nil, TokenInvalid
			}
		}
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, TokenInvalid
}
func (j *JWT) RefreshToken(tokenString string) (string, error) {
	jwt.TimeFunc = func() time.Time {
		return time.Unix(0, 0)
	}
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return j.SigningKey, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		jwt.TimeFunc = time.Now
		claims.StandardClaims.ExpiresAt = time.Now().Add((models.TOKEN_EXP)*time.Hour).Unix()
		return j.CreateToken(*claims)
	}
	return "", TokenInvalid
}
