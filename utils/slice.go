package utils

func MakeRange(min, max int) []int {
	a := make([]int, max-min+1)
	for i := range a {
		a[i] = min + i
	}
	return a
}

func InterfaceSliceToStringSlice(is []interface{}) []string {
	var ss []string
	for _, intf := range is {
		if s, ok := intf.(string); ok {
			ss = append(ss, s)
		} else {
			return nil
		}
	}
	return ss
}
