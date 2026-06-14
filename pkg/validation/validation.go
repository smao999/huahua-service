package validation

import (
	"fmt"
	"reflect"

	"github.com/go-playground/validator/v10"
)

// Translate 把 validator 的内部错误转成用户友好的中文描述
// 返回 map[字段名]提示，其他 handler 也可以复用
func Translate(err error, req interface{}) string {
	fieldErrors := string("")
	ves, ok := err.(validator.ValidationErrors)
	if !ok {
		fieldErrors = "请求格式错误，请检查 JSON 格式"
		return fieldErrors
	}
	reqType := reflect.TypeOf(req)
	if reqType.Kind() == reflect.Ptr {
		reqType = reqType.Elem()
	}
	for _, fe := range ves {
		field := resolveCN(fe.StructField(), reqType)
		switch fe.Tag() {
		case "required":
			fieldErrors = "不能为空"
		case "min":
			fieldErrors = fmt.Sprintf("%s长度不能少于 %s 个字符", field, fe.Param())
		case "max":
			fieldErrors = fmt.Sprintf("%s长度不能超过 %s 个字符", field, fe.Param())
		case "email":
			fieldErrors = "邮箱格式不正确"
		default:
			fieldErrors = fmt.Sprintf("%s校验失败: %s", field, fe.Tag())
		}
	}
	return fieldErrors
}

// resolveCN 从 struct tag 中读取 cn 标签，取不到则回退为 Go 字段名
func resolveCN(fieldName string, t reflect.Type) string {
	sf, ok := t.FieldByName(fieldName)
	if !ok {
		return fieldName
	}
	if cn := sf.Tag.Get("cn"); cn != "" {
		return cn
	}
	return fieldName
}
