package errors

import "net/http"

type Code int32

const (
	CodeOK       Code = 0
	SystemErrors Code = 500

	CodeInvalidParam Code = 10001

	CodeUnauthorized Code = 20001
	CodeForbidden    Code = 20002
	CodeNotFound     Code = 20003
	CodeConflict     Code = 20004

	CodeRateLimited    Code = 40001
	CodeRequestTimeout Code = 50002
	CodeUnknown        Code = 50000
	CodeInternal       Code = 50001
	ErrNilRepo         Code = 50003
	ErrInvalidTenantID Code = 50004

	// 风控/认证细分（示例）
	CodeFirewallForbidden Code = 40010
	CodeFirewallLimited   Code = 40011
	CodeAntiBrush         Code = 40012

	CodeAuthNotConfigured       Code = 50010
	CodeAuthInvalidToken        Code = 20010
	CodeAuthTokenExpired        Code = 20011
	CodeAuthTokenRevoked        Code = 20012
	CodeAuthProviderNotAllowed  Code = 20013
	CodeAuthRefreshTokenInvalid Code = 20020
	CodeAuthRefreshTokenExpired Code = 20021
	CodeAuthRefreshTokenRevoked Code = 20022

	//订阅相关错误
	CodeSubscriptionLimitExceeded Code = 400101 // 免费版订阅数量已达上限
	CodeSubscriptionNotFound      Code = 400102 // 订阅不存在
	CodeSubscriptionDeleted       Code = 400103 // 订阅已删除
	CodeSubscriptionInvalidAmount Code = 400104 // 金额非法
	CodeSubscriptionInvalidDate   Code = 400105 // 日期非法
	CodeSubscriptionInvalidCycle  Code = 400106 // 周期非法
	CodeSubscriptionInvalidStatus Code = 400107 // 状态非法
	CodeSubscriptionInvalidName   Code = 400108 // 服务名非法
	CodeSubscriptionInvalidRemind Code = 400109 // 提醒参数非法
	CodeUserNotFound              Code = 400110 // 用户不存在
)

func HTTPStatusByCode(code Code) int {
	switch {
	case code == CodeOK:
		return http.StatusOK
	case code >= 10000 && code < 20000:
		return http.StatusBadRequest
	case code == CodeForbidden:
		return http.StatusForbidden
	case code >= 20000 && code < 30000:
		return http.StatusUnauthorized
	case code == CodeNotFound:
		return http.StatusNotFound
	case code == CodeConflict:
		return http.StatusConflict
	case code >= 40000 && code < 50000:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
