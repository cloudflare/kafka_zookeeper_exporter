package main

func stringInSlice(s string, arr []string) bool {
	for _, k := range arr {
		if k == s {
			return true
		}
	}
	return false
}

func int32InSlice(i int32, arr []int32) bool {
	for _, k := range arr {
		if k == i {
			return true
		}
	}
	return false
}
