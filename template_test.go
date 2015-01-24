package main

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestContains(t *testing.T) {
	env := map[string]string{
		"PORT": "1234",
	}

	if !contains(env, "PORT") {
		t.Fail()
	}

	if contains(env, "MISSING") {
		t.Fail()
	}
}

func TestKeys(t *testing.T) {
	expected := "VIRTUAL_HOST"
	env := map[string]string{
		expected: "demo.local",
	}

	k, err := keys(env)
	if err != nil {
		t.Fatalf("Error fetching keys: %v", err)
	}
	vk := reflect.ValueOf(k)
	if vk.Kind() == reflect.Invalid {
		t.Fatalf("Got invalid kind for keys: %v", vk)
	}

	if len(env) != vk.Len() {
		t.Fatalf("Incorrect key count; expected %s, got %s", len(env), vk.Len())
	}

	got := vk.Index(0).Interface()
	if expected != got {
		t.Fatalf("Incorrect key found; expected %s, got %s", expected, got)
	}
}

func TestKeysEmpty(t *testing.T) {
	input := map[string]int{}

	k, err := keys(input)
	if err != nil {
		t.Fatalf("Error fetching keys: %v", err)
	}
	vk := reflect.ValueOf(k)
	if vk.Kind() == reflect.Invalid {
		t.Fatalf("Got invalid kind for keys: %v", vk)
	}

	if len(input) != vk.Len() {
		t.Fatalf("Incorrect key count; expected %s, got %s", len(input), vk.Len())
	}
}

func TestKeysNil(t *testing.T) {
	k, err := keys(nil)
	if err != nil {
		t.Fatalf("Error fetching keys: %v", err)
	}
	vk := reflect.ValueOf(k)
	if vk.Kind() != reflect.Invalid {
		t.Fatalf("Got invalid kind for keys: %v", vk)
	}
}

func TestGroupByExistingKey(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}

	groups := groupBy(containers, "Env.VIRTUAL_HOST")
	if len(groups) != 2 {
		t.Fail()
	}

	if len(groups["demo1.localhost"]) != 2 {
		t.Fail()
	}

	if len(groups["demo2.localhost"]) != 1 {
		t.Fail()
	}
	if groups["demo2.localhost"][0].ID != "3" {
		t.Fail()
	}
}

func TestGroupByMulti(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}

	groups := groupByMulti(containers, "Env.VIRTUAL_HOST", ",")
	if len(groups) != 3 {
		t.Fatalf("expected 3 got %d", len(groups))
	}

	if len(groups["demo1.localhost"]) != 2 {
		t.Fatalf("expected 2 got %s", len(groups["demo1.localhost"]))
	}

	if len(groups["demo2.localhost"]) != 1 {
		t.Fatalf("expected 1 got %s", len(groups["demo2.localhost"]))
	}
	if groups["demo2.localhost"][0].ID != "3" {
		t.Fatalf("expected 2 got %s", groups["demo2.localhost"][0].ID)
	}
	if len(groups["demo3.localhost"]) != 1 {
		t.Fatalf("expect 1 got %d", len(groups["demo3.localhost"]))
	}
	if groups["demo3.localhost"][0].ID != "2" {
		t.Fatalf("expected 2 got %s", groups["demo3.localhost"][0].ID)
	}
}

func TestHasPrefix(t *testing.T) {
	const prefix = "tcp://"
	const str = "tcp://127.0.0.1:2375"
	if !hasPrefix(prefix, str) {
		t.Fatalf("expected %s to have prefix %s", str, prefix)
	}
}

func TestHasSuffix(t *testing.T) {
	const suffix = ".local"
	const str = "myhost.local"
	if !hasSuffix(suffix, str) {
		t.Fatalf("expected %s to have suffix %s", str, suffix)
	}
}

func TestTrimPrefix(t *testing.T) {
	const prefix = "tcp://"
	const str = "tcp://127.0.0.1:2375"
	const trimmed = "127.0.0.1:2375"
	got := trimPrefix(prefix, str)
	if got != trimmed {
		t.Fatalf("expected trimPrefix(%s,%s) to be %s, got %s", prefix, str, trimmed, got)
	}
}

func TestTrimSuffix(t *testing.T) {
	const suffix = ".local"
	const str = "myhost.local"
	const trimmed = "myhost"
	got := trimSuffix(suffix, str)
	if got != trimmed {
		t.Fatalf("expected trimSuffix(%s,%s) to be %s, got %s", suffix, str, trimmed, got)
	}
}

func TestStringHead(t *testing.T) {
	const prefix = "tcp://"
	const prefix_len = len("tcp://")
	const str = "tcp://127.0.0.1:2375"
	strHead := stringHead(str, prefix_len)
	if strHead != prefix {
		t.Fatalf("expected first %d chars of %s to be %s, got '%s'",
			prefix_len, str, prefix, strHead)
	}
}

func TestStringTail(t *testing.T) {
	const suffix = "hats"
	const suffix_len = len(suffix)
	const str = "catscatscatsinhats"
	strTail := stringTail(str, suffix_len)
	if strTail != suffix {
		t.Fatalf("expected last %d chars of %s to be %s, got '%s'",
			suffix_len, str, suffix, strTail)
	}
}

func TestDict(t *testing.T) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}
	d, err := dict("/", containers)
	if err != nil {
		t.Fatal(err)
	}
	if d["/"] == nil {
		t.Fatalf("did not find containers in dict: %s", d)
	}
	if d["MISSING"] != nil {
		t.Fail()
	}
}

func TestSha1(t *testing.T) {
	sum := hashSha1("/path")
	if sum != "4f26609ad3f5185faaa9edf1e93aa131e2131352" {
		t.Fatal("Incorrect SHA1 sum")
	}
}

type marshalFunc func(interface{}) (string, error)

func _TestJsonInternal(t *testing.T, intFunc marshalFunc) {
	containers := []*RuntimeContainer{
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost",
			},
			ID: "1",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo1.localhost,demo3.localhost",
			},
			ID: "2",
		},
		&RuntimeContainer{
			Env: map[string]string{
				"VIRTUAL_HOST": "demo2.localhost",
			},
			ID: "3",
		},
	}
	output, err := intFunc(containers)
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.NewBufferString(output)
	dec := json.NewDecoder(buf)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []*RuntimeContainer
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != len(containers) {
		t.Fatal("Incorrect unmarshaled container count. Expected %d, got %d.", len(containers), len(decoded))
	}
}

func TestPrettyJson(t *testing.T) {
	_TestJsonInternal(t, marshalJsonPretty)
}
func TestJson(t *testing.T) {
	_TestJsonInternal(t, marshalJson)
}

func TestArrayClosestExact(t *testing.T) {
	if arrayClosest([]string{"foo.bar.com", "bar.com"}, "foo.bar.com") != "foo.bar.com" {
		t.Fatal("Expected foo.bar.com")
	}
}

func TestArrayClosestSubstring(t *testing.T) {
	if arrayClosest([]string{"foo.fo.com", "bar.com"}, "foo.bar.com") != "bar.com" {
		t.Fatal("Expected bar.com")
	}
}

func TestArrayClosestNoMatch(t *testing.T) {
	if arrayClosest([]string{"foo.fo.com", "bip.com"}, "foo.bar.com") != "" {
		t.Fatal("Expected ''")
	}
}

func TestHostEnviron(t *testing.T) {
	envKey := "BIBBLE"
	envVal := "I AM A WOMBAT"
	// check it's empty before we set it
	retEnvVal := hostEnviron(envKey)
	if retEnvVal != "" {
		t.Fatal("Expected %s='', got %s=%s", envKey, envVal, envKey, retEnvVal)
	}
	// set it
	if err := os.Setenv(envKey, envVal); err != nil {
		t.Fatal("Unable to set test env var '%s', err=%v", envKey, err)
	}
	// confirm it's fetched correctly
	retEnvVal = hostEnviron(envKey)
	if retEnvVal != envVal {
		t.Fatal("Expected %s=%s, got %s=%s", envKey, envVal, envKey, retEnvVal)
	}
}
