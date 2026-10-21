package code_

type ErrorCode struct {
	Code   int
	Format string
}

var (
	ServerInternalError = ErrorCode{Code: 1, Format: "服务器内部错误"}
	Canceled            = ErrorCode{Code: 160, Format: "请求被取消"}
	AuthError           = ErrorCode{Code: 401, Format: "您还未登录"}
	RequestError        = ErrorCode{Code: 400, Format: "请求错误"}
	PermissionError     = ErrorCode{Code: 403, Format: "您没有权限访问"}
	Timeout             = ErrorCode{Code: 408, Format: "请求超时"}
	Unavailable         = ErrorCode{Code: 503, Format: "服务不可用"}
	TaskNotFound        = ErrorCode{Code: 1404, Format: "任务不存在或已过期"}
	ParamIsBlank_       = ErrorCode{Code: 1001, Format: "参数%s不能为空"}
	DbFindError_        = ErrorCode{Code: 1002, Format: "数据库查询出错：%s"}
	ConvertError_       = ErrorCode{Code: 1003, Format: "数据转换出错:%s"}
	ParamError_         = ErrorCode{Code: 1004, Format: "参数%s错误"}
	ParamError          = ErrorCode{Code: 1004, Format: "参数错误"}
	ImageCodeError      = ErrorCode{Code: 1005, Format: "图片验证码错误"}
	SmsCodeError        = ErrorCode{Code: 1006, Format: "短信验证码错误"}

	DbAddError_      = ErrorCode{Code: 1011, Format: "数据添加出错:%s"}
	DbUpdateError_   = ErrorCode{Code: 1012, Format: "数据更新出错:%s"}
	DbDeleteError_   = ErrorCode{Code: 1013, Format: "数据删除出错:%s"}
	DbQueryError_    = ErrorCode{Code: 1014, Format: "数据查询出错:%s"}
	DbError_         = ErrorCode{Code: 1015, Format: "数据库执行出错:%s"}
	RecordExists_    = ErrorCode{Code: 1016, Format: "%s已存在"}
	RecordNotExists_ = ErrorCode{Code: 1017, Format: "%s不存在"}

	RpcError = ErrorCode{Code: 2001, Format: "%s"}
)
