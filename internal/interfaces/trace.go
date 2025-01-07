package interfaces

// TraceManager 定义跟踪管理的核心接口
type TraceManager interface {
	StartTrace() error
	StopTrace() error
}

// TraceReader 定义读取跟踪数据的接口
type TraceReader interface {
	ReadTrace() ([]byte, error)
}

// TraceWriter 定义写入跟踪数据的接口
type TraceWriter interface {
	WriteTrace(data []byte) error
}

// OutputWriter 定义输出写入器接口
type OutputWriter interface {
	Write(content string)
	WriteError(err string)
	WriteInfo(content string)
	WriteResponse(content string)
}
