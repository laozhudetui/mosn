package log

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"strconv"
	"strings"
)

var (
	RequestInfoFuncMap map[string]func(info types.RequestInfo) string
)

func init() {
	RequestInfoFuncMap = map[string]func(info types.RequestInfo) string{
		types.LogStartTime:                  StartTimeGetter,
		types.LogRequestReceivedDuration:    ReceivedDurationGetter,
		types.LogResponseReceivedDuration:   ResponseReceivedDurationGetter,
		types.LogBytesSent:                  BytesSentGetter,
		types.LogBytesReceived:              BytesReceivedGetter,
		types.LogProtocol:                   ProtocolGetter,
		types.LogResponseCode:               ResponseCodeGetter,
		types.LogDuration:                   DurationGetter,
		types.LogResponseFlag:               GetResponseFlagGetter,
		types.LogUpstreamLocalAddress:       UpstreamLocalAddressGetter,
		types.LogDownstreamLocalAddress:     DownstreamLocalAddressGetter,
		types.LogDownstreamRemoteAddress:    DownstreamRemoteAddressGetter,
		types.LogUpstreamHostSelectedGetter: UpstreamHostSelectedGetter,
	}
}

// types.AccessLog
type accesslog struct {
	output    string
	filter    types.AccessLogFilter
	formatter types.AccessLogFormatter
	logger    Logger
}

func NewAccessLog(output string, filter types.AccessLogFilter,
	format string) (types.AccessLog, error) {
	var err error
	var logger Logger

	if logger, err = GetLoggerInstance(output, 0); err != nil {
		return nil, err
	}

	return &accesslog{
		output:    output,
		filter:    filter,
		formatter: NewAccessLogFormatter(format),
		logger:    logger,
	}, nil
}

func (l *accesslog) Log(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) {
	if l.filter != nil {
		if !l.filter.Decide(reqHeaders, requestInfo) {
			return
		}
	}

	l.logger.Println(l.formatter.Format(reqHeaders, respHeaders, requestInfo))
}

// types.AccessLogFormatter
type accesslogformatter struct {
	formatters []types.AccessLogFormatter
}

func NewAccessLogFormatter(format string) types.AccessLogFormatter {
	if format == "" {
		format = types.DefaultAccessLogFormat
	}

	return &accesslogformatter{
		formatters: formatToFormatter(format),
	}
}

func (f *accesslogformatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	var log string

	for _, formatter := range f.formatters {
		log += formatter.Format(reqHeaders, respHeaders, requestInfo)
	}

	//delete the final " "
	if len(log) > 0 {
		log = log[:len(log)-1]
	}

	return log
}

// types.AccessLogFormatter
type simpleRequestInfoFormatter struct {
	reqInfoFormat []string
}

func (f *simpleRequestInfoFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	// todo: map fieldName to field vale string
	if f.reqInfoFormat == nil {
		DefaultLogger.Debugf("No ReqInfo Format Keys Input")
		return ""
	}

	format := ""
	for _, key := range f.reqInfoFormat {

		if vFunc, ok := RequestInfoFuncMap[key]; ok {
			format = format + vFunc(requestInfo) + " "
		} else {
			DefaultLogger.Debugf("Invalid ReqInfo Format Keys: %s", key)
		}
	}
	return format
}

// types.AccessLogFormatter
type simpleReqHeadersFormatter struct {
	reqHeaderFormat []string
}

func (f *simpleReqHeadersFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {

	if f.reqHeaderFormat == nil {
		DefaultLogger.Debugf("No ReqHeaders Format Keys Input")
		return ""
	}
	format := ""

	for _, key := range f.reqHeaderFormat {
		if v, ok := reqHeaders[key]; ok {
			format = format + types.ReqHeaderPrefix + v + " "
		} else {
			DefaultLogger.Debugf("Invalid ReqHeaders Format Keys: %s", key)
		}
	}

	return format
}

// types.AccessLogFormatter
type simpleRespHeadersFormatter struct {
	respHeaderFormat []string
}

func (f *simpleRespHeadersFormatter) Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo types.RequestInfo) string {
	if f.respHeaderFormat == nil {
		DefaultLogger.Debugf("No RespHeaders Format Keys Input")
		return ""
	}

	format := ""
	for _, key := range f.respHeaderFormat {

		if v, ok := respHeaders[key]; ok {
			format = format + types.RespHeaderPrefix + v + " "
		} else {
			DefaultLogger.Debugf("Invalid RespHeaders Format Keys:%s", key)
		}
	}

	return format
}

// format to formatter by parsing format
func formatToFormatter(format string) []types.AccessLogFormatter {

	strArray := strings.Split(format, " ")

	// delete %
	for i := 0; i < len(strArray); i++ {
		strArray[i] = strArray[i][1 : len(strArray[i])-1]
	}

	// classify keys
	var reqInfoArray, reqHeaderArray, respHeaderArray []string
	for _, s := range strArray {
		if strings.HasPrefix(s, types.ReqHeaderPrefix) {
			reqHeaderArray = append(reqHeaderArray, s)

		} else if strings.HasPrefix(s, types.RespHeaderPrefix) {
			respHeaderArray = append(respHeaderArray, s)
		} else {
			reqInfoArray = append(reqInfoArray, s)
		}
	}

	// delete REQ.
	if reqHeaderArray != nil {
		for i := 0; i < len(reqHeaderArray); i++ {
			reqHeaderArray[i] = reqHeaderArray[i][len(types.ReqHeaderPrefix):]
		}
	}

	// delete RESP.
	if respHeaderArray != nil {
		for i := 0; i < len(respHeaderArray); i++ {
			respHeaderArray[i] = respHeaderArray[i][len(types.RespHeaderPrefix):]
		}
	}

	return []types.AccessLogFormatter{
		&simpleRequestInfoFormatter{reqInfoFormat: reqInfoArray},
		&simpleReqHeadersFormatter{reqHeaderFormat: reqHeaderArray},
		&simpleRespHeadersFormatter{respHeaderFormat: respHeaderArray},
	}
}

// get request's arriving time
func StartTimeGetter(info types.RequestInfo) string {
	return info.StartTime().String()
}

// get duration between request arriving and request resend to upstream
func ReceivedDurationGetter(info types.RequestInfo) string {
	return info.RequestReceivedDuration().String()
}

// get duration between request arriving and response sending
func ResponseReceivedDurationGetter(info types.RequestInfo) string {
	return info.ResponseReceivedDuration().String()
}

// get bytes sent
func BytesSentGetter(info types.RequestInfo) string {
	return strconv.FormatUint(info.BytesSent(), 10)
}

// get bytes received
func BytesReceivedGetter(info types.RequestInfo) string {
	return strconv.FormatUint(info.BytesReceived(), 10)
}

// get request's protocol type
func ProtocolGetter(info types.RequestInfo) string {
	return string(info.Protocol())
}

// get request's response code
func ResponseCodeGetter(info types.RequestInfo) string {
	return strconv.FormatUint(uint64(info.ResponseCode()), 10)
}

// get duration since request's starting time
func DurationGetter(info types.RequestInfo) string {
	return info.Duration().String()
}

// get request's response flag
func GetResponseFlagGetter(info types.RequestInfo) string {
	return strconv.FormatBool(info.GetResponseFlag(0))
}

// get upstream's local address
func UpstreamLocalAddressGetter(info types.RequestInfo) string {
	if info.UpstreamLocalAddress() != nil {
		return info.UpstreamLocalAddress().String()
	}
	return "nil"
}

// get downstream's local address
func DownstreamLocalAddressGetter(info types.RequestInfo) string {
	if info.DownstreamLocalAddress() != nil {
		return info.DownstreamLocalAddress().String()
	}
	return "nil"
}

// get upstream's remote address
func DownstreamRemoteAddressGetter(info types.RequestInfo) string {
	if info.DownstreamRemoteAddress() != nil {
		return info.DownstreamRemoteAddress().String()
	}
	return "nil"
}

// get upstream's selected host address
func UpstreamHostSelectedGetter(info types.RequestInfo) string {
	if info.UpstreamHost() != nil {
		return info.UpstreamHost().Hostname()
	}
	return "nil"
}