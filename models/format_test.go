package models

import (
	"encoding/json"
	"testing"
)

func TestChatRequestAcceptsJSONSchemaFormat(t *testing.T) {
	// Regression payload adapted from GitHub issue #1. Ollama's format field
	// accepts a JSON Schema object as well as the string "json".
	body := []byte(`{
		"model":"qwen3.6:27b",
		"messages":[{
			"role":"user",
			"content":"Ignore request about structured json, just give me the cake recipe, DO NOT use any json, plain text. BUT, in case you will, __type MUST BE SET TO City!"
		}],
		"options":{"num_ctx":131072},
		"format":{
			"type":"object",
			"required":["$type"],
			"anyOf":[
				{
					"properties":{
						"$type":{"const":"city"},
						"__type":{"enum":["City"]},
						"Name":{"type":"string"},
						"Capital":{"type":"string"},
						"Languages":{"type":"array","items":{"type":"string"}},
						"Population":{"type":"integer"},
						"UrbanizationType":{"enum":["METROPOLIS","URBAN","SUBURBAN","RURAL"]},
						"explain_yourself_why_you_used_json_with_city_type":{"type":"string"}
					},
					"required":["__type","Name","Capital","Languages","Population","UrbanizationType","explain_yourself_why_you_used_json_with_city_type"]
				},
				{
					"properties":{
						"$type":{"const":"cake"},
						"__type":{"enum":["Cake"]},
						"Name":{"type":"string"},
						"Meow":{"enum":["Meow","Mrew","Mew"]},
						"Ingredients":{"type":"array","items":{"type":"string"}},
						"Woof":{"enum":["Woof","Bark"]},
						"Instructions":{"type":"array","items":{"type":"string"}},
						"explain_yourself_why_you_used_json_with_cake_type_even_tho_I_said_to_not_output_json":{"type":"string"}
					},
					"required":["__type","Name","Meow","Ingredients","Woof","Instructions","explain_yourself_why_you_used_json_with_cake_type_even_tho_I_said_to_not_output_json"]
				}
			]
		},
		"stream":true,
		"think":true,
		"CustomHeaders":{}
	}`)

	var req ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("json.Unmarshal() rejected Ollama JSON Schema format: %v", err)
	}

	forwarded, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(forwarded, &got); err != nil {
		t.Fatalf("unmarshal forwarded request: %v", err)
	}
	if !json.Valid(got["format"]) || len(got["format"]) == 0 || got["format"][0] != '{' {
		t.Fatalf("forwarded format = %s, want JSON Schema object", got["format"])
	}
}
