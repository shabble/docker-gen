package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"text/template"
)

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func groupByMulti(entries []*RuntimeContainer, key, sep string) map[string][]*RuntimeContainer {
	groups := make(map[string][]*RuntimeContainer)
	for _, v := range entries {
		value := deepGet(*v, key)
		if value != nil {
			items := strings.Split(value.(string), sep)
			for _, item := range items {
				groups[item] = append(groups[item], v)
			}

		}
	}
	return groups
}

// groupBy groups a list of *RuntimeContainers by the path property key
func groupBy(entries []*RuntimeContainer, key string) map[string][]*RuntimeContainer {
	groups := make(map[string][]*RuntimeContainer)
	for _, v := range entries {
		value := deepGet(*v, key)
		if value != nil {
			groups[value.(string)] = append(groups[value.(string)], v)
		}
	}
	return groups
}

// groupByKeys is the same as groupBy but only returns a list of keys
func groupByKeys(entries []*RuntimeContainer, key string) []string {
	groups := groupBy(entries, key)
	ret := []string{}
	for k, _ := range groups {
		ret = append(ret, k)
	}
	return ret
}

// hasPrefix returns whether a given string is a prefix of another string
func hasPrefix(prefix, s string) bool {
	return strings.HasPrefix(s, prefix)
}

// hasSuffix returns whether a given string is a suffix of another string
func hasSuffix(suffix, s string) bool {
	return strings.HasSuffix(s, suffix)
}

func keys(input interface{}) (interface{}, error) {
	if input == nil {
		return nil, nil
	}

	val := reflect.ValueOf(input)
	if val.Kind() != reflect.Map {
		return nil, fmt.Errorf("Cannot call keys on a non-map value: %v", input)
	}

	vk := val.MapKeys()
	k := make([]interface{}, val.Len())
	for i, _ := range k {
		k[i] = vk[i].Interface()
	}

	return k, nil
}

func contains(item map[string]string, key string) bool {
	if _, ok := item[key]; ok {
		return true
	}
	return false
}

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func hashSha1(input string) string {
	h := sha1.New()
	io.WriteString(h, input)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func marshalJsonInternal(input interface{}, prettify bool) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(input); err != nil {
		return "", err
	}
	if prettify {
		var out bytes.Buffer
		err := json.Indent(&out, buf.Bytes(), "", "    ")
		return out.String(), err
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

func marshalJson(input interface{}) (string, error) {
	return marshalJsonInternal(input, false)
}

func marshalJsonPretty(input interface{}) (string, error) {
	return marshalJsonInternal(input, true)
}

// arrayFirst returns first item in the array or nil if the
// input is nil or empty
func arrayFirst(input interface{}) interface{} {
	if input == nil {
		return nil
	}

	arr := reflect.ValueOf(input)

	if arr.Len() == 0 {
		return nil
	}

	return arr.Index(0).Interface()
}

// arrayLast returns last item in the array
func arrayLast(input interface{}) interface{} {
	arr := reflect.ValueOf(input)
	return arr.Index(arr.Len() - 1).Interface()
}

// arrayClosest find the longest matching substring in values
// that matches input
func arrayClosest(values []string, input string) string {
	best := ""
	for _, v := range values {
		if strings.Contains(input, v) && len(v) > len(best) {
			best = v
		}
	}
	return best
}

// dirList returns a list of files in the specified path
func dirList(path string) ([]string, error) {
	names := []string{}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return names, err
	}
	for _, f := range files {
		names = append(names, f.Name())
	}
	return names, nil
}

// coalesce returns the first non nil argument
func coalesce(input ...interface{}) interface{} {
	for _, v := range input {
		if v != nil {
			return v
		}
	}
	return nil
}

// trimPrefix returns whether a given string is a prefix of another string
func trimPrefix(prefix, s string) string {
	return strings.TrimPrefix(s, prefix)
}

// trimSuffix returns whether a given string is a suffix of another string
func trimSuffix(suffix, s string) string {
	return strings.TrimSuffix(s, suffix)
}

func stringHead(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length]
}

func stringTail(s string, length int) string {
	if len(s) <= length {
		return ""
	}
	return s[len(s)-length:]
}

func hostEnviron(envKey string) string {
	return os.Getenv(envKey)
}
func generateFile(config Config, containers Context) bool {
	templatePath := config.Template
	tmpl, err := template.New(filepath.Base(templatePath)).Funcs(template.FuncMap{
		"closest":      arrayClosest,
		"coalesce":     coalesce,
		"contains":     contains,
		"dict":         dict,
		"dir":          dirList,
		"exists":       exists,
		"first":        arrayFirst,
		"groupBy":      groupBy,
		"groupByKeys":  groupByKeys,
		"groupByMulti": groupByMulti,
		"hasPrefix":    hasPrefix,
		"hasSuffix":    hasSuffix,
		"json":         marshalJson,
		"jsonPretty":   marshalJsonPretty,
		"keys":         keys,
		"last":         arrayLast,
		"replace":      strings.Replace,
		"sha1":         hashSha1,
		"split":        strings.Split,
		"trimPrefix":   trimPrefix,
		"trimSuffix":   trimSuffix,
		"stringHead":   stringHead,
		"stringTail":   stringTail,
		"hostEnviron":  hostEnviron,
	}).ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("unable to parse template: %s", err)
	}

	filteredContainers := Context{}
	if config.OnlyPublished {
		for _, container := range containers {
			if len(container.PublishedAddresses()) > 0 {
				filteredContainers = append(filteredContainers, container)
			}
		}
	} else if config.OnlyExposed {
		for _, container := range containers {
			if len(container.Addresses) > 0 {
				filteredContainers = append(filteredContainers, container)
			}
		}
	} else {
		filteredContainers = containers
	}

	dest := os.Stdout
	if config.Dest != "" {
		dest, err = ioutil.TempFile(filepath.Dir(config.Dest), "docker-gen")
		defer func() {
			dest.Close()
			os.Remove(dest.Name())
		}()
		if err != nil {
			log.Fatalf("unable to create temp file: %s\n", err)
		}
	}

	var buf bytes.Buffer
	multiwriter := io.MultiWriter(dest, &buf)
	err = tmpl.ExecuteTemplate(multiwriter, filepath.Base(templatePath), &filteredContainers)
	if err != nil {
		log.Fatalf("template error: %s\n", err)
	}

	if config.Dest != "" {

		contents := []byte{}
		if fi, err := os.Stat(config.Dest); err == nil {
			if err := dest.Chmod(fi.Mode()); err != nil {
				log.Fatalf("unable to chmod temp file: %s\n", err)
			}
			if err := dest.Chown(int(fi.Sys().(*syscall.Stat_t).Uid), int(fi.Sys().(*syscall.Stat_t).Gid)); err != nil {
				log.Fatalf("unable to chown temp file: %s\n", err)
			}
			contents, err = ioutil.ReadFile(config.Dest)
			if err != nil {
				log.Fatalf("unable to compare current file contents: %s: %s\n", config.Dest, err)
			}
		}

		if bytes.Compare(contents, buf.Bytes()) != 0 {
			err = os.Rename(dest.Name(), config.Dest)
			if err != nil {
				log.Fatalf("unable to create dest file %s: %s\n", config.Dest, err)
			}
			log.Printf("Generated '%s' from %d containers", config.Dest, len(filteredContainers))
			return true
		}
		return false
	}
	return true
}
