package domain

import "testing"

func TestValidateDestination(t *testing.T) {
	cases := []struct { name, input string; valid bool }{
		{name:"https",input:"https://example.com/path",valid:true},
		{name:"http",input:"http://example.com",valid:true},
		{name:"missing host",input:"https:///path"},
		{name:"unsupported scheme",input:"javascript:alert(1)"},
		{name:"credentials",input:"https://user:pass@example.com"},
		{name:"empty",input:""},
	}
	for _,tc:=range cases{t.Run(tc.name,func(t *testing.T){got,err:=ValidateDestination(tc.input);if tc.valid&&err!=nil{t.Fatalf("unexpected error: %v",err)};if !tc.valid&&err==nil{t.Fatalf("expected invalid URL, got %q",got)}})}
}

func TestValidateCode(t *testing.T) {
	for _,tc:=range []struct{code string;valid bool}{{"abcd",true},{"a_B-9",true},{"abc",false},{"space bad",false},{"abc/def",false},{"abcdefghijklmnopqrstuvwxyz123456789",false}}{if got:=ValidateCode(tc.code);got!=tc.valid{t.Errorf("ValidateCode(%q)=%v want %v",tc.code,got,tc.valid)}}
}
