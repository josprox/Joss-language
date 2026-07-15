package pluginpkg

import "github.com/jossecurity/joss/pkg/parser"

const SymbolSchemaVersion = 1

type SymbolIndex struct {
	Schema    int              `json:"schema"`
	Package   string           `json:"package"`
	Version   string           `json:"version"`
	Classes   []SymbolClass    `json:"classes,omitempty"`
	Functions []SymbolCallable `json:"functions,omitempty"`
}

type SymbolClass struct {
	Name       string           `json:"name"`
	SuperClass string           `json:"super_class,omitempty"`
	Methods    []SymbolCallable `json:"methods,omitempty"`
	Properties []string         `json:"properties,omitempty"`
}

type SymbolCallable struct {
	Name       string            `json:"name"`
	Parameters []SymbolParameter `json:"parameters,omitempty"`
}

type SymbolParameter struct {
	Name string `json:"name"`
	Type string `json:"type,omitempty"`
}

func BuildSymbolIndex(program *parser.Program, packageName, version string) SymbolIndex {
	index := SymbolIndex{Schema: SymbolSchemaVersion, Package: packageName, Version: version}
	if program == nil {
		return index
	}
	for _, statement := range program.Statements {
		switch node := statement.(type) {
		case *parser.ClassStatement:
			class := SymbolClass{Name: identifierValue(node.Name)}
			if node.SuperClass != nil {
				class.SuperClass = identifierValue(node.SuperClass)
			}
			if node.Body != nil {
				for _, member := range node.Body.Statements {
					switch value := member.(type) {
					case *parser.MethodStatement:
						class.Methods = append(class.Methods, callableSymbol(value.Name, value.Parameters))
					case *parser.LetStatement:
						if value.Name != nil {
							class.Properties = append(class.Properties, identifierValue(value.Name))
						}
					case *parser.MultiLetStatement:
						for _, declaration := range value.Declarations {
							class.Properties = append(class.Properties, identifierValue(declaration.Name))
						}
					}
				}
			}
			index.Classes = append(index.Classes, class)
		case *parser.MethodStatement:
			index.Functions = append(index.Functions, callableSymbol(node.Name, node.Parameters))
		}
	}
	return index
}

func callableSymbol(name *parser.Identifier, parameters []*parser.Parameter) SymbolCallable {
	callable := SymbolCallable{Name: identifierValue(name)}
	for _, parameter := range parameters {
		if parameter == nil || parameter.Name == nil {
			continue
		}
		typeName := parameter.Type.Literal
		if parameter.Type.Type == parser.VAR {
			typeName = ""
		}
		callable.Parameters = append(callable.Parameters, SymbolParameter{
			Name: identifierValue(parameter.Name),
			Type: typeName,
		})
	}
	return callable
}

func identifierValue(identifier *parser.Identifier) string {
	if identifier == nil {
		return ""
	}
	return identifier.Value
}
