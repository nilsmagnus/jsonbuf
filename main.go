package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bep/gr"
	"github.com/bep/gr/el"
	"github.com/bep/gr/evt"
	"io"
	"io/ioutil"
	"strings"
)

const (
	header = `syntax = "proto3";

package jsonbuf;

`
)

func main() {
	component := gr.New(new(rawValue))
	gr.RenderLoop(func() {
		component.Render("jsonbuf", gr.Props{})
	})
}

type rawValue struct {
	*gr.This
}

// Implements the StateInitializer interface.
func (c rawValue) GetInitialState() gr.State {
	fmt.Println("Get Initial State")
	return gr.State{
		"json": "{\"foo\":\"bar\"}",
	}
}

// Implements the Renderer interface.
func (c rawValue) Render() gr.Component {

	jsonString := c.State().String("json")
	protoString, err := toProtoBuf(strings.NewReader(jsonString))
	if err != nil {
		protoString = err.Error()
	}

	onUpdateJsonTextArea := func(newJson string) { c.SetState(gr.State{"json": newJson}) }

	elem := el.Div(
		gr.Style("", ""),
		el.Div(
			gr.Text("Json to protobuf :"),
			el.Break(),
			textArea(jsonString, onUpdateJsonTextArea),
			textArea(protoString, func(i string) {}),
		),
		el.Break(),
	)

	return elem
}
func textArea(contentString string, onNewValue func(newJson string)) gr.Modifier {

	return el.TextArea(
		gr.Prop("rows", "40"),
		gr.Style("width", "40%"),
		gr.Style("margin", "5px"),
		gr.Prop("value", contentString),
		evt.Change(func(e *gr.Event) { onNewValue(e.TargetValue().String()) }),
	)
}

// Implements the ShouldComponentUpdate interface.
func (c rawValue) ShouldComponentUpdate(next gr.Cops) bool {
	return c.State().HasChanged(next.State, "json")
}

//--------------------
func toProtoBuf(reader io.Reader) (string, error) {

	jsonMap, err := unmarshal(reader)

	if err != nil {
		return "", err
	}

	protodefs := toProtos("RootBuf", jsonMap)

	pretties := toPrettyStrings(protodefs)

	var strbuf bytes.Buffer

	strbuf.WriteString(header)
	for _, p := range pretties {
		strbuf.WriteString(p)
		strbuf.WriteString("\n\n")
	}

	return strbuf.String(), nil
}
func toPrettyStrings(strings map[string]map[string]string) []string {
	res := make([]string, 0)
	for messageName, message := range strings {
		res = append(res, toPrettyProtoString(messageName, message))
	}
	return res
}
func toPrettyProtoString(messageName string, strings map[string]string) string {
	var buffer bytes.Buffer

	buffer.WriteString(fmt.Sprintf("message %s {\n", messageName))

	for _, v := range strings {
		buffer.WriteString(v)
		buffer.WriteString("\n")
	}
	buffer.WriteString("}")

	return buffer.String()
}

func toProtos(messageName string, unknowns map[string]interface{}) map[string]map[string]string {
	res := make(map[string]map[string]string)

	res[messageName] = make(map[string]string)
	counter := 0
	for k, v := range unknowns {
		t, m := typeWithNameAndIndex(k, v, counter)
		if len(m) > 0 {
			nestedName := k
			res[messageName][nestedName] = fmt.Sprintf("\t%s %s = %d;", nestedName, k, counter)
			nestedMap := toProtos(nestedName, m)
			res = mergemaps(res, nestedMap)
		}
		if t != "" {
			res[messageName][k] = t
		}
		counter++
	}
	return res

}

func typeWithNameAndIndex(name string, dataType interface{}, count int) (string, map[string]interface{}) {
	nameit := func(t, n string, i int) (string, map[string]interface{}) {
		return fmt.Sprintf("\t%s %s = %d ;", t, n, i), make(map[string]interface{})
	}
	switch firstType := dataType.(type) {
	case string:
		return nameit("string ", name, count)
	case []interface{}:
		fmt.Println("repeated interface")
	default:
		switch innerType := firstType.(type) {
		case int:
		case int32:
			return nameit("int32", name, count)
		case int64:
			return nameit("int64", name, count)
		case float32:
			return nameit("float32", name, count)
		case float64:
			return nameit("float64", name, count)
		case map[string]interface{}:
			return "", dataType.(map[string]interface{})
		default:
			return nameit(fmt.Sprintf("%v", innerType), name, count)
		}
	}

	return "", make(map[string]interface{})

}

func mergemaps(map1, map2 map[string]map[string]string) map[string]map[string]string {
	result := make(map[string]map[string]string)

	for k, v := range map1 {
		result[k] = v
	}
	for k, v := range map2 {
		result[k] = v
	}
	return result
}

func unmarshal(reader io.Reader) (map[string]interface{}, error) {
	emptyResponse := make(map[string]interface{}, 0)

	content, readerr := ioutil.ReadAll(reader)
	if readerr != nil {
		return emptyResponse, readerr
	}

	var raw interface{}
	jerr := json.Unmarshal(content, &raw)

	if jerr != nil {
		return emptyResponse, jerr
	}

	jsonMap, ok := raw.(map[string]interface{})

	if !ok {
		return emptyResponse, fmt.Errorf("Could not cast raw to json-map.")
	}

	return jsonMap, nil
}
