package main

import (
	"fmt"
	"regexp"
	"time"
	"unsafe"

	"github.com/valyala/fastjson"
)

func main() {
	go_filter(nil, 0, 0, 0, nil, 0) // dummy call to make sure the go_filter function is exported and keep the IDE happy
}

var (
	config_name = "fluent_bit_wasm_filter_config"
)

func logDebugOnly(msg string) {
	if false {
		fmt.Println(msg)
	}
}

type filterEntry struct {
	pattern * string;
	invert bool;
}

//export go_filter
func go_filter(tag *uint8, tag_len uint, time_sec uint, time_nsec uint, record *uint8, record_len uint) *uint8 {
	btag := unsafe.Slice(tag, tag_len)
	brecord := unsafe.Slice(record, record_len)
	now := time.Unix(int64(time_sec), int64(time_nsec))

	entry, err := NewLogEntry(string(btag), now, string(brecord))
	if err != nil {
		fmt.Println(err)
		return entry.keep_log()
	}

	config := readConfig(entry.record)
	isfilterlog := filterLog(entry.record, config)

	if isfilterlog {
		logDebugOnly("keep log")
		return entry.keep_log()
	} else {
		logDebugOnly("skip log")
		return entry.skip_log()
	}
}

type log_entry struct {
	tag    string
	time   time.Time
	record *fastjson.Value
}

func NewLogEntry(tag string, time time.Time, record string) (*log_entry, error) {
	var p fastjson.Parser
	value, err := p.Parse(record)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &log_entry{
		tag:    tag,
		time:   time,
		record: value,
	}, nil
}

func readConfig(value *fastjson.Value) ConfigFileConfiguration {
	configStr := extractString(value, config_name);
	var p fastjson.Parser
	parsedConfig, err := p.Parse(configStr)
	if err != nil {
		fmt.Println(err)
		return ConfigFileConfiguration{}
	}

	config := ConfigFileConfiguration{
		config: parsedConfig,
	}

	if(config.config != nil) {
		logDebugOnly("config: " + config.config.String())
	}

	return config
}

func (e *log_entry) keep_log() *uint8 {
	e.record.Del(config_name)
	marshalled := string(e.record.MarshalTo(nil))

	//replace control characters with their hex representation
	//we need to do this because fluent-bit does not like control characters in the resulting json
	//The marshalling to the internal msgpack will fail with an error like this:
	//  invalid JSON format. ret: -1, buf
	fixedResult := regexp.MustCompile(`[\x00-\x1F\x7F]`).ReplaceAllStringFunc(marshalled, func(s string) string {
		return fmt.Sprintf("\\x%02X", s[0])
	})

	rv := append([]byte(fixedResult), []byte(string(rune(0)))...)

	return &rv[0]
}

func (e *log_entry) skip_log() *uint8 {
	return nil
}

type Configuration interface {
	GetConfig() *fastjson.Value
}

type ConfigFileConfiguration struct {
	config *fastjson.Value
}

func (c ConfigFileConfiguration) GetConfig() *fastjson.Value {
	return c.config
}

func extractString(value *fastjson.Value, key string) string {
	if value == nil {
		return ""
	}

	v := value.Get(key)
	if v == nil {
		return ""
	}

	str, err := v.StringBytes()
	if err != nil {
		return ""
	}

	return string(str)
}

func filterLog(record *fastjson.Value, configSource Configuration) bool {
	containerName := extractString(record, "container_name")
	namespaceName := extractString(record, "namespace_name")
	fullPodName := extractString(record, "pod_name")
	log := extractString(record, "log")

	logDebugOnly("container_name: " + string(containerName))
	logDebugOnly("namespace_name: " + string(namespaceName))
	logDebugOnly("fullPodName: " + string(fullPodName))
	logDebugOnly("log: " + string(log))

	if containerName == "" || namespaceName == "" || fullPodName == "" || log == "" {
		return true // no data, keep log
	}

	podName := extractPodName(string(fullPodName))

	logDebugOnly("podName: " + string(podName))

	filter := getFilter(string(containerName), string(namespaceName), podName, configSource)

	if filter == nil {
		logDebugOnly("no filter found")
		return true // no filter found, keep log
	} else {
		logDebugOnly("filter found: " + string(*filter.pattern) + ", invert: " + fmt.Sprint(filter.invert))

		regex, err := regexp.Compile(*filter.pattern)
		if err != nil {
			fmt.Println(err)
			return true // invalid filter, keep log
		}

		isMatch := regex.MatchString(string(log))
		if filter.invert {
			return !isMatch // filter found, keep log if it does not match
		}else {
			return isMatch // filter found, keep log if it matches
		}
	}
}

func getFilter(containerName, namespaceName, podName string, configSource Configuration) *filterEntry {
	config := configSource.GetConfig()

	precedence := [][]string{
		{containerName, namespaceName, podName},
		{containerName, namespaceName, "*"},
		{containerName, "*", podName},
		{containerName, "*", "*"},
		{"*", namespaceName, podName},
		{"*", "*", podName},
		{"*", namespaceName, "*"},
		{"*", "*", "*"},
	}

	for _, t := range precedence {
		container, namespace, pod := t[0], t[1], t[2]

		v := config.Get(container).Get(namespace).Get(pod)
		if v == nil {
			continue
		}

		complexFilter := v.Get("pattern")
		if complexFilter != nil {
			if filter, err := complexFilter.StringBytes(); err == nil {
				filterStr := string(filter)

				var invert bool;
				invertEntry := v.Get("invert")
				if invertEntry == nil {
					invert = false
				} else {
					invert = invertEntry.GetBool()
				}

				return &filterEntry{
					pattern: &filterStr,
					invert:  invert,
				}
			}
		}

		if filter, err := v.StringBytes(); err == nil {
			filterStr := string(filter)
			return &filterEntry{
				pattern: &filterStr,
				invert:  false,
			}
		}
	}

	return nil
}

func extractPodName(fullPodName string) string {
	re := regexp.MustCompile(`^(.+?)-[^%-]{10}-[^%-]{5}$|^(.+?)-\d+$|^(.+?)-[^%-]{5}$`)

	captures := re.FindStringSubmatch(fullPodName)

	if len(captures) == 0 {
		return fullPodName
	} else {
		for i := 1; i <= 3; i++ {
			if captures[i] != "" {
				return captures[i]
			}
		}

		return fullPodName
	}
}
