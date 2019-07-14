package Rf

import (
	"testing"
	"regexp"
	"fmt"
)

func TestExcel2Json(t *testing.T){
	match,_:=regexp.MatchString("^[0-9]+","peddach")
	fmt.Println(match)
}