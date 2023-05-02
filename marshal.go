package bars

import "encoding/json"

func Marshal(obj interface{}) ([]byte, error) {
	return json.Marshal(obj)
}
