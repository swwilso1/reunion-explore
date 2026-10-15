package web

import (
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/kedoco/reunion-explore/model"
)

var (
	openAPICache     map[string]any
	openAPICacheOnce sync.Once
)

func (s *Server) handleOpenAPI(w http.ResponseWriter, r *http.Request) {
	openAPICacheOnce.Do(func() {
		openAPICache = buildOpenAPISpec()
	})
	writeJSON(w, http.StatusOK, openAPICache)
}

func buildOpenAPISpec() map[string]any {
	schemas := make(map[string]any)

	// Generate schemas from response types
	schemaFromType(reflect.TypeOf(PersonRef{}), schemas)
	schemaFromType(reflect.TypeOf(PersonDetail{}), schemas)
	schemaFromType(reflect.TypeOf(ResolvedEvent{}), schemas)
	schemaFromType(reflect.TypeOf(SourceCitationDisplay{}), schemas)
	schemaFromType(reflect.TypeOf(NoteDisplay{}), schemas)
	schemaFromType(reflect.TypeOf(TreeEntryRef{}), schemas)
	schemaFromType(reflect.TypeOf(PlaceRef{}), schemas)
	schemaFromType(reflect.TypeOf(FamilyRef{}), schemas)
	schemaFromType(reflect.TypeOf(FamilyDetail{}), schemas)
	schemaFromType(reflect.TypeOf(StatsResponse{}), schemas)
	schemaFromType(reflect.TypeOf(SummaryResponse{}), schemas)
	schemaFromType(reflect.TypeOf(PaginatedResponse{}), schemas)
	schemaFromType(reflect.TypeOf(model.Place{}), schemas)
	schemaFromType(reflect.TypeOf(model.EventDefinition{}), schemas)
	schemaFromType(reflect.TypeOf(model.Source{}), schemas)
	schemaFromType(reflect.TypeOf(model.Note{}), schemas)

	return map[string]any{
		"openapi": "3.1.0",
		"info": map[string]any{
			"title":       "Reunion Explorer API",
			"description": "REST API for browsing Reunion 13/14 family file data",
			"version":     "1.0.0",
		},
		"paths": buildPaths(),
		"components": map[string]any{
			"schemas": schemas,
		},
	}
}

func buildPaths() map[string]any {
	return map[string]any{
		"/api/stats": pathItem("get", "Get statistics", "StatsResponse"),
		"/api/persons": pathItemWithParams("get", "List persons", "PaginatedResponse",
			queryParam("surname", "string", "Filter by surname"),
			queryParam("q", "string", "Search query"),
			queryParam("page", "integer", "Page number"),
			queryParam("per_page", "integer", "Items per page"),
		),
		"/api/persons/{id}":             pathItemWithID("get", "Get person detail", "PersonDetail"),
		"/api/persons/{id}/families":    pathItemWithID("get", "Get person's families", "array:FamilyDetail"),
		"/api/persons/{id}/ancestors":   pathItemWithIDAndParams("get", "Get ancestors", "array:TreeEntryRef", queryParam("generations", "integer", "Max generations")),
		"/api/persons/{id}/descendants": pathItemWithIDAndParams("get", "Get descendants", "array:TreeEntryRef", queryParam("generations", "integer", "Max generations")),
		"/api/persons/{id}/treetops":    pathItemWithID("get", "Get treetops", "array:PersonRef"),
		"/api/persons/{id}/summary":     pathItemWithID("get", "Get person summary", "SummaryResponse"),
		"/api/families": pathItemWithParams("get", "List families", "PaginatedResponse",
			queryParam("page", "integer", "Page number"),
			queryParam("per_page", "integer", "Items per page"),
		),
		"/api/families/{id}":     pathItemWithID("get", "Get family detail", "FamilyDetail"),
		"/api/places":            pathItem("get", "List places", "array:Place"),
		"/api/places/{id}":       pathItemWithID("get", "Get place", "Place"),
		"/api/places/{id}/persons": pathItemWithID("get", "Get persons at place", "array:PersonRef"),
		"/api/events":            pathItem("get", "List event types", "array:EventDefinition"),
		"/api/events/{id}":       pathItemWithID("get", "Get event type", "EventDefinition"),
		"/api/events/{id}/persons": pathItemWithID("get", "Get persons with event type", "array:PersonRef"),
		"/api/sources":              pathItem("get", "List sources", "array:Source"),
		"/api/sources/{id}":         pathItemWithID("get", "Get source", "Source"),
		"/api/sources/{id}/persons": pathItemWithID("get", "Get persons citing source", "array:PersonRef"),
		"/api/notes": pathItemWithParams("get", "List notes", "array:Note",
			queryParam("person_id", "integer", "Filter by person ID"),
		),
		"/api/notes/{id}": pathItemWithID("get", "Get note", "Note"),
		"/api/search": pathItemWithParams("get", "Search persons", "array:PersonRef",
			queryParam("q", "string", "Search query"),
		),
	}
}

func pathItem(method, summary, schemaRef string) map[string]any {
	return map[string]any{
		method: map[string]any{
			"summary":   summary,
			"responses": okResponse(schemaRef),
		},
	}
}

func pathItemWithParams(method, summary, schemaRef string, params ...map[string]any) map[string]any {
	return map[string]any{
		method: map[string]any{
			"summary":    summary,
			"parameters": params,
			"responses":  okResponse(schemaRef),
		},
	}
}

func pathItemWithID(method, summary, schemaRef string) map[string]any {
	return map[string]any{
		method: map[string]any{
			"summary":    summary,
			"parameters": []map[string]any{idParam()},
			"responses":  okResponse(schemaRef),
		},
	}
}

func pathItemWithIDAndParams(method, summary, schemaRef string, params ...map[string]any) map[string]any {
	allParams := []map[string]any{idParam()}
	allParams = append(allParams, params...)
	return map[string]any{
		method: map[string]any{
			"summary":    summary,
			"parameters": allParams,
			"responses":  okResponse(schemaRef),
		},
	}
}

func idParam() map[string]any {
	return map[string]any{
		"name":     "id",
		"in":       "path",
		"required": true,
		"schema":   map[string]any{"type": "integer"},
	}
}

func queryParam(name, typ, desc string) map[string]any {
	return map[string]any{
		"name":        name,
		"in":          "query",
		"description": desc,
		"schema":      map[string]any{"type": typ},
	}
}

func okResponse(schemaRef string) map[string]any {
	var schema map[string]any
	if strings.HasPrefix(schemaRef, "array:") {
		itemRef := strings.TrimPrefix(schemaRef, "array:")
		schema = map[string]any{
			"type":  "array",
			"items": map[string]any{"$ref": "#/components/schemas/" + itemRef},
		}
	} else {
		schema = map[string]any{"$ref": "#/components/schemas/" + schemaRef}
	}
	return map[string]any{
		"200": map[string]any{
			"description": "OK",
			"content": map[string]any{
				"application/json": map[string]any{
					"schema": schema,
				},
			},
		},
	}
}

// schemaFromType generates a JSON Schema from a Go struct type using reflect.
func schemaFromType(t reflect.Type, schemas map[string]any) map[string]any {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	name := t.Name()
	if name != "" {
		if _, exists := schemas[name]; exists {
			return map[string]any{"$ref": "#/components/schemas/" + name}
		}
	}

	if t.Kind() != reflect.Struct {
		return goTypeToJSONSchema(t, schemas)
	}

	// Reserve to prevent infinite recursion
	if name != "" {
		schemas[name] = nil
	}

	properties := make(map[string]any)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		tag := field.Tag.Get("json")
		if tag == "-" {
			continue
		}

		jsonName, opts := parseJSONTag(tag)
		if jsonName == "" {
			jsonName = field.Name
		}

		propSchema := goTypeToJSONSchema(field.Type, schemas)
		properties[jsonName] = propSchema

		if !opts.omitempty {
			required = append(required, jsonName)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}

	if name != "" {
		schemas[name] = schema
		return map[string]any{"$ref": "#/components/schemas/" + name}
	}
	return schema
}

func goTypeToJSONSchema(t reflect.Type, schemas map[string]any) map[string]any {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return map[string]any{"type": "string", "format": "byte"}
		}
		return map[string]any{
			"type":  "array",
			"items": goTypeToJSONSchema(t.Elem(), schemas),
		}
	case reflect.Map:
		return map[string]any{
			"type":                 "object",
			"additionalProperties": goTypeToJSONSchema(t.Elem(), schemas),
		}
	case reflect.Struct:
		return schemaFromType(t, schemas)
	case reflect.Interface:
		return map[string]any{}
	default:
		return map[string]any{"type": "string"}
	}
}

type jsonTagOpts struct {
	omitempty bool
}

func parseJSONTag(tag string) (string, jsonTagOpts) {
	if tag == "" {
		return "", jsonTagOpts{}
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	opts := jsonTagOpts{}
	for _, p := range parts[1:] {
		if p == "omitempty" {
			opts.omitempty = true
		}
	}
	return name, opts
}
