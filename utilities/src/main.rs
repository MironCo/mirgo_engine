mod commands;

use std::env;
use std::process;

fn main() {
    let args: Vec<String> = env::args().collect();

    if args.len() < 2 {
        print_usage();
        process::exit(1);
    }

    match args[1].as_str() {
        "newscript" => {
            if args.len() < 3 {
                eprintln!("Usage: mirgo-utils newscript <ScriptName>");
                eprintln!("Example: mirgo-utils newscript EnemyChaser");
                process::exit(1);
            }
            commands::newscript::run(&args[2]);
        }
        "flipnormals" => {
            if args.len() < 3 {
                eprintln!("Usage: mirgo-utils flipnormals <path/to/file.gltf>");
                eprintln!("Example: mirgo-utils flipnormals assets/models/duck.gltf");
                process::exit(1);
            }
            commands::flipnormals::run(&args[2]);
        }
        "build" => {
            let output_name = args.get(2).map(|s| s.as_str());
            commands::build::run(output_name);
        }
        "help" | "--help" | "-h" => {
            print_usage();
        }
        other => {
            eprintln!("Unknown command: {other}");
            print_usage();
            process::exit(1);
        }
    }
}

fn print_usage() {
    eprintln!("mirgo-utils - Mirgo Engine utilities\n");
    eprintln!("Commands:");
    eprintln!("  newscript <Name>    Create a new Go script component");
    eprintln!("  flipnormals <path>  Flip normals in a GLTF model");
    eprintln!("  build [name]        Build game as macOS .app bundle");
    eprintln!("  help                Show this help message");
}
