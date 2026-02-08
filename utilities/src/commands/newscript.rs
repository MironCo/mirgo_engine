use std::fs;
use std::path::Path;
use std::process;

const SCRIPTS_DIR: &str = "internal/components/scripts";

const TEMPLATE: &str = r#"package scripts

import "test3d/internal/engine"

type {{NAME}} struct {
	engine.BaseComponent
	Speed float32
}

func (s *{{NAME}}) Update(deltaTime float32) {
	g := s.GetGameObject()
	if g == nil {
		return
	}
	// TODO: implement behavior
}

func init() {
	engine.RegisterScript("{{NAME}}", {{LOWER}}Factory, {{LOWER}}Serializer)
}

func {{LOWER}}Factory(props map[string]any) engine.Component {
	speed := float32(1)
	if v, ok := props["speed"].(float64); ok {
		speed = float32(v)
	}
	return &{{NAME}}{Speed: speed}
}

func {{LOWER}}Serializer(c engine.Component) map[string]any {
	s, ok := c.(*{{NAME}})
	if !ok {
		return nil
	}
	return map[string]any{
		"speed": s.Speed,
	}
}
"#;

pub fn run(name: &str) {
    if name.is_empty() || !name.chars().next().unwrap().is_uppercase() {
        eprintln!("Error: script name must start with an uppercase letter");
        process::exit(1);
    }

    let lower = {
        let mut chars = name.chars();
        let first = chars.next().unwrap().to_lowercase().to_string();
        format!("{first}{}", chars.as_str())
    };

    let filename = format!("{}.go", to_snake_case(name));
    let out_path = Path::new(SCRIPTS_DIR).join(&filename);

    if out_path.exists() {
        eprintln!("Error: {} already exists", out_path.display());
        process::exit(1);
    }

    let content = TEMPLATE
        .replace("{{NAME}}", name)
        .replace("{{LOWER}}", &lower);

    if let Err(e) = fs::write(&out_path, content) {
        eprintln!("Error writing file: {e}");
        process::exit(1);
    }

    println!("Created {}", out_path.display());
    println!("Script \"{name}\" registered. Add it to a scene object:\n");
    println!("  {{");
    println!("    \"type\": \"Script\",");
    println!("    \"name\": \"{name}\",");
    println!("    \"props\": {{ \"speed\": 1.0 }}");
    println!("  }}");
}

fn to_snake_case(s: &str) -> String {
    let mut result = String::new();
    for (i, c) in s.chars().enumerate() {
        if c.is_uppercase() && i > 0 {
            result.push('_');
        }
        result.push(c.to_lowercase().next().unwrap());
    }
    result
}
