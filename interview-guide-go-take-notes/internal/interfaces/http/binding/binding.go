// Package binding 提供泛型 HTTP 包装；核心入口 Handle：bindRequest + Validate + 业务函数，成功写 Result。
// bindRequest 支持 application/json、multipart/form-data（form 标签，见 bindMultipart 注释）及 path、query。Exec 无入参。
package binding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"interview-guide-go/shared/response"

	"github.com/go-chi/chi/v5"
)

const (
	defaultMaxBody      int64 = 4 << 20
	defaultMultipartMax int64 = 32 << 20
)

// Option 配置请求体大小等。
type Option func(*config)

type config struct {
	maxBody            int64
	multipartMaxMemory int64
}

// MaxBody 设置 JSON 请求体最大字节数（默认 4MB，仅对 Content-Type: application/json 的 POST/PUT/PATCH 生效）。
func MaxBody(n int64) Option {
	return func(c *config) { c.maxBody = n }
}

// MultipartMaxMemory 设置 ParseMultipartForm 的 maxMemory（默认 32MB，仅对 Content-Type: multipart/form-data 生效）；
// 超过该值的文件部分会落临时文件，与标准库行为一致。
func MultipartMaxMemory(n int64) Option {
	return func(c *config) { c.multipartMaxMemory = n }
}

// Handle 先 BindRequest，再 Validate，再执行业务函数。GET 仅走 path/query；POST/PUT/PATCH 支持 application/json 或
// multipart/form-data（form 标签），最后统一 bind path/query。
func Handle[Req, Resp any](fn func(context.Context, Req) (Resp, error), opts ...Option) http.HandlerFunc {
	cfg := &config{maxBody: defaultMaxBody, multipartMaxMemory: defaultMultipartMax}
	for _, o := range opts {
		o(cfg)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := bindRequest(w, r, &req, cfg); err != nil {
			writeAsHTTPResult(w, err)
			return
		}
		if err := Validate(req); err != nil {
			writeAsHTTPResult(w, err)
			return
		}
		resp, err := fn(r.Context(), req)
		if err != nil {
			writeErr(w, err)
			return
		}
		response.WriteJSON(w, http.StatusOK, response.Success(resp))
	}
}

// Exec 无入参、不需要绑定与校验的端点（如健康检查、静态枚举类 GET）。
func Exec[Resp any](fn func(context.Context) (Resp, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := fn(r.Context())
		if err != nil {
			writeErr(w, err)
			return
		}
		response.WriteJSON(w, http.StatusOK, response.Success(resp))
	}
}

// bindRequest：POST/PUT/PATCH 时按 Content-Type 选 JSON、multipart 或忽略 body；再绑定 path、query。
func bindRequest(w http.ResponseWriter, r *http.Request, dst any, cfg *config) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return response.Err(http.StatusInternalServerError, "bind 目标须为非空指针")
	}
	if v.Elem().Kind() != reflect.Struct {
		return response.Err(http.StatusInternalServerError, "bind 目标须为指向结构体的指针")
	}

	if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch {
		ct := r.Header.Get("Content-Type")
		switch {
		case strings.Contains(ct, "application/json"):
			if r.Body != nil {
				defer r.Body.Close()
			}
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, cfg.maxBody))
			if err := dec.Decode(dst); err != nil {
				if errors.Is(err, io.EOF) {
					return response.Err(http.StatusBadRequest, "请求体不能为空")
				}
				var maxErr *http.MaxBytesError
				if errors.As(err, &maxErr) {
					return response.Err(http.StatusRequestEntityTooLarge, "请求体过大")
				}
				return response.Err(http.StatusBadRequest, "JSON 格式无效")
			}
		case strings.HasPrefix(ct, "multipart/") && strings.Contains(ct, "form-data"):
			if err := bindMultipart(r, dst, cfg.multipartMaxMemory); err != nil {
				return err
			}
		}
	}
	return bindParams(r, dst)
}

// bindMultipart 将 multipart/form-data 按 form 标签写入：[]byte 视为文件体（form 为字段名）；
// 同一次上传中从 FileHeader 填写 Filename、ContentType 字段（按结构体字段名识别）；string 使用 FormValue。
func bindMultipart(r *http.Request, dst any, maxMemory int64) error {
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		return response.Err(http.StatusBadRequest, "invalid multipart form: "+err.Error())
	}
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		formName := field.Tag.Get("form")
		if formName == "" || formName == "-" {
			continue
		}
		if fv.Kind() == reflect.Slice && fv.Type().Elem().Kind() == reflect.Uint8 {
			fh, hdr, err := r.FormFile(formName)
			if err != nil {
				return response.Err(http.StatusBadRequest, "missing or invalid file field (use name="+formName+"): "+err.Error())
			}
			b, err := io.ReadAll(fh)
			_ = fh.Close()
			if err != nil {
				return response.Err(http.StatusBadRequest, "read upload failed: "+err.Error())
			}
			fv.SetBytes(b)
			fillFileMetadataFromHeader(v, t, hdr)
			continue
		}
		if fv.Kind() == reflect.String {
			fv.SetString(strings.TrimSpace(r.FormValue(formName)))
		}
	}
	return nil
}

func fillFileMetadataFromHeader(v reflect.Value, t reflect.Type, hdr *multipart.FileHeader) {
	if hdr == nil {
		return
	}
	filename := strings.TrimSpace(hdr.Filename)
	ct := strings.TrimSpace(hdr.Header.Get("Content-Type"))
	if ct == "" {
		ct = "application/octet-stream"
	}
	for j := 0; j < t.NumField(); j++ {
		fj := t.Field(j)
		fjv := v.Field(j)
		if !fjv.CanSet() {
			continue
		}
		switch fj.Name {
		case "Filename":
			if fjv.Kind() == reflect.String {
				fjv.SetString(filename)
			}
		case "ContentType":
			if fjv.Kind() == reflect.String {
				fjv.SetString(ct)
			}
		}
	}
}

// Validate 最简规则：若字段 tag 含 validate:"required"（或子串 "required"），且为零值，则 400。
// 数值 0、空字符串、nil 指针等均为零值，注意 int 的 0 会判为未填（与 go-playground/validator 常见用法不同
// 需业务自行用指针区分「未传」和「0」时可改用 *int）。
func Validate(req any) error {
	v := reflect.ValueOf(req)
	t := reflect.TypeOf(req)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
		t = t.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("validate")
		if !strings.Contains(tag, "required") {
			continue
		}
		fv := v.Field(i)
		if isMissingRequired(fv) {
			name := field.Name
			if j := field.Tag.Get("json"); j != "" {
				name = strings.Split(j, ",")[0]
			} else if p := field.Tag.Get("path"); p != "" {
				name = p
			} else if q := field.Tag.Get("query"); q != "" {
				name = q
			} else if f := field.Tag.Get("form"); f != "" {
				name = f
			}
			return response.Err(http.StatusBadRequest, fmt.Sprintf("%s 为必填", name))
		}
	}
	return nil
}

// isMissingRequired 用于 validate:required：[]byte 空切片/ nil 视为未提供。
func isMissingRequired(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	if v.Kind() == reflect.Slice && v.Type().Elem().Kind() == reflect.Uint8 {
		return v.Len() == 0
	}
	return v.IsZero()
}

// writeAsHTTPResult 将错误写入 HTTP 响应
func writeAsHTTPResult(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	var he *response.Error
	if errors.As(err, &he) {
		response.ErrJSON(w, he.Code, he.Message)
		return
	}
	response.WriteErr(w, err)
}

// writeErr 将错误写入 HTTP 响应
func writeErr(w http.ResponseWriter, err error) {
	var be *response.BizError
	if errors.As(err, &be) {
		response.WriteJSON(w, http.StatusOK, response.Result{
			Code:    be.Code,
			Message: be.Message,
			Data:    nil,
		})
		return
	}
	response.WriteErr(w, err)
}

// bindParams 将 chi 路径参数与 query 写入 struct。
func bindParams(r *http.Request, dst any) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return nil
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()
	rctx := chi.RouteContext(r.Context())
	query := r.URL.Query()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		if tag := field.Tag.Get("path"); tag != "" {
			raw := ""
			if rctx != nil {
				raw = rctx.URLParam(tag)
			}
			if err := setField(fv, raw, field.Name); err != nil {
				return err
			}
		}
		if tag := field.Tag.Get("query"); tag != "" {
			if raw := query.Get(tag); raw != "" {
				if err := setField(fv, raw, field.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// setField 设置字段值
func setField(fv reflect.Value, raw string, name string) error {
	if raw == "" {
		return nil
	}
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return response.Err(http.StatusBadRequest, fmt.Sprintf("参数 %s 格式无效", name))
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return response.Err(http.StatusBadRequest, fmt.Sprintf("参数 %s 格式无效", name))
		}
		fv.SetUint(n)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return response.Err(http.StatusBadRequest, fmt.Sprintf("参数 %s 格式无效", name))
		}
		fv.SetBool(b)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return response.Err(http.StatusBadRequest, fmt.Sprintf("参数 %s 格式无效", name))
		}
		fv.SetFloat(f)
	case reflect.Ptr:
		elem := reflect.New(fv.Type().Elem())
		if err := setField(elem.Elem(), raw, name); err != nil {
			return err
		}
		fv.Set(elem)
	default:
		return response.Err(http.StatusBadRequest, fmt.Sprintf("参数 %s: 不支持的类型", name))
	}
	return nil
}
