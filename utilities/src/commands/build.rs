use std::fs;
use std::path::Path;
use std::process::{self, Command};

pub fn run(output_name: Option<&str>) {
    let name = output_name.unwrap_or("game");
    let build_dir = Path::new("build");

    println!("Building game (without editor)...");

    // Create build directory
    if !build_dir.exists() {
        if let Err(e) = fs::create_dir_all(build_dir) {
            eprintln!("Error creating build directory: {e}");
            process::exit(1);
        }
    }

    // On macOS, create an .app bundle
    #[cfg(target_os = "macos")]
    {
        build_macos_app(name, build_dir);
    }

    // Non-macOS: just build the binary
    #[cfg(not(target_os = "macos"))]
    {
        build_binary(name, build_dir);
    }
}

#[cfg(target_os = "macos")]
fn build_macos_app(name: &str, build_dir: &Path) {
    let app_name = format!("{}.app", name);
    let app_path = build_dir.join(&app_name);
    let contents_path = app_path.join("Contents");
    let macos_path = contents_path.join("MacOS");
    let resources_path = contents_path.join("Resources");

    // Create bundle structure
    fs::create_dir_all(&macos_path).expect("Failed to create MacOS dir");
    fs::create_dir_all(&resources_path).expect("Failed to create Resources dir");

    // Build the Go binary into the bundle
    let binary_path = macos_path.join(name);
    run_go_build(&binary_path);

    // Copy assets into Resources
    copy_assets(&resources_path.join("assets"));

    // Create Info.plist
    let plist = format!(
        r#"<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleExecutable</key>
    <string>{name}</string>
    <key>CFBundleIdentifier</key>
    <string>com.mirgo.{name}</string>
    <key>CFBundleName</key>
    <string>{name}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleVersion</key>
    <string>1.0</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>"#
    );

    fs::write(contents_path.join("Info.plist"), plist).expect("Failed to write Info.plist");

    println!("\nBuild complete!");
    println!("Created: build/{app_name}");
    println!("Double-click to run or drag to Applications!");
}

#[cfg(not(target_os = "macos"))]
fn build_binary(name: &str, build_dir: &Path) {
    let output_path = build_dir.join(name);
    run_go_build(&output_path);
    copy_assets(&build_dir.join("assets"));

    println!("\nBuild complete!");
    println!("Run with: cd build && ./{name}");
}

fn run_go_build(output_path: &Path) {
    let status = Command::new("go")
        .args([
            "build",
            "-tags",
            "game",
            "-o",
            output_path.to_str().unwrap(),
            "./cmd/test3d",
        ])
        .status();

    match status {
        Ok(s) if s.success() => {
            println!("Built binary: {}", output_path.display());
        }
        Ok(s) => {
            eprintln!(
                "Go build failed with exit code: {}",
                s.code().unwrap_or(-1)
            );
            process::exit(1);
        }
        Err(e) => {
            eprintln!("Failed to run go build: {e}");
            process::exit(1);
        }
    }
}

fn copy_assets(dst: &Path) {
    let assets_src = Path::new("assets");

    if assets_src.exists() {
        println!("Copying assets...");
        if let Err(e) = copy_dir_recursive(assets_src, dst) {
            eprintln!("Error copying assets: {e}");
            process::exit(1);
        }
    }
}

fn copy_dir_recursive(src: &Path, dst: &Path) -> std::io::Result<()> {
    if !dst.exists() {
        fs::create_dir_all(dst)?;
    }

    for entry in fs::read_dir(src)? {
        let entry = entry?;
        let src_path = entry.path();
        let dst_path = dst.join(entry.file_name());

        if src_path.is_dir() {
            copy_dir_recursive(&src_path, &dst_path)?;
        } else {
            fs::copy(&src_path, &dst_path)?;
        }
    }

    Ok(())
}
