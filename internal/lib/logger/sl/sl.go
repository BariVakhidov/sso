package sl

import "log/slog"

func Err(er error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(er.Error()),
	}
}
