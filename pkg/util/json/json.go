// Package json is originally taken from https://github.com/argoproj/gitops-engine
// https://github.com/argoproj/gitops-engine/blob/master/pkg/utils/json/json.go
// Originally taken from argoproj gitops-engine (Copyright Apache 2.0)
package json

// https://github.com/ksonnet/ksonnet/blob/master/pkg/kubecfg/diff.go
func removeFields(config, live interface{}) interface{} {
	switch c := config.(type) {
	case map[string]interface{}:
		l, ok := live.(map[string]interface{})
		if ok {
			return RemoveMapFields(c, l)
		}
		return live
	case []interface{}:
		l, ok := live.([]interface{})
		if ok {
			return RemoveListFields(c, l)
		}
		return live
	default:
		return live
	}

}

// RemoveMapFields remove all non-existent fields in the live that don't exist in the config
func RemoveMapFields(config, live map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v1 := range config {
		v2, ok := live[k]
		if !ok {
			continue
		}
		if v2 != nil {
			v2 = removeFields(v1, v2)
		}
		result[k] = v2
	}
	return result
}

// RemoveListFields removes matching list fields from the given config and live
// interfaces.
func RemoveListFields(config, live []interface{}) []interface{} {
	// If live is longer than config, then the extra elements at the end of the
	// list will be returned as-is so they appear in the diff.
	result := make([]interface{}, 0, len(live))
	for i, v2 := range live {
		if len(config) > i {
			if v2 != nil {
				v2 = removeFields(config[i], v2)
			}
			result = append(result, v2)
		} else {
			result = append(result, v2)
		}
	}
	return result
}
