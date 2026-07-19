package plan

import (
	"reflect"
	"strings"
	"testing"
)

func TestPreviewAndUndoTypesCannotStorePageContent(t *testing.T) {
	for name, value := range map[string]any{
		"preview": PreviewRecord{},
		"undo":    UndoSnapshot{},
	} {
		assertNoPageContentField(t, name, reflect.TypeOf(value), map[reflect.Type]bool{})
	}
}

func assertNoPageContentField(t *testing.T, path string, value reflect.Type, seen map[reflect.Type]bool) {
	t.Helper()
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Slice || value.Kind() == reflect.Map {
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct || seen[value] {
		return
	}
	seen[value] = true
	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		jsonName := strings.Split(field.Tag.Get("json"), ",")[0]
		switch strings.ToLower(jsonName) {
		case "text", "body", "pagecontent", "html", "content":
			t.Fatalf("%s.%s can persist page content", path, field.Name)
		}
		assertNoPageContentField(t, path+"."+field.Name, field.Type, seen)
	}
}
