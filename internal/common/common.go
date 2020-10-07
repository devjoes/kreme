package common

import "strings"

func LowerArray(arr *[]string) {
	for i, s := range *arr {
		(*arr)[i] = strings.ToLower(s)
	}
}
