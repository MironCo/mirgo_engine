use std::env;
use std::fs;
use std::path::Path;
use std::process;

use serde_json::Value;

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

fn main() {
    let args: Vec<String> = env::args().collect();

    if args.len() < 2 {
        eprintln!("Usage: mirgo-utils <command>");
        eprintln!("Commands: newscript");
        process::exit(1);
    }

    match args[1].as_str() {
        "newscript" => {
            if args.len() < 3 {
                eprintln!("Usage: mirgo-utils newscript <ScriptName>");
                eprintln!("Example: mirgo-utils newscript EnemyChaser");
                process::exit(1);
            }
            new_script(&args[2]);
        }
        "flipnormals" => {
            if args.len() < 3 {
                eprintln!("Usage: mirgo-utils flipnormals <path/to/file.gltf>");
                eprintln!("Example: mirgo-utils flipnormals assets/models/duck.gltf");
                process::exit(1);
            }
            flip_normals(&args[2]);
        }
        other => {
            eprintln!("Unknown command: {other}");
            eprintln!("Commands: newscript, flipnormals");
            process::exit(1);
        }
    }
}

fn new_script(name: &str) {
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

fn flip_normals(gltf_path: &str) {
    let mut path = Path::new(gltf_path).to_path_buf();
    if !path.exists() {
        eprintln!("Error: file not found: {gltf_path}");
        process::exit(1);
    }

    // If path is a directory, look for a .gltf file inside
    if path.is_dir() {
        let mut found = None;
        if let Ok(entries) = fs::read_dir(&path) {
            for entry in entries.flatten() {
                if entry.path().extension().map(|e| e == "gltf").unwrap_or(false) {
                    found = Some(entry.path());
                    break;
                }
            }
        }
        match found {
            Some(p) => {
                println!("Found GLTF file: {}", p.display());
                path = p;
            }
            None => {
                eprintln!("Error: no .gltf file found in directory {gltf_path}");
                process::exit(1);
            }
        }
    }

    let content = match fs::read_to_string(&path) {
        Ok(c) => c,
        Err(e) => {
            eprintln!("Error reading file: {e}");
            process::exit(1);
        }
    };

    let gltf: Value = match serde_json::from_str(&content) {
        Ok(v) => v,
        Err(e) => {
            eprintln!("Error parsing GLTF JSON: {e}");
            process::exit(1);
        }
    };

    // Find the .bin file path (same directory, referenced in buffers)
    let bin_path = if let Some(buffers) = gltf.get("buffers").and_then(|b| b.as_array()) {
        if let Some(uri) = buffers.first().and_then(|b| b.get("uri")).and_then(|u| u.as_str()) {
            path.parent().unwrap_or(Path::new(".")).join(uri)
        } else {
            eprintln!("Error: no buffer URI found in GLTF");
            process::exit(1);
        }
    } else {
        eprintln!("Error: no buffers found in GLTF");
        process::exit(1);
    };

    let mut bin_data = match fs::read(&bin_path) {
        Ok(d) => d,
        Err(e) => {
            eprintln!("Error reading binary file {}: {e}", bin_path.display());
            process::exit(1);
        }
    };

    // Find NORMAL accessors and flip them
    let mut normals_flipped = 0;

    if let Some(meshes) = gltf.get("meshes").and_then(|m| m.as_array()) {
        for mesh in meshes {
            if let Some(primitives) = mesh.get("primitives").and_then(|p| p.as_array()) {
                for primitive in primitives {
                    if let Some(attributes) = primitive.get("attributes").and_then(|a| a.as_object()) {
                        if let Some(normal_idx) = attributes.get("NORMAL").and_then(|n| n.as_u64()) {
                            normals_flipped += flip_accessor_normals(
                                &gltf,
                                normal_idx as usize,
                                &mut bin_data,
                            );
                        }
                    }
                }
            }
        }
    }

    if normals_flipped == 0 {
        println!("No normals found to flip in {gltf_path}");
        return;
    }

    // Write back the modified binary data
    if let Err(e) = fs::write(&bin_path, &bin_data) {
        eprintln!("Error writing binary file: {e}");
        process::exit(1);
    }

    println!("Flipped {normals_flipped} normal vectors in {}", bin_path.display());
}

fn flip_accessor_normals(gltf: &Value, accessor_idx: usize, bin_data: &mut [u8]) -> usize {
    let accessors = match gltf.get("accessors").and_then(|a| a.as_array()) {
        Some(a) => a,
        None => return 0,
    };

    let accessor = match accessors.get(accessor_idx) {
        Some(a) => a,
        None => return 0,
    };

    let buffer_view_idx = match accessor.get("bufferView").and_then(|b| b.as_u64()) {
        Some(idx) => idx as usize,
        None => return 0,
    };

    let count = match accessor.get("count").and_then(|c| c.as_u64()) {
        Some(c) => c as usize,
        None => return 0,
    };

    let component_type = accessor.get("componentType").and_then(|c| c.as_u64()).unwrap_or(0);
    let accessor_type = accessor.get("type").and_then(|t| t.as_str()).unwrap_or("");

    // We expect VEC3 of floats (componentType 5126)
    if accessor_type != "VEC3" || component_type != 5126 {
        eprintln!("Warning: NORMAL accessor is not VEC3 float, skipping");
        return 0;
    }

    let buffer_views = match gltf.get("bufferViews").and_then(|b| b.as_array()) {
        Some(bv) => bv,
        None => return 0,
    };

    let buffer_view = match buffer_views.get(buffer_view_idx) {
        Some(bv) => bv,
        None => return 0,
    };

    let byte_offset = buffer_view.get("byteOffset").and_then(|o| o.as_u64()).unwrap_or(0) as usize;
    let accessor_offset = accessor.get("byteOffset").and_then(|o| o.as_u64()).unwrap_or(0) as usize;
    let byte_stride = buffer_view.get("byteStride").and_then(|s| s.as_u64()).unwrap_or(12) as usize;

    let start = byte_offset + accessor_offset;

    for i in 0..count {
        let offset = start + i * byte_stride;

        // Read and negate each component (x, y, z)
        for j in 0..3 {
            let float_offset = offset + j * 4;
            if float_offset + 4 > bin_data.len() {
                eprintln!("Warning: buffer overflow at normal {i}, component {j}");
                continue;
            }

            let bytes: [u8; 4] = [
                bin_data[float_offset],
                bin_data[float_offset + 1],
                bin_data[float_offset + 2],
                bin_data[float_offset + 3],
            ];
            let value = f32::from_le_bytes(bytes);
            let negated = -value;
            let new_bytes = negated.to_le_bytes();

            bin_data[float_offset] = new_bytes[0];
            bin_data[float_offset + 1] = new_bytes[1];
            bin_data[float_offset + 2] = new_bytes[2];
            bin_data[float_offset + 3] = new_bytes[3];
        }
    }

    count
}
