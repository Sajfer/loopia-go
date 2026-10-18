package loopia

import (
	"fmt"
	"strconv"
)

// LoopiaAPI is not always consistent about the XML-RPC types it uses for a
// field (e.g. 0/1 integers vs booleans, integers vs doubles), so loosely typed
// fields are decoded into any and converted with these helpers.

func toBool(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case int64:
		return x != 0
	case float64:
		return x != 0
	case string:
		b, _ := strconv.ParseBool(x)
		return b
	}
	return false
}

func toFloat(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case int64:
		return float64(x)
	case string:
		f, _ := strconv.ParseFloat(x, 64)
		return f
	}
	return 0
}

func toString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	}
	return fmt.Sprint(v)
}
